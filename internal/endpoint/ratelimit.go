package endpoint

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/kimsehoon/stagesync/internal/auth"
	"github.com/kimsehoon/stagesync/internal/ratelimit"
)

// RateLimit — Token Bucket 기반 미들웨어.
//
// identity 우선순위:
//  1. 인증된 player_id (ctx 의 auth.Claims)
//  2. `X-Forwarded-For` 첫 번째 IP (프록시 뒤 배치 시)
//  3. `X-Real-IP`
//  4. `r.RemoteAddr` (폴백)
//
// 초과 시 `429 Too Many Requests` + `Retry-After: 1` 헤더 + JSON 바디.
// limiter 가 nil 이면 미들웨어는 no-op.
func RateLimit(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			id := identityOf(r)
			if !limiter.Allow(id) {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"code":    "RATE_LIMITED",
					"message": "too many requests — retry later",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// identityOf — 요청에서 rate-limit 식별자 추출.
// 인증 player_id → 프록시 헤더 → RemoteAddr 순.
func identityOf(r *http.Request) string {
	if player, ok := auth.PlayerIDFrom(r.Context()); ok && player != "" {
		return "player:" + player
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF 는 쉼표 구분의 IP 리스트 — 첫 번째가 원 클라이언트.
		if i := strings.Index(xff, ","); i > 0 {
			return "ip:" + strings.TrimSpace(xff[:i])
		}
		return "ip:" + strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return "ip:" + xri
	}
	// RemoteAddr 는 "host:port" 형태 — host 부분만.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return "ip:" + host
}
