package aoi

// Naive — 매 호출마다 결과 슬라이스를 새로 할당하는 단순 구현.
// self 는 others 에 포함되지 않는다고 가정 (호출자 책임).
// 반환값: others 내 인덱스 중 self 로부터 radius 이내인 것들.
func Naive(self Point, others []Point, radius float64) []int {
	r2 := radius * radius
	result := make([]int, 0, 64) // 매번 새 할당 (벤치 비교용)
	for i, o := range others {
		if squaredDist(self, o) <= r2 {
			result = append(result, i)
		}
	}
	return result
}
