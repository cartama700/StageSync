package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
	gen "github.com/kimsehoon/stagesync/internal/persistence/mysql/gen"
)

// mysqlErrDuplicateEntry — MySQL duplicate key 에러 코드.
const mysqlErrDuplicateEntry = 1062

// ProfileRepo — MySQL 기반 Profile 저장소.
// sqlc 가 생성한 gen.Queries 를 래핑하여 service/profile.Repository 인터페이스 만족.
// inmem.ProfileRepo 와 동일 시그니처 → 환경변수로 swap 가능.
type ProfileRepo struct {
	q *gen.Queries
}

// NewProfileRepo — DB 연결을 받아 Repo 생성.
// gen.DBTX 는 *sql.DB 와 *sql.Tx 모두 만족 (트랜잭션 지원).
func NewProfileRepo(db gen.DBTX) *ProfileRepo {
	return &ProfileRepo{q: gen.New(db)}
}

// Get — ID 로 プロフィール 조회.
// 없으면 sql.ErrNoRows → domain.ErrNotFound 로 매핑.
func (r *ProfileRepo) Get(ctx context.Context, id string) (*domain.Profile, error) {
	row, err := r.q.GetProfile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("queries.GetProfile: %w", err)
	}
	return &domain.Profile{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
	}, nil
}

// Create — プロフィール 생성.
// duplicate key (err 1062) → domain.ErrAlreadyExists 로 매핑.
// 여기서 errors.As 로 타입 기반 에러 매칭 (errors.Is 는 값 비교).
func (r *ProfileRepo) Create(ctx context.Context, p *domain.Profile) error {
	err := r.q.CreateProfile(ctx, gen.CreateProfileParams{
		ID:        p.ID,
		Name:      p.Name,
		CreatedAt: p.CreatedAt,
	})
	if err == nil {
		return nil
	}

	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
		return domain.ErrAlreadyExists
	}
	return fmt.Errorf("queries.CreateProfile: %w", err)
}
