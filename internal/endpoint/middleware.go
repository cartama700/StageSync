package endpoint

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// loggerCtxKey — 요청 스코프 slog logger 를 ctx 에 저장할 때 쓰는 키.
// 핸들러·서비스 레이어는 LoggerFrom(ctx) 으로 꺼내 쓰면 request_id 가 자동으로 붙음.
type loggerCtxKey struct{}

// RequestLogger — chi middleware.RequestID 뒤에 붙여야 함.
// request_id 를 포함한 slog logger 를 ctx 에 주입 + 요청 종료 시 access log 1 줄 기록.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			l := base.With(
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
			)
			ctx := context.WithValue(r.Context(), loggerCtxKey{}, l)

			// status code 캡처용 wrapper.
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				l.Info("request",
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
				)
			}()

			next.ServeHTTP(ww, r.WithContext(ctx))
		})
	}
}

// LoggerFrom — ctx 에 주입된 request-scoped slog logger 반환.
// 미들웨어 미장착 시 slog.Default() 로 fallback (테스트 편의).
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// RequestMetrics — HTTP 요청 지연을 Prometheus HistogramVec 에 기록하는 미들웨어.
// 레이블:
//   - method: GET/POST/... 원본 그대로
//   - path: chi 의 RoutePattern — IDs 는 `{id}` 로 유지 (high cardinality 방지)
//   - status: HTTP status code (숫자 문자열)
//
// 주의: chi 라우팅이 끝나야 RoutePattern 이 채워지므로 `next.ServeHTTP` **이후** 에 읽어야 함.
// 미매칭 경로 (404) 는 "(unknown)" 으로 집계 — 스캔·크롤링 노이즈 제거.
func RequestMetrics(hist *prometheus.HistogramVec) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			pattern := chi.RouteContext(r.Context()).RoutePattern()
			if pattern == "" {
				pattern = "(unknown)"
			}
			hist.WithLabelValues(
				r.Method,
				pattern,
				strconv.Itoa(ww.Status()),
			).Observe(time.Since(start).Seconds())
		})
	}
}
