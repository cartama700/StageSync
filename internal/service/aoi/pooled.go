package aoi

import "sync"

// idxPool — []int 버퍼를 재사용하는 풀. New 는 초기 capacity 64 인 빈 슬라이스 제공.
var idxPool = sync.Pool{
	New: func() any {
		b := make([]int, 0, 64)
		return &b
	},
}

// Pooled — sync.Pool 에서 버퍼를 꺼내 재사용하는 구현.
// 호출자가 fn 안에서 결과를 소비한다. fn 이 반환되면 버퍼는 pool 로 돌려보내짐.
// (Release 누락 방지 위한 callback 스타일.)
func Pooled(self Point, others []Point, radius float64, fn func([]int)) {
	bp := idxPool.Get().(*[]int)
	buf := (*bp)[:0] // reset — 이전 데이터 지우기
	defer func() {
		*bp = buf[:0]
		idxPool.Put(bp)
	}()

	r2 := radius * radius
	for i, o := range others {
		if squaredDist(self, o) <= r2 {
			buf = append(buf, i)
		}
	}
	fn(buf)
}
