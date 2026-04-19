package inmem

import (
	"context"
	"sync"
	"time"

	domain "github.com/kimsehoon/stagesync/internal/domain/battle"
)

// BattleRepo — 메모리 기반. MYSQL_DSN 없을 때 graceful fallback.
//
// **주의**: V1Naive 의 "DB 락 경합 재현" 효과는 **실 MySQL 에서만** 의미 있음.
// inmem 은 단순히 mutex 로 보호되므로 V1 = V2 와 동일하게 직렬 실행 → 락 경합 재현 불가.
// Phase 19 벤치는 반드시 `MYSQL_DSN` 설정해서 돌려야 함.
type BattleRepo struct {
	mu         sync.Mutex
	hpByPlayer map[string]int
}

// NewBattleRepo — 빈 저장소.
func NewBattleRepo() *BattleRepo {
	return &BattleRepo{hpByPlayer: map[string]int{}}
}

// ApplyDamageNaive — mutex 로 보호된 get-then-set. 실 MySQL FOR UPDATE 대체.
func (r *BattleRepo) ApplyDamageNaive(_ context.Context, playerID string, damage int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hp, ok := r.hpByPlayer[playerID]
	if !ok {
		hp = domain.DefaultInitialHP
	}
	hp -= damage
	if hp < 0 {
		hp = 0
	}
	r.hpByPlayer[playerID] = hp
	return hp, nil
}

// Get — 현재 HP.
func (r *BattleRepo) Get(_ context.Context, playerID string) (*domain.PlayerHP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hp, ok := r.hpByPlayer[playerID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.PlayerHP{
		PlayerID:  playerID,
		HP:        hp,
		UpdatedAt: time.Now(),
	}, nil
}

// Reset — 테스트 편의.
func (r *BattleRepo) Reset(_ context.Context, playerID string, hp int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hpByPlayer[playerID] = hp
	return nil
}
