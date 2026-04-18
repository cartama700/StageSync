// Package aoi — Area of Interest 필터.
// リズムゲーム의 バーチャルライブ 같은 상황에서 "내 반경 R 안의 플레이어" 만
// 추려내는 핵심 핫패스. naive (매번 할당) · pooled (sync.Pool 재사용) 두 구현 제공.
package aoi

// Point — 2D 월드 좌표.
type Point struct {
	X, Y float64
}

// squaredDist — 두 점 사이 거리의 제곱 (sqrt 비용 회피).
func squaredDist(a, b Point) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
