package aoi_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/service/aoi"
)

// TestFilter — Naive · Pooled 두 구현 모두 동일한 정확성 보장.
// 테이블 주도 + t.Parallel 로 서브테스트 병렬 실행.
func TestFilter(t *testing.T) {
	t.Parallel()

	me := aoi.Point{X: 0, Y: 0}
	others := []aoi.Point{
		{X: 10, Y: 0},   // d=10   → 안
		{X: 0, Y: 30},   // d=30   → 안
		{X: 30, Y: 40},  // d=50   → 경계 (포함)
		{X: 51, Y: 0},   // d=51   → 밖
		{X: 70, Y: 70},  // d≈99   → 밖
		{X: -20, Y: 30}, // d≈36   → 안
	}
	radius := 50.0
	want := []int{0, 1, 2, 5}

	tests := []struct {
		name string
		run  func() []int
	}{
		{
			name: "Naive",
			run:  func() []int { return aoi.Naive(me, others, radius) },
		},
		{
			name: "Pooled",
			run: func() []int {
				var got []int
				aoi.Pooled(me, others, radius, func(indices []int) {
					got = append([]int{}, indices...)
				})
				return got
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.run()
			sort.Ints(got)
			require.Equal(t, want, got)
		})
	}
}

// TestFilterEquivalence — 다양한 데이터셋에서 Naive · Pooled 결과가 동일함.
func TestFilterEquivalence(t *testing.T) {
	t.Parallel()

	datasets := []struct {
		name   string
		me     aoi.Point
		others []aoi.Point
		radius float64
	}{
		{
			name:   "small scattered",
			me:     aoi.Point{X: 0, Y: 0},
			others: []aoi.Point{{X: 1, Y: 1}, {X: 10, Y: 10}, {X: 100, Y: 100}},
			radius: 5.0,
		},
		{
			name:   "all inside",
			me:     aoi.Point{X: 50, Y: 50},
			others: []aoi.Point{{X: 49, Y: 49}, {X: 51, Y: 51}, {X: 50, Y: 50}},
			radius: 10.0,
		},
		{
			name:   "all outside",
			me:     aoi.Point{X: 0, Y: 0},
			others: []aoi.Point{{X: 1000, Y: 0}, {X: -999, Y: 0}},
			radius: 5.0,
		},
		{
			name:   "empty",
			me:     aoi.Point{X: 0, Y: 0},
			others: []aoi.Point{},
			radius: 100.0,
		},
	}

	for _, ds := range datasets {
		ds := ds
		t.Run(ds.name, func(t *testing.T) {
			t.Parallel()

			naiveResult := aoi.Naive(ds.me, ds.others, ds.radius)
			sort.Ints(naiveResult)

			var pooledResult []int
			aoi.Pooled(ds.me, ds.others, ds.radius, func(indices []int) {
				pooledResult = append([]int{}, indices...)
			})
			sort.Ints(pooledResult)

			require.Equal(t, naiveResult, pooledResult)
		})
	}
}
