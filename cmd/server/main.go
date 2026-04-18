package main

import (
	"context"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 의존성 wire-up (수동 DI — 일본 Go 관행 R5).
	// Persistence → Service → Endpoint 순서로 조립.
	rm := room.NewRoom()
	optState := &lifecycle.Optimize{}

	// プロフィール 레이어 체인 — MYSQL_DSN 있으면 MySQL, 없으면 inmem (graceful degrade).
	var profileRepo profilesvc.Repository
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		db, err := mysqlrepo.Open(context.Background(), dsn)
		if err != nil {
			slog.Error("mysql open", "err", err)
			os.Exit(1)
		}
		defer func() { _ = db.Close() }()

		if err := mysqlrepo.Migrate(db); err != nil {
			slog.Error("mysql migrate", "err", err)
			os.Exit(1)
		}
		profileRepo = mysqlrepo.NewProfileRepo(db)
		slog.Info("persistence", "backend", "mysql")
	} else {
		profileRepo = inmem.NewProfileRepo()
		slog.Info("persistence", "backend", "inmem", "hint", "set MYSQL_DSN for MySQL")
	}
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

	// 관측·운영
	r.Get("/api/metrics", metricsHandler.Get)
	r.Post("/api/optimize", optHandler.ServeHTTP)
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	// プロフィール REST
	r.Get("/api/profile/{id}", profileHandler.Get)
	r.Post("/api/profile", profileHandler.Create)

	// 실시간 (보너스축)
	r.Get("/ws/room", wsHandler.ServeHTTP)

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
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
