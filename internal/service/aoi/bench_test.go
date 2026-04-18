package aoi

import (
	"math/rand/v2"
	"testing"
)

// benchWorld — N 명 규모의 랜덤 월드 (me 는 원점, 다른 플레이어는 [-100,100] 정사각형 안).
func benchWorld(n int) (Point, []Point) {
	rng := rand.New(rand.NewPCG(1, 2)) // 고정 seed — 벤치 재현성
	me := Point{X: 0, Y: 0}
	others := make([]Point, n)
	for i := range others {
		others[i] = Point{
			X: rng.Float64()*200 - 100,
			Y: rng.Float64()*200 - 100,
		}
	}
	return me, others
}

// benchResultSink — 패키지 변수에 저장해 escape analysis 가 스택 최적화하지 못하게
// 방지. 실제 브로드캐스트·네트워크 전송 상황에 더 근접.
var benchResultSink []int

// BenchmarkNaive — 반환 슬라이스를 패키지 변수에 저장 → heap escape 강제.
// 실제 사용 (브로드캐스트 대상 리스트를 네트워크 계층에 넘김) 시 이 경로 발생.
func BenchmarkNaive(b *testing.B) {
	me, others := benchWorld(1000)
	b.ResetTimer()
	for b.Loop() {
		benchResultSink = Naive(me, others, 30.0)
	}
}

// BenchmarkPooled — callback 안에서만 슬라이스 사용. fn 이 반환되면 pool 로 복귀.
// escape 없음 + pool 재사용 = alloc 0.
func BenchmarkPooled(b *testing.B) {
	me, others := benchWorld(1000)
	var count int
	b.ResetTimer()
	for b.Loop() {
		Pooled(me, others, 30.0, func(indices []int) {
			count = len(indices) // 실제 사용 흉내 (len 만 측정).
		})
	}
	_ = count
}
