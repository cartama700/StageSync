package mysql

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// migrationsFS — migrations/*.sql 을 바이너리에 내장.
// 배포 시 별도 마이그레이션 파일 배포가 불필요.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate — goose up 실행.
// 서버 기동 시 호출하여 최신 스키마 자동 적용 (개발·시연 편의).
// 프로덕션에선 별도 배포 스텝으로 분리 권장.
func Migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
