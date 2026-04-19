// Package battle — HP 동시 차감 데드락 랩 도메인 모델 (Phase 19).
//
// 최소 구현: player_id × hp 2 필드. 실제 게임은 복잡하지만 본 랩의 목적은
// **"한 유저 row 에 쏠리는 동시 쓰기"** 를 재현하고 3 가지 해결책을 비교하는 것.
package battle

import "time"

// PlayerHP — 플레이어 HP 스냅샷.
type PlayerHP struct {
	PlayerID  string
	HP        int
	UpdatedAt time.Time
}

// 정책 상수.
const (
	// MaxDamagePerRequest — 1 회 요청당 허용 최대 데미지 (악의적 거대값 차단).
	MaxDamagePerRequest = 100_000

	// DefaultInitialHP — ApplyDamage 시 플레이어가 존재하지 않으면 이 값으로 초기화.
	DefaultInitialHP = 10_000
)
