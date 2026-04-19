package endpoint

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/apperror"
	"github.com/kimsehoon/stagesync/internal/auth"
)

// AuthHandler — `/api/auth/login` 엔드포인트.
//
// ⚠️ **개발 · 데모용**: 현재는 `player_id` 하나만 받아 JWT 를 발급. 실 프로덕션이라면:
//   - 패스워드 · OAuth provider · 디바이스 토큰 등 **실제 자격증명** 검증 필수
//   - 이 핸들러가 externally-hosted IdP (Auth0 · Cognito · 자체 SSO) 로 대체됨
//   - 또는 클라이언트가 IdP 에서 직접 받은 토큰을 본 서버가 **검증만** 수행
type AuthHandler struct {
	Issuer *auth.Issuer
}

// Mount — 로그인 엔드포인트 등록.
func (h *AuthHandler) Mount(r chi.Router) {
	r.Post("/api/auth/login", h.Login)
}

type loginReq struct {
	Player string `json:"player" validate:"required,min=1,max=64"`
}

type loginResp struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	PlayerID  string    `json:"player_id"`
}

// Login — `POST /api/auth/login`. body `{"player": "p1"}` → JWT.
//
// 설계 메모: 응답은 `{"token", "expires_at", "player_id"}` 로 최소. refresh token 은 현 스코프 밖.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.Issuer == nil {
		apperror.WriteJSON(w, apperror.Internal("auth not configured",
			errors.New("AUTH_SECRET is empty — server started in auth-disabled mode")))
		return
	}

	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid body", []apperror.FieldError{
			{Field: "body", Tag: "json", Message: err.Error()},
		}))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}

	token, exp, err := h.Issuer.Issue(req.Player)
	if err != nil {
		apperror.WriteJSON(w, apperror.Internal("issue token", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, loginResp{
		Token:     token,
		ExpiresAt: exp,
		PlayerID:  req.Player,
	})
}

// RequireAuth — JWT 검증 미들웨어.
//
// 동작:
//   - `validator == nil` (AUTH_SECRET 미설정) → **no-op, pass-through** (개발 편의).
//     프로덕션 배포 시엔 반드시 AUTH_SECRET 설정 → validator 활성화.
//   - Bearer 토큰 없음 · 잘못됨 · 만료 → `401 Unauthorized`.
//   - 성공 → `Claims` 를 `ctx` 에 주입 → 하위 핸들러는 `auth.PlayerIDFrom(ctx)` 로 취득.
//
// 헤더: `Authorization: Bearer <token>`.
func RequireAuth(validator *auth.Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validator == nil {
				// 개발 모드: auth 비활성. 그대로 통과.
				next.ServeHTTP(w, r)
				return
			}

			token, ok := bearerTokenFrom(r)
			if !ok {
				writeUnauthorized(w, "missing or malformed Authorization header")
				return
			}

			claims, err := validator.Validate(token)
			if err != nil {
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := auth.InjectClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerTokenFrom — `Authorization: Bearer <token>` 에서 token 추출.
// 대소문자 관용 (RFC 7235 준수).
func bearerTokenFrom(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(h[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// writeUnauthorized — 401 + `{"code":"unauthorized","message":"..."}` JSON.
func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="stagesync"`)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	// apperror 의 코드 세트에 unauthorized 가 없으므로 직접 inline 응답.
	// 향후 apperror 에 CodeUnauthorized 를 추가하면 그걸로 통일.
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "UNAUTHORIZED",
		"message": msg,
	})
}
