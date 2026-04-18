package main

import (
	"context"
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
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
)

const listenAddr = ":5050"

func main() {
	// run() 패턴 — defer 가 os.Exit 전에 확실히 실행되도록 분리.
	if err := run(); err != nil {
		slog.Error("server fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 의존성 wire-up (수동 DI — 일본 Go 관행 R5).
	rm := room.NewRoom()
	optState := &lifecycle.Optimize{}

	// プロフィール 레이어 체인 — MYSQL_DSN 있으면 MySQL, 없으면 inmem.
	profileRepo, cleanup, err := buildProfileRepo()
	if err != nil {
		return fmt.Errorf("profile repo: %w", err)
	}
	defer cleanup()
	profileService := profilesvc.NewService(profileRepo)

	// Handler 들 (모두 구조체 메서드 패턴)
	metricsHandler := &endpoint.MetricsHandler{Room: rm, Opt: optState}
	healthHandler := &endpoint.HealthHandler{}
	wsHandler := &endpoint.WSHandler{Room: rm}
	optHandler := &endpoint.OptimizeHandler{State: optState}
	profileHandler := &endpoint.ProfileHandler{Service: profileService}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	// 각 핸들러가 자기 라우트를 직접 등록 (도메인이 자기 URL 책임짐).
	metricsHandler.Mount(r)
	optHandler.Mount(r)
	healthHandler.Mount(r)
	profileHandler.Mount(r)
	wsHandler.Mount(r)

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

// buildProfileRepo — MYSQL_DSN 있으면 MySQL repo + goose migrate, 없으면 inmem.
// cleanup 은 defer 로 호출; inmem 일 땐 no-op.
func buildProfileRepo() (profilesvc.Repository, func(), error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		slog.Info("persistence", "backend", "inmem", "hint", "set MYSQL_DSN for MySQL")
		return inmem.NewProfileRepo(), func() {}, nil
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
	return mysqlrepo.NewProfileRepo(db), cleanup, nil
}
