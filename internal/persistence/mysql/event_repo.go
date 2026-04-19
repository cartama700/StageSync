package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"

	domain "github.com/kimsehoon/stagesync/internal/domain/event"
	gen "github.com/kimsehoon/stagesync/internal/persistence/mysql/gen"
)

// EventRepo — MySQL 기반 이벤트 저장소. service/event.Repository 만족.
type EventRepo struct {
	db *sql.DB
	q  *gen.Queries
}

// NewEventRepo — DB 연결을 받아 Repo 생성.
func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db, q: gen.New(db)}
}

// CreateEvent — 중복 PK → ErrAlreadyExists.
func (r *EventRepo) CreateEvent(ctx context.Context, e *domain.Event) error {
	err := r.q.CreateEvent(ctx, gen.CreateEventParams{
		ID:      e.ID,
		Name:    e.Name,
		StartAt: e.StartAt,
		EndAt:   e.EndAt,
	})
	if err == nil {
		return nil
	}
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
		return domain.ErrAlreadyExists
	}
	return fmt.Errorf("queries.CreateEvent: %w", err)
}

// GetEvent — 없으면 ErrNotFound.
func (r *EventRepo) GetEvent(ctx context.Context, id string) (*domain.Event, error) {
	row, err := r.q.GetEvent(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("queries.GetEvent: %w", err)
	}
	return &domain.Event{
		ID:        row.ID,
		Name:      row.Name,
		StartAt:   row.StartAt,
		EndAt:     row.EndAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

// ListCurrentEvents — start_at <= now AND end_at >= now.
func (r *EventRepo) ListCurrentEvents(ctx context.Context, now time.Time) ([]*domain.Event, error) {
	rows, err := r.q.ListCurrentEvents(ctx, gen.ListCurrentEventsParams{
		StartAt: now,
		EndAt:   now,
	})
	if err != nil {
		return nil, fmt.Errorf("queries.ListCurrentEvents: %w", err)
	}
	out := make([]*domain.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.Event{
			ID:        row.ID,
			Name:      row.Name,
			StartAt:   row.StartAt,
			EndAt:     row.EndAt,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

// AddScore — UPSERT (points = points + delta).
func (r *EventRepo) AddScore(ctx context.Context, eventID, playerID string, delta int64) error {
	if err := r.q.AddEventScore(ctx, gen.AddEventScoreParams{
		EventID:  eventID,
		PlayerID: playerID,
		Points:   delta,
	}); err != nil {
		return fmt.Errorf("queries.AddEventScore: %w", err)
	}
	return nil
}

// GetScore — 없으면 points=0 으로 반환 (sql.ErrNoRows → nil).
func (r *EventRepo) GetScore(ctx context.Context, eventID, playerID string) (*domain.EventScore, error) {
	row, err := r.q.GetEventScore(ctx, gen.GetEventScoreParams{
		EventID:  eventID,
		PlayerID: playerID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return &domain.EventScore{EventID: eventID, PlayerID: playerID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queries.GetEventScore: %w", err)
	}
	return &domain.EventScore{
		EventID:   row.EventID,
		PlayerID:  row.PlayerID,
		Points:    row.Points,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// ListRewardTiers — Tier ASC.
func (r *EventRepo) ListRewardTiers(ctx context.Context, eventID string) ([]domain.RewardTier, error) {
	rows, err := r.q.ListRewardTiers(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("queries.ListRewardTiers: %w", err)
	}
	out := make([]domain.RewardTier, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RewardTier{
			EventID:    row.EventID,
			Tier:       int(row.Tier),
			MinPoints:  row.MinPoints,
			RewardName: row.RewardName,
		})
	}
	return out, nil
}

// InsertRewardTier — 등록.
func (r *EventRepo) InsertRewardTier(ctx context.Context, t domain.RewardTier) error {
	if err := r.q.InsertRewardTier(ctx, gen.InsertRewardTierParams{
		EventID:    t.EventID,
		Tier:       int32(t.Tier),
		MinPoints:  t.MinPoints,
		RewardName: t.RewardName,
	}); err != nil {
		return fmt.Errorf("queries.InsertRewardTier: %w", err)
	}
	return nil
}
