package main

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseScenario — 알려진 이름만 허용.
func TestParseScenario(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"even", "herd", "cluster"} {
		_, err := parseScenario(name)
		require.NoError(t, err, "scenario %q should be valid", name)
	}

	_, err := parseScenario("unknown")
	require.Error(t, err)
}

// TestScenarios_BoundsAndDeterminism —
// 각 시나리오가 의도한 범위 안에서 값을 내는지 + 같은 seed 면 결정적인지.
func TestScenarios_BoundsAndDeterminism(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scen     scenario
		minX, maxX float64
		minY, maxY float64
	}{
		{"even", evenScenario, -500, 500, -500, 500},
		{"herd", herdScenario, -20, 20, -20, 20},
		// cluster 는 중심 ±30 → 최외곽 {±300, ±300} 기준 ±30 → -330..330.
		{"cluster", clusterScenario, -330, 330, -330, 330},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r1 := rand.New(rand.NewPCG(42, 42))
			r2 := rand.New(rand.NewPCG(42, 42))

			for i := uint64(0); i < 200; i++ {
				x1, y1 := tc.scen(r1, i)
				x2, y2 := tc.scen(r2, i)
				require.Equal(t, x1, x2, "같은 seed → 같은 값")
				require.Equal(t, y1, y2)

				require.GreaterOrEqual(t, x1, tc.minX)
				require.LessOrEqual(t, x1, tc.maxX)
				require.GreaterOrEqual(t, y1, tc.minY)
				require.LessOrEqual(t, y1, tc.maxY)
			}
		})
	}
}
