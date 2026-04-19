package ratelimit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/ratelimit"
)

func TestLimiter_AllowsBurst(t *testing.T) {
	t.Parallel()
	// 1 RPS, burst 3 → 버스트 3 개까지 연속 성공.
	l := ratelimit.New(1, 3)
	for i := 0; i < 3; i++ {
		require.True(t, l.Allow("p1"), "burst[%d] 는 허용되어야 함", i)
	}
	// 4 번째는 거절.
	require.False(t, l.Allow("p1"))
}

func TestLimiter_IndependentIdentities(t *testing.T) {
	t.Parallel()
	// identity 별로 버킷 독립 — 한 쪽이 소진돼도 다른 쪽은 영향 없음.
	l := ratelimit.New(1, 1)
	require.True(t, l.Allow("p1"))
	require.False(t, l.Allow("p1"))
	require.True(t, l.Allow("p2"), "p2 는 별도 bucket 이라 허용")
}

func TestLimiter_ZeroRPS_UnlimitedMode(t *testing.T) {
	t.Parallel()
	// rps=0 → "무제한" 모드 — 레이트 리미트 비활성화 같은 설정에 유용.
	l := ratelimit.New(0, 1)
	for i := 0; i < 100; i++ {
		require.True(t, l.Allow("p1"))
	}
}

func TestLimiter_Sweep_RemovesIdleBuckets(t *testing.T) {
	t.Parallel()
	fixed := time.Now()
	clock := &testClock{now: fixed}

	l := ratelimit.New(10, 10,
		ratelimit.WithClock(clock.Now),
		ratelimit.WithIdleTTL(1*time.Minute),
	)
	_ = l.Allow("p1")
	_ = l.Allow("p2")
	require.Equal(t, 2, l.Size())

	// 2 분 경과 — 둘 다 idle 기준 초과.
	clock.Advance(2 * time.Minute)
	removed := l.Sweep()
	require.Equal(t, 2, removed)
	require.Equal(t, 0, l.Size())
}

func TestLimiter_Sweep_KeepsRecent(t *testing.T) {
	t.Parallel()
	fixed := time.Now()
	clock := &testClock{now: fixed}

	l := ratelimit.New(10, 10,
		ratelimit.WithClock(clock.Now),
		ratelimit.WithIdleTTL(1*time.Minute),
	)
	_ = l.Allow("p1")

	clock.Advance(30 * time.Second) // idleTTL 미만.
	removed := l.Sweep()
	require.Equal(t, 0, removed)
	require.Equal(t, 1, l.Size())
}

func TestLimiter_ConcurrentAllow(t *testing.T) {
	t.Parallel()
	// 고루틴 다수가 동시에 Allow 해도 데이터 레이스 없어야 함 (-race).
	l := ratelimit.New(1000, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = l.Allow("p1")
			}
		}()
	}
	wg.Wait()
}

// testClock — 결정적 테스트용.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
