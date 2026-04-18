package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/lifecycle"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	mysqlrepo "github.com/kimsehoon/stagesync/internal/persistence/mysql"
	"github.com/kimsehoon/stagesync/internal/room"
	gachasvc "github.com/kimsehoon/stagesync/internal/service/gacha"
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
)

const listenAddr = ":5050"

func main() {
	if err := run(); err != nil {
		slog.Error("server fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 공유 인프라
	rm := room.NewRoom()
	optState := &lifecycle.Optimize{}

	// DB 열기 + 마이그레이션 (MYSQL_DSN 있을 때만)
	db, cleanup, err := openMaybeMySQL()
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer cleanup()

	// Repository 들 — db 가 nil 이면 inmem, 아니면 mysql
	profileRepo, gachaRepo := buildRepos(db)

	// Service 들
	profileService := profilesvc.NewService(profileRepo)
	gachaPools := gachasvc.NewStaticPoolRegistry()
	gachaService := gachasvc.NewService(gachaRepo, gachaPools)

	// Handler 들 (모두 Mount 패턴)
	metricsHandler := &endpoint.MetricsHandler{Room: rm, Opt: optState}
	healthHandler := &endpoint.HealthHandler{}
	wsHandler := &endpoint.WSHandler{Room: rm}
	optHandler := &endpoint.OptimizeHandler{State: optState}
	profileHandler := &endpoint.ProfileHandler{Service: profileService}
	gachaHandler := &endpoint.GachaHandler{Service: gachaService}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// 각 핸들러가 자기 라우트를 직접 등록.
	metricsHandler.Mount(r)
	healthHandler.Mount(r)
	optHandler.Mount(r)
	profileHandler.Mount(r)
	gachaHandler.Mount(r)
	wsHandler.Mount(r) // 보너스축

	h2s := &http2.Server{}
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: h2c.NewHandler(r, h2s),
	}

	slog.Info("server starting",
		"addr", listenAddr,
		"protocols", "HTTP/1.1 + h2c (WebSocket upgrade preserved)",
	)
	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

// openMaybeMySQL — MYSQL_DSN 있으면 MySQL 열고 마이그레이션 실행.
// 없으면 (nil, no-op, nil) 반환 → inmem 모드.
func openMaybeMySQL() (*sql.DB, func(), error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		slog.Info("persistence", "backend", "inmem", "hint", "set MYSQL_DSN for MySQL")
		return nil, func() {}, nil
	}

	db, err := mysqlrepo.Open(context.Background(), dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("mysql open: %w", err)
	}
	cleanup := func() { _ = db.Close() }

	if err := mysqlrepo.Migrate(db); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("mysql migrate: %w", err)
	}
	slog.Info("persistence", "backend", "mysql")
	return db, cleanup, nil
}

// buildRepos — db 있으면 MySQL repo, 없으면 inmem. 두 도메인 repo 를 함께 묶어 반환.
func buildRepos(db *sql.DB) (profilesvc.Repository, gachasvc.Repository) {
	if db == nil {
		return inmem.NewProfileRepo(), inmem.NewGachaRepo()
	}
	return mysqlrepo.NewProfileRepo(db), mysqlrepo.NewGachaRepo(db)
}
