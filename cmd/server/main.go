package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/kimsehoon/stagesync/internal/config"
	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/lifecycle"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	mysqlrepo "github.com/kimsehoon/stagesync/internal/persistence/mysql"
	redisrepo "github.com/kimsehoon/stagesync/internal/persistence/redis"
	"github.com/kimsehoon/stagesync/internal/room"
	eventsvc "github.com/kimsehoon/stagesync/internal/service/event"
	gachasvc "github.com/kimsehoon/stagesync/internal/service/gacha"
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
	rankingsvc "github.com/kimsehoon/stagesync/internal/service/ranking"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	// SIGINT/SIGTERM 수신 시 ctx 취소 → graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 공유 인프라
	rm := room.NewRoom()
	optState := &lifecycle.Optimize{}
	readiness := lifecycle.NewReadiness()

	db, cleanup, err := openMaybeMySQL(ctx, cfg.MySQLDSN)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer cleanup()

	// Leaderboard — Redis 주소 있으면 Redis, 없으면 inmem.
	leaderboard, redisCleanup, err := openLeaderboard(ctx, cfg.RedisAddr)
	if err != nil {
		return fmt.Errorf("open redis: %w", err)
	}
	defer redisCleanup()

	profileRepo, gachaRepo, eventRepo := buildRepos(db)
	profileService := profilesvc.NewService(profileRepo)
	gachaPools := gachasvc.NewStaticPoolRegistry()
	gachaService := gachasvc.NewService(gachaRepo, gachaPools)
	eventService := eventsvc.NewService(eventRepo, eventsvc.WithLeaderboard(leaderboard))
	rankingService := rankingsvc.NewService(leaderboard)

	metricsHandler := &endpoint.MetricsHandler{Room: rm, Opt: optState}
	healthHandler := &endpoint.HealthHandler{State: readiness}
	wsHandler := &endpoint.WSHandler{Room: rm}
	optHandler := &endpoint.OptimizeHandler{State: optState}
	profileHandler := &endpoint.ProfileHandler{Service: profileService}
	gachaHandler := &endpoint.GachaHandler{Service: gachaService}
	eventHandler := &endpoint.EventHandler{Service: eventService}
	rankingHandler := &endpoint.RankingHandler{Service: rankingService}
	promHandler := endpoint.NewPrometheusHandler(rm, optState)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(endpoint.RequestLogger(logger))
	r.Use(endpoint.RequestMetrics(promHandler.HTTPDurationHistogram()))
	r.Use(middleware.Recoverer)

	// pprof — Timeout 적용 전에 Mount. `/debug/pprof/profile?seconds=30` 같은 장시간 profile
	// 수집이 RequestTimeout 에 의해 잘리면 안 되기 때문.
	// 프로덕션 노출 시에는 ingress 에서 /debug/* 차단 필요 — 지금은 개발 편의.
	r.Mount("/debug", middleware.Profiler())

	// Timeout 은 Group 안쪽에만 적용 — pprof 를 제외한 모든 앱 라우트.
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(cfg.RequestTimeout))

		metricsHandler.Mount(r)
		healthHandler.Mount(r)
		optHandler.Mount(r)
		profileHandler.Mount(r)
		gachaHandler.Mount(r)
		eventHandler.Mount(r)
		rankingHandler.Mount(r)
		promHandler.Mount(r)
		wsHandler.Mount(r) // long-lived conn 이지만 WebSocket 업그레이드는 Hijack 이라 Timeout 영향 없음
	})

	h2s := &http2.Server{}
	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           h2c.NewHandler(r, h2s),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 서버 고루틴.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server starting",
			"addr", cfg.Listen,
			"protocols", "HTTP/1.1 + h2c",
			"request_timeout", cfg.RequestTimeout,
			"shutdown_timeout", cfg.ShutdownTimeout,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// 시그널 대기 or 서버 장애.
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}

	// Readiness gate — /health/ready 가 503 을 반환하도록 전환 → K8s LB 가 pod 를
	// endpoint 에서 빼낼 시간을 벌어준다. 그 뒤에야 실제 Shutdown 을 시작해야
	// 신규 요청이 in-flight 로 잡히지 않음.
	// Phase 14 full 단계에서 sleep 시간을 configurable (DRAIN_DELAY 환경변수) 로 승격 예정.
	readiness.SetDraining()
	slog.Info("readiness set to draining, awaiting LB to observe")
	time.Sleep(5 * time.Second)

	// Graceful shutdown — in-flight 요청에 ShutdownTimeout 만큼 여유.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		return fmt.Errorf("shutdown: %w", err)
	}
	slog.Info("server stopped cleanly")
	return nil
}

// openMaybeMySQL — DSN 있으면 MySQL 열고 마이그레이션. 없으면 inmem 모드.
func openMaybeMySQL(ctx context.Context, dsn string) (*sql.DB, func(), error) {
	if dsn == "" {
		slog.Info("persistence", "backend", "inmem", "hint", "set MYSQL_DSN for MySQL")
		return nil, func() {}, nil
	}

	db, err := mysqlrepo.Open(ctx, dsn)
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

// openLeaderboard — Redis 주소 있으면 연결 검증 후 Redis 구현, 없으면 inmem.
// rankingsvc.Store 와 eventsvc.LeaderboardWriter 양쪽 인터페이스를 동시에 만족.
func openLeaderboard(ctx context.Context, addr string) (leaderboardBackend, func(), error) {
	if addr == "" {
		slog.Info("leaderboard", "backend", "inmem", "hint", "set REDIS_ADDR for Redis")
		return inmem.NewLeaderboard(), func() {}, nil
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis ping %s: %w", addr, err)
	}
	slog.Info("leaderboard", "backend", "redis", "addr", addr)
	return redisrepo.NewLeaderboard(client), func() { _ = client.Close() }, nil
}

// leaderboardBackend — main.go 가 ranking + event 양쪽에 주입하기 위해 쓰는 합성 인터페이스.
// 실제 두 서비스는 각자 좁은 subset 만 요구하지만, wiring 단계에서는 한 변수로 다뤄야 편함.
type leaderboardBackend interface {
	rankingsvc.Store
	eventsvc.LeaderboardWriter
}

func buildRepos(db *sql.DB) (profilesvc.Repository, gachasvc.Repository, eventsvc.Repository) {
	if db == nil {
		return inmem.NewProfileRepo(), inmem.NewGachaRepo(), inmem.NewEventRepo()
	}
	return mysqlrepo.NewProfileRepo(db), mysqlrepo.NewGachaRepo(db), mysqlrepo.NewEventRepo(db)
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
