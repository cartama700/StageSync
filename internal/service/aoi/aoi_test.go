package aoi

import (
	"sort"
	"testing"
)

// testWorld — 결정적 (deterministic) 월드 생성: me 중심 반경 1~100 사이 거리의 점들.
// 절반은 AOI 반경(50) 이내, 절반은 바깥.
func testWorld() (Point, []Point) {
	me := Point{X: 0, Y: 0}
	others := []Point{
		{X: 10, Y: 0},   // d=10    → 안
		{X: 0, Y: 30},   // d=30    → 안
		{X: 30, Y: 40},  // d=50    → 경계
		{X: 51, Y: 0},   // d=51    → 밖
		{X: 70, Y: 70},  // d≈99    → 밖
		{X: -20, Y: 30}, // d≈36    → 안
	}
	return me, others
}

// TestNaive_Correctness — Naive 가 올바른 인덱스 리스트를 반환하는지.
func TestNaive_Correctness(t *testing.T) {
	me, others := testWorld()
	got := Naive(me, others, 50.0)
	sort.Ints(got)
	want := []int{0, 1, 2, 5}
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d (got=%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got=%d want=%d", i, got[i], want[i])
		}
	}
}

// TestPooled_Correctness — Pooled 도 같은 결과를 낸다.
func TestPooled_Correctness(t *testing.T) {
	me, others := testWorld()
	var got []int
	Pooled(me, others, 50.0, func(indices []int) {
		// fn 바깥으로 유출 방지 위해 복사.
		got = append([]int(nil), indices...)
	})
	sort.Ints(got)
	want := []int{0, 1, 2, 5}
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d (got=%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] got=%d want=%d", i, got[i], want[i])
		}
	}
}

// TestEquivalence — Naive 와 Pooled 가 동일한 결과를 보장한다.
func TestEquivalence(t *testing.T) {
	me, others := testWorld()
	naiveResult := Naive(me, others, 50.0)
	sort.Ints(naiveResult)

	var pooledResult []int
	Pooled(me, others, 50.0, func(indices []int) {
		pooledResult = append([]int(nil), indices...)
	})
	sort.Ints(pooledResult)

	if len(naiveResult) != len(pooledResult) {
		t.Fatalf("length mismatch: naive=%d pooled=%d", len(naiveResult), len(pooledResult))
	}
	for i := range naiveResult {
		if naiveResult[i] != pooledResult[i] {
			t.Errorf("[%d] naive=%d pooled=%d", i, naiveResult[i], pooledResult[i])
		}
	}
}
