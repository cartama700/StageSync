// Package gacha — ガチャ (가챠) 도메인 모델.
// Card · Pool · Roll · Rarity · PityState 순수 타입과 관련 상수.
package gacha

import "time"

// Rarity — 카드 레어도. R / SR / SSR.
type Rarity string

const (
	RarityR   Rarity = "R"
	RaritySR  Rarity = "SR"
	RaritySSR Rarity = "SSR"
)

// IsValid — 알려진 Rarity 값인지 확인.
func (r Rarity) IsValid() bool {
	switch r {
	case RarityR, RaritySR, RaritySSR:
		return true
	}
	return false
}

// Card — 가챠 풀의 개별 카드/캐릭터.
type Card struct {
	ID     string
	Name   string
	Rarity Rarity
	Weight int // 상대 가중치 (같은 풀 내 합산 분모로)
}

// Pool — 한 가챠 풀 정의. 일정 기간 운영되는 스트링 이벤트 배너 단위.
type Pool struct {
	ID            string
	Name          string
	PityThreshold int // 이 횟수 연속 SSR 없으면 다음 roll 에서 SSR 확정
	Cards         []Card
}

// Roll — 한 번의 뽑기 결과 기록.
type Roll struct {
	ID       string // UUID v7 (time-ordered)
	PlayerID string
	PoolID   string
	CardID   string
	Rarity   Rarity
	IsPity   bool // 천장으로 확정된 roll 여부
	PulledAt time.Time
}

// PityState — 플레이어 × 풀 단위 천장 카운터 스냅샷.
type PityState struct {
	PlayerID string
	PoolID   string
	Counter  int // 마지막 SSR 이후 roll 수
}
