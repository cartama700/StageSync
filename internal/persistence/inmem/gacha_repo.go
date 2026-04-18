package inmem

import (
	"context"
	"sync"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
)

// GachaRepo — in-memory 가챠 저장소. service/gacha.Repository 인터페이스 만족.
// mutex 하에서 rolls · pity 를 함께 업데이트 → 호출은 자동으로 원자적.
type GachaRepo struct {
	mu    sync.Mutex
	rolls []*domain.Roll
	pity  map[string]int // key = playerID + "/" + poolID
}

// NewGachaRepo — 빈 저장소 생성.
func NewGachaRepo() *GachaRepo {
	return &GachaRepo{
		pity: map[string]int{},
	}
}

func pityKey(playerID, poolID string) string { return playerID + "/" + poolID }

// GetPity — 없으면 0 반환 (sentinel 아님).
func (r *GachaRepo) GetPity(_ context.Context, playerID, poolID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pity[pityKey(playerID, poolID)], nil
}

// InsertRollsAndUpdatePity — mutex 하에 원자적 업데이트.
func (r *GachaRepo) InsertRollsAndUpdatePity(_ context.Context, rolls []*domain.Roll, playerID, poolID string, newCounter int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rolls = append(r.rolls, rolls...)
	r.pity[pityKey(playerID, poolID)] = newCounter
	return nil
}

// ListRollsByPlayer — 최신 pulled_at 순 limit 건.
// rolls 는 append 순 (= 삽입 시간 순) 이라 끝에서부터 역순 순회.
func (r *GachaRepo) ListRollsByPlayer(_ context.Context, playerID string, limit int) ([]*domain.Roll, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.Roll, 0, limit)
	for i := len(r.rolls) - 1; i >= 0 && len(out) < limit; i-- {
		if r.rolls[i].PlayerID == playerID {
			out = append(out, r.rolls[i])
		}
	}
	return out, nil
}
