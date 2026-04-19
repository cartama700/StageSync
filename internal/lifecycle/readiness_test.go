package lifecycle_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/lifecycle"
)

// TestReadiness_DefaultReady — NewReadiness 직후엔 트래픽 수용 가능.
func TestReadiness_DefaultReady(t *testing.T) {
	t.Parallel()
	r := lifecycle.NewReadiness()
	require.True(t, r.Ready())
}

// TestReadiness_SetDraining — SetDraining 후 Ready() == false.
// 멱등 — 여러 번 불러도 false 유지.
func TestReadiness_SetDraining(t *testing.T) {
	t.Parallel()
	r := lifecycle.NewReadiness()

	r.SetDraining()
	require.False(t, r.Ready())

	r.SetDraining()
	require.False(t, r.Ready())
}

// TestReadiness_ConcurrentReadWrite — SetDraining / Ready 를 고루틴에서 섞어 호출해도
// 데이터 레이스가 발생하지 않아야 (-race 로 검증).
func TestReadiness_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	r := lifecycle.NewReadiness()

	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N * 2)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = r.Ready()
		}()
		go func() {
			defer wg.Done()
			r.SetDraining()
		}()
	}
	wg.Wait()

	// drain 호출 다발 이후에는 false 로 수렴해야 함.
	require.False(t, r.Ready())
}
