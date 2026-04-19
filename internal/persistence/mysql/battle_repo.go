package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domain "github.com/kimsehoon/stagesync/internal/domain/battle"
)

// BattleRepo — Phase 19 HP 데드락 랩 전용 MySQL 레포지토리.
//
// **핵심 포인트**: `ApplyDamageNaive` 가 단일 트랜잭션 안에서
// `SELECT ... FOR UPDATE` + `UPDATE` 를 실행 → 같은 `player_id` 로 동시 요청이 N 개 오면
// **MySQL 내부 락 대기 큐에 쌓이고 `innodb_lock_wait_timeout` 초과 시 에러 발생**.
// 이게 v1-naive 에서 재현하려는 장애 패턴.
//
// 서비스 레이어의 `V2UserQueue` 가 같은 레포를 사용하지만 **Go 레벨에서 playerID 별로 단일화**
// 하여 DB 에는 한 번에 한 요청만 내려보냄 → 락 경합 0.
type BattleRepo struct {
	db *sql.DB
}

// NewBattleRepo — DB 연결을 받아 레포 생성.
func NewBattleRepo(db *sql.DB) *BattleRepo {
	return &BattleRepo{db: db}
}

// ApplyDamageNaive — v1-naive 구현.
//
// 순서:
//  1. BEGIN
//  2. SELECT hp FROM player_hp WHERE player_id = ? FOR UPDATE   ← 행 락 획득
//  3. 행 없으면 INSERT (INITIAL_HP)
//  4. UPDATE player_hp SET hp = hp - ? WHERE player_id = ?
//  5. COMMIT
//
// 동시 요청이 같은 player_id 에 쏠리면 2번에서 락 대기 → 대부분은 순차 처리되지만
// `innodb_lock_wait_timeout` 기본 50s 넘어가는 극단 상황에선 에러.
// 스트레스 테스트에서 `SET SESSION innodb_lock_wait_timeout = 2` 로 낮추면
// 쉽게 "lock wait timeout exceeded" 를 재현 가능.
func (r *BattleRepo) ApplyDamageNaive(ctx context.Context, playerID string, damage int) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var hp int
	err = tx.QueryRowContext(ctx,
		`SELECT hp FROM player_hp WHERE player_id = ? FOR UPDATE`, playerID,
	).Scan(&hp)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// 초기화 — 게임 로직에 맞게 기본 HP 로 삽입.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO player_hp (player_id, hp) VALUES (?, ?)`,
			playerID, domain.DefaultInitialHP,
		); err != nil {
			return 0, fmt.Errorf("insert hp: %w", err)
		}
		hp = domain.DefaultInitialHP
	case err != nil:
		return 0, fmt.Errorf("select for update: %w", err)
	}

	newHP := hp - damage
	if newHP < 0 {
		newHP = 0
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE player_hp SET hp = ? WHERE player_id = ?`, newHP, playerID,
	); err != nil {
		return 0, fmt.Errorf("update hp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return newHP, nil
}

// Get — 현재 HP 조회 (락 없음).
func (r *BattleRepo) Get(ctx context.Context, playerID string) (*domain.PlayerHP, error) {
	var p domain.PlayerHP
	err := r.db.QueryRowContext(ctx,
		`SELECT player_id, hp, updated_at FROM player_hp WHERE player_id = ?`, playerID,
	).Scan(&p.PlayerID, &p.HP, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return &p, nil
}

// Reset — 테스트·벤치 편의. hp 를 특정 값으로 설정 (UPSERT).
func (r *BattleRepo) Reset(ctx context.Context, playerID string, hp int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO player_hp (player_id, hp) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE hp = VALUES(hp)`,
		playerID, hp,
	)
	if err != nil {
		return fmt.Errorf("reset hp: %w", err)
	}
	return nil
}
