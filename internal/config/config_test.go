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

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":5050", cfg.Listen)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, 30*time.Second, cfg.RequestTimeout)
	require.Empty(t, cfg.MySQLDSN)
}

// TestLoad_Overrides — 환경변수로 디폴트 덮어쓰기.
func TestLoad_Overrides(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("REQUEST_TIMEOUT", "10")
	t.Setenv("MYSQL_DSN", "user:pass@tcp(x:3306)/db")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.Listen)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, 5*time.Second, cfg.ShutdownTimeout)
	require.Equal(t, 10*time.Second, cfg.RequestTimeout, "정수 초 형식도 허용")
	require.Equal(t, "user:pass@tcp(x:3306)/db", cfg.MySQLDSN)
}

// TestLoad_InvalidLogLevel — 잘못된 로그 레벨은 거부.
func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "LOG_LEVEL")
}
