package inmem

import (
	"context"
	"sort"
	"sync"
	"time"

	domain "github.com/kimsehoon/stagesync/internal/domain/event"
)

// EventRepo — in-memory 이벤트 저장소. service/event.Repository 인터페이스 만족.
// mutex 하에서 events · scores · rewards 를 함께 관리.
type EventRepo struct {
	mu      sync.Mutex
	events  map[string]*domain.Event
	scores  map[string]*domain.EventScore // key = eventID + "/" + playerID
	rewards map[string][]domain.RewardTier
}

// NewEventRepo — 빈 저장소 생성.
func NewEventRepo() *EventRepo {
	return &EventRepo{
		events:  map[string]*domain.Event{},
		scores:  map[string]*domain.EventScore{},
		rewards: map[string][]domain.RewardTier{},
	}
}

func scoreKey(eventID, playerID string) string { return eventID + "/" + playerID }

// CreateEvent — 중복 ID 는 ErrAlreadyExists.
func (r *EventRepo) CreateEvent(_ context.Context, e *domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.events[e.ID]; ok {
		return domain.ErrAlreadyExists
	}
	clone := *e
	r.events[e.ID] = &clone
	return nil
}

// GetEvent — 없으면 ErrNotFound.
func (r *EventRepo) GetEvent(_ context.Context, id string) (*domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.events[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *e
	return &clone, nil
}

// ListCurrentEvents — 진행 중인 것만 end_at ASC.
func (r *EventRepo) ListCurrentEvents(_ context.Context, now time.Time) ([]*domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.Event, 0)
	for _, e := range r.events {
		if (e.StartAt.Before(now) || e.StartAt.Equal(now)) &&
			(e.EndAt.After(now) || e.EndAt.Equal(now)) {
			clone := *e
			out = append(out, &clone)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EndAt.Before(out[j].EndAt) })
	return out, nil
}

// AddScore — 누적 가산. 최초면 생성.
func (r *EventRepo) AddScore(_ context.Context, eventID, playerID string, delta int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := scoreKey(eventID, playerID)
	now := time.Now()
	if sc, ok := r.scores[k]; ok {
		sc.Points += delta
		sc.UpdatedAt = now
		return nil
	}
	r.scores[k] = &domain.EventScore{
		EventID:   eventID,
		PlayerID:  playerID,
		Points:    delta,
		UpdatedAt: now,
	}
	return nil
}

// GetScore — 없으면 points=0 으로 반환 (err=nil).
func (r *EventRepo) GetScore(_ context.Context, eventID, playerID string) (*domain.EventScore, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if sc, ok := r.scores[scoreKey(eventID, playerID)]; ok {
		clone := *sc
		return &clone, nil
	}
	return &domain.EventScore{EventID: eventID, PlayerID: playerID}, nil
}

// ListRewardTiers — Tier 오름차순.
func (r *EventRepo) ListRewardTiers(_ context.Context, eventID string) ([]domain.RewardTier, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tiers := r.rewards[eventID]
	out := make([]domain.RewardTier, len(tiers))
	copy(out, tiers)
	sort.Slice(out, func(i, j int) bool { return out[i].Tier < out[j].Tier })
	return out, nil
}

// InsertRewardTier — 단순 append (중복 tier 는 호출자가 책임).
func (r *EventRepo) InsertRewardTier(_ context.Context, t domain.RewardTier) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rewards[t.EventID] = append(r.rewards[t.EventID], t)
	return nil
}
