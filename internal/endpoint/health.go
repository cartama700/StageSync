package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/lifecycle"
)

// HealthHandler — K8s liveness/readiness probe 응답.
// State 가 주입되어 있으면 drain 중 (Ready()==false) 에는 503 응답 →
// K8s load balancer 가 pod 를 endpoint 에서 제외.
type HealthHandler struct {
	// State — nil 이면 Ready 는 항상 200 (테스트/기동 초기 편의).
	// 필드명은 State 로 고정 — 메서드 Ready 와 충돌 회피.
	State *lifecycle.Readiness
}

// Mount — /health/* 라우트 등록.
func (h *HealthHandler) Mount(r chi.Router) {
	r.Get("/health/live", h.Live)
	r.Get("/health/ready", h.Ready)
}

// Live — /health/live. 항상 200 (프로세스 살아있음).
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Ready — /health/ready.
// State 가 없거나 Ready()==true 면 200.
// drain 시작 (SetDraining 호출) 이후엔 503 + {"ready": false} JSON.
func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	if h.State != nil && !h.State.Ready() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		// 단순 payload — encoding/json 거치지 않고 고정 문자열로.
		_, _ = w.Write([]byte(`{"ready":false}`))
		return
	}
	w.WriteHeader(http.StatusOK)
}
