// Package ratelimit — Token Bucket 기반 identity-별 Rate Limiter.
//
// 동작: 각 identity (player_id 또는 IP) 마다 독립된 `*rate.Limiter` 를 유지.
// 장시간 미사용 identity 는 TTL sweep 으로 정리 → 메모리 누수 방지.
//
// 분산 제약: 단일 프로세스 기준. 다중 Pod 라면 각 Pod 가 독립 limiter 를 가지므로
// 전체 관점에서는 `Pod 수 × RPS` 가 실제 상한. 엄격한 분산 rate limit 이 필요하면
// Redis 기반 (`incr` + TTL) 으로 교체 가능하나 본 MVP 는 단순성을 위해 local.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter — identity 별 Token Bucket 집합.
type Limiter struct {
	mu sync.Mutex

	// identity → limiter 매핑.
	buckets map[string]*bucket

	rps       rate.Limit    // 초당 허용 토큰 (평균 RPS).
	burst     int           // 버킷 최대 크기 (버스트 허용량).
	idleTTL   time.Duration // 이 시간 넘게 미사용인 bucket 은 sweep.
	now       func() time.Time
	stopSweep chan struct{}
}

type bucket struct {
	l        *rate.Limiter
	lastSeen time.Time
}

// Option — 테스트 · 커스터마이즈용.
type Option func(*Limiter)

// WithClock — 시계 함수 주입.
func WithClock(fn func() time.Time) Option {
	return func(l *Limiter) { l.now = fn }
}

// WithIdleTTL — idle bucket sweep 기준 시간. 기본 10 분.
func WithIdleTTL(d time.Duration) Option {
	return func(l *Limiter) { l.idleTTL = d }
}

// New — rps · burst 로 Limiter 생성.
// rps<=0 이면 "무제한" (Allow 항상 true). burst<=0 이면 burst=1.
func New(rps float64, burst int, opts ...Option) *Limiter {
	if burst <= 0 {
		burst = 1
	}
	l := &Limiter{
		buckets:   map[string]*bucket{},
		rps:       rate.Limit(rps),
		burst:     burst,
		idleTTL:   10 * time.Minute,
		now:       time.Now,
		stopSweep: make(chan struct{}),
	}
	for _, o := range opts {
		o(l)
	}
	return l
}

// Allow — identity 에 토큰이 남았는지 확인하고 하나 소비.
// rps<=0 이면 항상 true (무제한 모드).
func (l *Limiter) Allow(identity string) bool {
	if l.rps <= 0 {
		return true
	}
	l.mu.Lock()
	b, ok := l.buckets[identity]
	if !ok {
		b = &bucket{l: rate.NewLimiter(l.rps, l.burst)}
		l.buckets[identity] = b
	}
	b.lastSeen = l.now()
	limiter := b.l
	l.mu.Unlock()
	return limiter.Allow()
}

// Sweep — idleTTL 넘은 bucket 삭제. 반환: 삭제된 개수.
// StartSweeper 가 주기적으로 호출. 테스트에서는 직접 호출 가능.
func (l *Limiter) Sweep() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := l.now().Add(-l.idleTTL)
	removed := 0
	for k, b := range l.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(l.buckets, k)
			removed++
		}
	}
	return removed
}

// Size — 현재 보유 중인 bucket 수 (테스트·관측용).
func (l *Limiter) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// StartSweeper — interval 주기로 Sweep 실행. Stop 으로 종료.
// ctx 취소를 지원하지 않는 건 간단함 우선 — 테스트에서는 Stop + 이후 Sweep 을 명시 호출.
func (l *Limiter) StartSweeper(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				l.Sweep()
			case <-l.stopSweep:
				return
			}
		}
	}()
}

// Stop — StartSweeper 로 띄운 고루틴 종료.
func (l *Limiter) Stop() {
	close(l.stopSweep)
}
