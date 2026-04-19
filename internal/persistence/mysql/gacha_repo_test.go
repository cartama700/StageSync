package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
	mysqlrepo "github.com/kimsehoon/stagesync/internal/persistence/mysql"
)

// newMockDB — sqlmock 기반 *sql.DB + 모의 객체 쌍 생성.
// QueryMatcherEqual 을 쓰면 파라미터 바인딩에도 정확히 매칭해 테스트가 덜 깨짐.
// 대신 sqlc 생성 쿼리 문자열과 공백까지 같아야 함 → QueryMatcherRegexp 로 완화.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// TestGachaRepo_GetPity_Hit — 행 존재 → 정수 반환.
func TestGachaRepo_GetPity_Hit(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT counter")).
		WithArgs("p1", "pool1").
		WillReturnRows(sqlmock.NewRows([]string{"counter"}).AddRow(42))

	got, err := repo.GetPity(context.Background(), "p1", "pool1")
	require.NoError(t, err)
	require.Equal(t, 42, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_GetPity_NoRows — sql.ErrNoRows → (0, nil).
func TestGachaRepo_GetPity_NoRows(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT counter")).
		WithArgs("p1", "pool1").
		WillReturnError(sql.ErrNoRows)

	got, err := repo.GetPity(context.Background(), "p1", "pool1")
	require.NoError(t, err)
	require.Equal(t, 0, got, "행이 없으면 0")
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_GetPity_OtherError — 일반 에러는 wrap 해서 전파.
func TestGachaRepo_GetPity_OtherError(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	wantErr := errors.New("boom")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT counter")).
		WithArgs("p1", "pool1").
		WillReturnError(wantErr)

	_, err := repo.GetPity(context.Background(), "p1", "pool1")
	require.ErrorIs(t, err, wantErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_InsertRollsAndUpdatePity_Commit — 성공 플로우.
// BEGIN → InsertRoll*N → UpsertPity → COMMIT 순서로 정확히 호출되는지 검증.
func TestGachaRepo_InsertRollsAndUpdatePity_Commit(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	rolls := []*domain.Roll{
		{ID: "r1", PlayerID: "p1", PoolID: "pool1", CardID: "c1", Rarity: domain.RarityR, IsPity: false, PulledAt: now},
		{ID: "r2", PlayerID: "p1", PoolID: "pool1", CardID: "c2", Rarity: domain.RaritySSR, IsPity: true, PulledAt: now},
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_rolls")).
		WithArgs("r1", "p1", "pool1", "c1", "R", false, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_rolls")).
		WithArgs("r2", "p1", "pool1", "c2", "SSR", true, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_pity")).
		WithArgs("p1", "pool1", int32(0)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.InsertRollsAndUpdatePity(context.Background(), rolls, "p1", "pool1", 0)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_InsertRollsAndUpdatePity_RollbackOnInsertFail —
// 중간 InsertRoll 실패 시 Rollback 호출 + 에러 전파. pity 는 쓰여지면 안 됨.
func TestGachaRepo_InsertRollsAndUpdatePity_RollbackOnInsertFail(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	now := time.Now()
	rolls := []*domain.Roll{
		{ID: "r1", PlayerID: "p1", PoolID: "pool1", CardID: "c1", Rarity: domain.RarityR, PulledAt: now},
		{ID: "r2", PlayerID: "p1", PoolID: "pool1", CardID: "c2", Rarity: domain.RaritySR, PulledAt: now},
	}

	wantErr := errors.New("insert boom")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_rolls")).
		WithArgs("r1", "p1", "pool1", "c1", "R", false, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_rolls")).
		WithArgs("r2", "p1", "pool1", "c2", "SR", false, now).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	err := repo.InsertRollsAndUpdatePity(context.Background(), rolls, "p1", "pool1", 0)
	require.ErrorIs(t, err, wantErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_InsertRollsAndUpdatePity_RollbackOnPityFail —
// UpsertPity 실패 시에도 Rollback.
func TestGachaRepo_InsertRollsAndUpdatePity_RollbackOnPityFail(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	now := time.Now()
	rolls := []*domain.Roll{
		{ID: "r1", PlayerID: "p1", PoolID: "pool1", CardID: "c1", Rarity: domain.RarityR, PulledAt: now},
	}

	wantErr := errors.New("pity boom")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_rolls")).
		WithArgs("r1", "p1", "pool1", "c1", "R", false, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO gacha_pity")).
		WithArgs("p1", "pool1", int32(5)).
		WillReturnError(wantErr)
	mock.ExpectRollback()

	err := repo.InsertRollsAndUpdatePity(context.Background(), rolls, "p1", "pool1", 5)
	require.ErrorIs(t, err, wantErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_InsertRollsAndUpdatePity_BeginFail — BEGIN 자체 실패 시 에러 전파.
func TestGachaRepo_InsertRollsAndUpdatePity_BeginFail(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	wantErr := errors.New("begin boom")
	mock.ExpectBegin().WillReturnError(wantErr)

	err := repo.InsertRollsAndUpdatePity(context.Background(), nil, "p1", "pool1", 0)
	require.ErrorIs(t, err, wantErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_ListRollsByPlayer — row → domain.Roll 매핑.
func TestGachaRepo_ListRollsByPlayer(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	t1 := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	rows := sqlmock.NewRows([]string{"id", "player_id", "pool_id", "card_id", "rarity", "is_pity", "pulled_at"}).
		AddRow("r2", "p1", "pool1", "c2", "SSR", true, t2).
		AddRow("r1", "p1", "pool1", "c1", "R", false, t1)

	mock.ExpectQuery(regexp.QuoteMeta("FROM gacha_rolls")).
		WithArgs("p1", int32(10)).
		WillReturnRows(rows)

	got, err := repo.ListRollsByPlayer(context.Background(), "p1", 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "r2", got[0].ID)
	require.Equal(t, domain.RaritySSR, got[0].Rarity)
	require.True(t, got[0].IsPity)
	require.Equal(t, "r1", got[1].ID)
	require.Equal(t, domain.RarityR, got[1].Rarity)
	require.False(t, got[1].IsPity)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestGachaRepo_ListRollsByPlayer_Empty — 행 없으면 빈 슬라이스.
func TestGachaRepo_ListRollsByPlayer_Empty(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewGachaRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("FROM gacha_rolls")).
		WithArgs("nobody", int32(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "player_id", "pool_id", "card_id", "rarity", "is_pity", "pulled_at"}))

	got, err := repo.ListRollsByPlayer(context.Background(), "nobody", 5)
	require.NoError(t, err)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}
