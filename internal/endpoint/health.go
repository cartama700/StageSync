package endpoint

import "net/http"

// HealthHandler — K8s liveness/readiness probe 응답.
// Phase 14 에서 Readiness gate (atomic.Bool) 가 Ready 에 추가되어 drain 중 503 응답.
type HealthHandler struct {
	// Phase 14: Ready *lifecycle.Readiness
}

// Live — /health/live. 항상 200 (프로세스 살아있음).
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Ready — /health/ready. Phase 14 까지는 항상 200 (트래픽 수용 가능).
// 그 이후엔 drain 시작 시 503 으로 바뀌어 K8s load balancer 가 pod 빼감.
func (h *HealthHandler) Ready(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
