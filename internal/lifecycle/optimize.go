// Package lifecycle — 서버 전역 런타임 플래그·상태 (readiness, optimize 토글 등) 관리.
package lifecycle

import "sync/atomic"

// Optimize — 런타임 최적화 토글 (Naive ↔ Pooled 전환용).
// 락 없이 thread-safe — 핫패스에서 매 호출마다 Load 해도 부담 없음.
type Optimize struct {
	on atomic.Bool
}

// On — 현재 최적화 활성 여부.
func (o *Optimize) On() bool {
	return o.on.Load()
}

// Set — 활성/비활성 설정.
func (o *Optimize) Set(v bool) {
	o.on.Store(v)
}
