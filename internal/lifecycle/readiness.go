package lifecycle

import "sync/atomic"

// Readiness — K8s readiness gate 용 flag. atomic.Bool 래퍼.
// 서버 기동 직후엔 true (트래픽 수용 가능), SIGTERM 수신 시 SetDraining() 으로 false 전환 →
// /health/ready 가 503 응답 → K8s load balancer 가 pod 를 endpoint 에서 제거.
// 락 없이 thread-safe — 핫패스 (probe 마다 Load) 에 부담 없음.
type Readiness struct {
	ready atomic.Bool
}

// NewReadiness — 초기 상태 ready=true 로 생성.
// 서버 바인딩 완료 직후 핸들러에 주입되는 걸 전제로 함.
func NewReadiness() *Readiness {
	r := &Readiness{}
	r.ready.Store(true)
	return r
}

// Ready — 현재 트래픽 수용 가능 여부.
func (r *Readiness) Ready() bool {
	return r.ready.Load()
}

// SetDraining — drain 시작. 이후 Ready() == false.
// 멱등 — 여러 번 호출해도 문제 없음.
func (r *Readiness) SetDraining() {
	r.ready.Store(false)
}
