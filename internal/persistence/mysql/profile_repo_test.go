package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
	mysqlrepo "github.com/kimsehoon/stagesync/internal/persistence/mysql"
)

// TestProfileRepo_Get_Hit — 행 존재 → 매핑된 Profile 반환.
func TestProfileRepo_Get_Hit(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	created := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM profiles")).
		WithArgs("u1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_at"}).
			AddRow("u1", "alice", created))

	got, err := repo.Get(context.Background(), "u1")
	require.NoError(t, err)
	require.Equal(t, "u1", got.ID)
	require.Equal(t, "alice", got.Name)
	require.True(t, got.CreatedAt.Equal(created))
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Get_NotFound — sql.ErrNoRows → domain.ErrNotFound 매핑.
func TestProfileRepo_Get_NotFound(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta("FROM profiles")).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.Get(context.Background(), "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Get_OtherError — 일반 에러는 wrap 해서 전파.
func TestProfileRepo_Get_OtherError(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	wantErr := errors.New("connection lost")
	mock.ExpectQuery(regexp.QuoteMeta("FROM profiles")).
		WithArgs("u1").
		WillReturnError(wantErr)

	_, err := repo.Get(context.Background(), "u1")
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, domain.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Create_Success — 정상 INSERT.
func TestProfileRepo_Create_Success(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	created := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	p := &domain.Profile{ID: "u1", Name: "alice", CreatedAt: created}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles")).
		WithArgs("u1", "alice", created).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Create(context.Background(), p)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Create_Duplicate — MySQL err 1062 → domain.ErrAlreadyExists.
// 이 매핑은 MySQL-specific 이라 mock 에서 드라이버 에러를 직접 반환해야 검증 가능.
func TestProfileRepo_Create_Duplicate(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	dupErr := &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'u1' for key 'PRIMARY'"}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles")).
		WithArgs("u1", "alice", sqlmock.AnyArg()).
		WillReturnError(dupErr)

	err := repo.Create(context.Background(), &domain.Profile{ID: "u1", Name: "alice", CreatedAt: time.Now()})
	require.ErrorIs(t, err, domain.ErrAlreadyExists)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Create_OtherMySQLError — duplicate 외 MySQL 에러는 wrap.
func TestProfileRepo_Create_OtherMySQLError(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	otherErr := &mysqldriver.MySQLError{Number: 1048, Message: "Column cannot be null"}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles")).
		WithArgs("u1", "alice", sqlmock.AnyArg()).
		WillReturnError(otherErr)

	err := repo.Create(context.Background(), &domain.Profile{ID: "u1", Name: "alice", CreatedAt: time.Now()})
	require.Error(t, err)
	require.NotErrorIs(t, err, domain.ErrAlreadyExists)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestProfileRepo_Create_GenericError — non-MySQL 에러도 wrap.
func TestProfileRepo_Create_GenericError(t *testing.T) {
	t.Parallel()
	db, mock := newMockDB(t)
	repo := mysqlrepo.NewProfileRepo(db)

	wantErr := errors.New("boom")
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles")).
		WithArgs("u1", "alice", sqlmock.AnyArg()).
		WillReturnError(wantErr)

	err := repo.Create(context.Background(), &domain.Profile{ID: "u1", Name: "alice", CreatedAt: time.Now()})
	require.ErrorIs(t, err, wantErr)
	require.NotErrorIs(t, err, domain.ErrAlreadyExists)
	require.NoError(t, mock.ExpectationsWereMet())
}
