// Package mysql — Aurora MySQL 호환 저장소.
// 현재는 profiles 테이블만. 후속 Phase 에서 players, mail, event_scores 등 추가.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Blank import — MySQL driver 가 init() 에서 database/sql 에 자기 등록.
	// driver 의 실제 심볼은 사용자 코드에서 직접 쓸 일 없음.
	_ "github.com/go-sql-driver/mysql"
)

// Open — DSN 으로 MySQL 연결 + 풀 기본값 설정.
// DSN 예: "user:pass@tcp(127.0.0.1:3306)/dbname?parseTime=true&loc=Local"
// parseTime=true 는 TIMESTAMP → time.Time 자동 변환에 필수.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// 풀 설정 — 게임 서버 기본값 (공고 高負荷 대응).
	// 실제 프로덕션에선 부하 프로파일링 후 조정.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	// 실제 연결 확인 — sql.Open 은 lazy 이므로 Ping 으로 검증.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return db, nil
}
