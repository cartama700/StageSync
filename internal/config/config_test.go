package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/config"
)

// TestLoad_Defaults — 환경변수 없을 때 안전한 디폴트.
func TestLoad_Defaults(t *testing.T) {
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("REQUEST_TIMEOUT", "")
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("AUTH_SECRET", "")
	t.Setenv("AUTH_TOKEN_TTL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":5050", cfg.Listen)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, 30*time.Second, cfg.RequestTimeout)
	require.Empty(t, cfg.MySQLDSN)
	require.Empty(t, cfg.RedisAddr)
	require.Empty(t, cfg.AuthSecret, "AUTH_SECRET 비어있으면 인증 비활성 (개발 편의)")
	require.Equal(t, 15*time.Minute, cfg.AuthTokenTTL)
	require.InDelta(t, 10.0, cfg.RateLimitRPS, 0.001)
	require.Equal(t, 20, cfg.RateLimitBurst)
	require.Equal(t, 5*time.Minute, cfg.IdempotencyTTL)
}

// TestLoad_Overrides — 환경변수로 디폴트 덮어쓰기.
func TestLoad_Overrides(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("REQUEST_TIMEOUT", "10")
	t.Setenv("MYSQL_DSN", "user:pass@tcp(x:3306)/db")
	t.Setenv("AUTH_SECRET", "super-secret-48-bytes-random-base64")
	t.Setenv("AUTH_TOKEN_TTL", "1h")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.Listen)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, 5*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, 10*time.Second, cfg.RequestTimeout, "정수 초 형식도 허용")
	require.Equal(t, "user:pass@tcp(x:3306)/db", cfg.MySQLDSN)
	require.Equal(t, "super-secret-48-bytes-random-base64", cfg.AuthSecret)
	require.Equal(t, time.Hour, cfg.AuthTokenTTL)
}

// TestLoad_RateLimitOverride — RATE_LIMIT_RPS · BURST · IDEMPOTENCY_TTL 덮어쓰기.
func TestLoad_RateLimitOverride(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "100")
	t.Setenv("RATE_LIMIT_BURST", "200")
	t.Setenv("IDEMPOTENCY_TTL", "30s")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.InDelta(t, 100.0, cfg.RateLimitRPS, 0.001)
	require.Equal(t, 200, cfg.RateLimitBurst)
	require.Equal(t, 30*time.Second, cfg.IdempotencyTTL)
}

// TestLoad_InvalidLogLevel — 잘못된 로그 레벨은 거부.
func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "LOG_LEVEL")
}
