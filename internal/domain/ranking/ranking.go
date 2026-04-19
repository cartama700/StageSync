// Package ranking — 랭킹 도메인 모델.
// 이벤트 × 플레이어 점수 기반 랭킹 Entry 와 관련 상수.
package ranking

// Entry — 단일 랭킹 엔트리 (플레이어 + 점수 + 순위).
// Rank 는 1-based — 1 위부터 시작.
type Entry struct {
	PlayerID string
	Score    int64
	Rank     int
}

// 정책 상수.
const (
	// MaxTopN — Top-N 조회 허용 상한. 과도한 응답 크기 방지.
	MaxTopN = 100

	// DefaultTopN — ?n 미지정 시 기본값.
	DefaultTopN = 10

	// MaxAroundRadius — 내 주변 랭킹의 ±radius 상한.
	MaxAroundRadius = 25

	// DefaultAroundRadius — 기본 ±5.
	DefaultAroundRadius = 5
)
