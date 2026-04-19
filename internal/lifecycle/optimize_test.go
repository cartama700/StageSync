package lifecycle_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/lifecycle"
)

// TestOptimize_DefaultOff — zero value 의 Optimize 는 꺼져 있다.
func TestOptimize_DefaultOff(t *testing.T) {
	t.Parallel()
	var o lifecycle.Optimize
	require.False(t, o.On())
}

// TestOptimize_SetToggle — Set 으로 on/off 전환.
func TestOptimize_SetToggle(t *testing.T) {
	t.Parallel()
	var o lifecycle.Optimize

	o.Set(true)
	require.True(t, o.On())

	o.Set(false)
	require.False(t, o.On())

	o.Set(true)
	require.True(t, o.On())
}

// TestOptimize_ConcurrentReadWrite — Set/On 을 고루틴에서 섞어 호출해도
// 데이터 레이스가 발생하지 않아야 (-race 로 검증).
func TestOptimize_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	var o lifecycle.Optimize

	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N * 2)

	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			o.Set(i%2 == 0)
		}()
		go func() {
			defer wg.Done()
			_ = o.On()
		}()
	}
	wg.Wait()
}
