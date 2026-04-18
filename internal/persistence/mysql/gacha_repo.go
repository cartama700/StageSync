package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
	gen "github.com/kimsehoon/stagesync/internal/persistence/mysql/gen"
)

// GachaRepo — MySQL 기반 가챠 저장소. service/gacha.Repository 만족.
// `InsertRollsAndUpdatePity` 가 단일 트랜잭션으로 rolls + pity 를 원자적 쓰기.
type GachaRepo struct {
	db *sql.DB
	q  *gen.Queries
}

// NewGachaRepo — DB 연결을 받아 Repo 생성.
func NewGachaRepo(db *sql.DB) *GachaRepo {
	return &GachaRepo{
		db: db,
		q:  gen.New(db),
	}
}

// GetPity — 행 없으면 0 반환 (sql.ErrNoRows 를 nil + 0 으로 매핑).
func (r *GachaRepo) GetPity(ctx context.Context, playerID, poolID string) (int, error) {
	c, err := r.q.GetPity(ctx, gen.GetPityParams{
		PlayerID: playerID,
		PoolID:   poolID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("queries.GetPity: %w", err)
	}
	return int(c), nil
}

// InsertRollsAndUpdatePity — 10-roll 전체를 원자적 트랜잭션으로 처리.
// 실패 시 전체 롤백 (rolls · pity 둘 다 쓰여지지 않음).
func (r *GachaRepo) InsertRollsAndUpdatePity(
	ctx context.Context,
	rolls []*domain.Roll,
	playerID, poolID string,
	newCounter int,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// defer Rollback 은 안전망 — Commit 성공하면 no-op.
	defer func() { _ = tx.Rollback() }()

	qtx := r.q.WithTx(tx)

	for _, roll := range rolls {
		if err := qtx.InsertRoll(ctx, gen.InsertRollParams{
			ID:       roll.ID,
			PlayerID: roll.PlayerID,
			PoolID:   roll.PoolID,
			CardID:   roll.CardID,
			Rarity:   string(roll.Rarity),
			IsPity:   roll.IsPity,
			PulledAt: roll.PulledAt,
		}); err != nil {
			return fmt.Errorf("insert roll %s: %w", roll.ID, err)
		}
	}

	if err := qtx.UpsertPity(ctx, gen.UpsertPityParams{
		PlayerID: playerID,
		PoolID:   poolID,
		Counter:  int32(newCounter),
	}); err != nil {
		return fmt.Errorf("upsert pity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListRollsByPlayer — 최신순 limit 건.
func (r *GachaRepo) ListRollsByPlayer(ctx context.Context, playerID string, limit int) ([]*domain.Roll, error) {
	rows, err := r.q.ListRollsByPlayer(ctx, gen.ListRollsByPlayerParams{
		PlayerID: playerID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("queries.ListRollsByPlayer: %w", err)
	}
	out := make([]*domain.Roll, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.Roll{
			ID:       row.ID,
			PlayerID: row.PlayerID,
			PoolID:   row.PoolID,
			CardID:   row.CardID,
			Rarity:   domain.Rarity(row.Rarity),
			IsPity:   row.IsPity,
			PulledAt: row.PulledAt,
		})
	}
	return out, nil
}
