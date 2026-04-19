package endpoint

import (
	"bytes"
	"context"
	"net/http"

	"github.com/kimsehoon/stagesync/internal/auth"
	"github.com/kimsehoon/stagesync/internal/idempotency"
)

// Idempotency — `Idempotency-Key` 헤더 기반 중복 요청 캐싱 미들웨어.
//
// 동작:
//   - `Idempotency-Key` 헤더 없음 → pass-through (일반 처리).
//   - GET 요청 → pass-through (write 메서드 대상).
//   - Store 히트 → 캐시된 응답 리플레이 + `Idempotency-Replayed: true` 헤더.
//   - Store 미스 → 핸들러 실행 → 응답 캡처 → Store 에 저장 (best-effort).
//
// 스코프: authenticated player_id 가 있으면 `<player>:<key>`, 없으면 `anon:<key>`.
// 다른 유저가 같은 idempotency-key 를 써도 충돌하지 않도록.
//
// store 가 nil 이면 미들웨어는 no-op (dev 모드).
func Idempotency(store idempotency.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if key == "" || r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			scoped := scopedIdempotencyKey(ctx, key)

			// Cache lookup.
			entry, hit, err := store.Get(ctx, scoped)
			if err != nil {
				LoggerFrom(ctx).Warn("idempotency store get failed", "err", err, "key", scoped)
				// 저장소 장애는 요청을 차단하지 않음 — 정상 처리 경로로 진행.
			} else if hit {
				w.Header().Set("Idempotency-Replayed", "true")
				w.WriteHeader(entry.Status)
				_, _ = w.Write(entry.Body)
				return
			}

			// Capture response.
			rec := &recordedWriter{ResponseWriter: w, body: &bytes.Buffer{}, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Store (best-effort — 실패해도 클라이언트에는 정상 응답이 이미 나갔음).
			if err := store.Set(ctx, scoped, idempotency.Entry{
				Status: rec.status,
				Body:   rec.body.Bytes(),
			}); err != nil {
				LoggerFrom(ctx).Warn("idempotency store set failed", "err", err, "key", scoped)
			}
		})
	}
}

// scopedIdempotencyKey — player 별 namespace 적용.
func scopedIdempotencyKey(ctx context.Context, key string) string {
	if player, ok := auth.PlayerIDFrom(ctx); ok && player != "" {
		return player + ":" + key
	}
	return "anon:" + key
}

// recordedWriter — ResponseWriter 를 감싸 status + body 를 버퍼에 복제.
// 핸들러 응답 작성 후 body 를 Idempotency Store 에 저장하기 위함.
type recordedWriter struct {
	http.ResponseWriter
	body   *bytes.Buffer
	status int
	wrote  bool
}

func (w *recordedWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.status = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *recordedWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		// Implicit 200 — net/http 기본 동작에 맞춤.
		w.WriteHeader(http.StatusOK)
	}
	// 실 응답과 버퍼에 동시 기록.
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
