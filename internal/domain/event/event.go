// Package event — イベント (이벤트) 도메인 모델.
// Event · EventScore · Status · RewardTier 순수 타입.
package event

import "time"

// Status — 시간 기반 이벤트 상태.
// start_at · end_at · now 로부터 계산 (DB 컬럼이 아닌 derived value).
type Status string

const (
	StatusUpcoming Status = "UPCOMING" // now < start_at
	StatusOngoing  Status = "ONGOING"  // start_at <= now <= end_at
	StatusEnded    Status = "ENDED"    // now > end_at
)

// Event — 이벤트 정의. 일정 기간 운영되는 점수 누적 대상.
type Event struct {
	ID        string
	Name      string
	StartAt   time.Time
	EndAt     time.Time
	CreatedAt time.Time
}

// StatusAt — 특정 시각 기준 상태 계산.
func (e *Event) StatusAt(now time.Time) Status {
	switch {
	case now.Before(e.StartAt):
		return StatusUpcoming
	case now.After(e.EndAt):
		return StatusEnded
	default:
		return StatusOngoing
	}
}

// EventScore — (player, event) 단위 누적 점수 스냅샷.
type EventScore struct {
	EventID   string
	PlayerID  string
	Points    int64
	UpdatedAt time.Time
}

// RewardTier — 점수 구간별 보상.
// 플레이어 점수가 MinPoints 이상이면 Reward 획득 가능.
type RewardTier struct {
	EventID    string
	Tier       int // 1, 2, 3 ... 오름차순
	MinPoints  int64
	RewardName string
}

// EligibleRewards — 점수 기준으로 획득 가능 보상만 필터.
// tiers 는 Tier 오름차순이라 가정. 정렬은 호출자 책임.
func EligibleRewards(tiers []RewardTier, points int64) []RewardTier {
	out := make([]RewardTier, 0, len(tiers))
	for _, t := range tiers {
		if points >= t.MinPoints {
			out = append(out, t)
		}
	}
	return out
}
