// Package config — 환경변수 기반 서버 설정.
// main.go 가 직접 os.Getenv 를 호출하지 않도록 단일 진입점으로 집중.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config — 서버 런타임 설정.
// 필드는 zero-value 가 안전한 디폴트가 되도록 설계 (DSN 빈 문자열 = inmem 모드).
type Config struct {
	// Listen — HTTP 서버 바인딩 주소 (ex. ":5050").
	Listen string

	// LogLevel — slog 레벨 ("debug"|"info"|"warn"|"error").
	LogLevel string

	// ShutdownTimeout — SIGTERM 수신 시 in-flight 요청에 줄 최대 시간.
	ShutdownTimeout time.Duration

	// RequestTimeout — 개별 HTTP 요청의 최대 처리 시간.
	RequestTimeout time.Duration

	// MySQLDSN — 비어있으면 inmem 모드.
	MySQLDSN string

	// RedisAddr — 비어있으면 inmem leaderboard 로 graceful degrade.
	// 예: "127.0.0.1:6379" 또는 docker compose 안에서 "redis:6379".
	RedisAddr string

	// AuthSecret — JWT HS256 서명 시크릿.
	// **빈 문자열 = 인증 비활성** (로컬 개발 · 기존 테스트 호환). 프로덕션은 반드시 세팅.
	// 예: `openssl rand -base64 48` 같은 랜덤 48 byte+ 권장.
	AuthSecret string

	// AuthTokenTTL — 발급된 JWT 의 유효기간. 기본 15 분.
	AuthTokenTTL time.Duration

	// RateLimitRPS — identity 별 평균 초당 허용 요청 수. 0 이면 Rate Limit 비활성.
	RateLimitRPS float64

	// RateLimitBurst — identity 별 버스트 허용량 (토큰 버킷 최대 크기). 기본 20.
	RateLimitBurst int

	// IdempotencyTTL — `Idempotency-Key` 캐시 유효시간. 기본 5 분.
	IdempotencyTTL time.Duration
}

// Load — 환경변수에서 Config 생성. 유효성 에러 발생 시 즉시 실패.
func Load() (*Config, error) {
	cfg := &Config{
		Listen:          getEnv("LISTEN_ADDR", ":5050"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		RequestTimeout:  getDurationEnv("REQUEST_TIMEOUT", 30*time.Second),
		MySQLDSN:        os.Getenv("MYSQL_DSN"),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		AuthSecret:      os.Getenv("AUTH_SECRET"),
		AuthTokenTTL:    getDurationEnv("AUTH_TOKEN_TTL", 15*time.Minute),
		RateLimitRPS:    getFloatEnv("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  getIntEnv("RATE_LIMIT_BURST", 20),
		IdempotencyTTL:  getDurationEnv("IDEMPOTENCY_TTL", 5*time.Minute),
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q (want debug|info|warn|error)", c.LogLevel)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be > 0, got %v", c.ShutdownTimeout)
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT must be > 0, got %v", c.RequestTimeout)
	}
	if c.AuthTokenTTL <= 0 {
		return fmt.Errorf("AUTH_TOKEN_TTL must be > 0, got %v", c.AuthTokenTTL)
	}
	if c.RateLimitRPS < 0 {
		return fmt.Errorf("RATE_LIMIT_RPS must be >= 0, got %v", c.RateLimitRPS)
	}
	if c.RateLimitBurst <= 0 {
		return fmt.Errorf("RATE_LIMIT_BURST must be > 0, got %v", c.RateLimitBurst)
	}
	if c.IdempotencyTTL <= 0 {
		return fmt.Errorf("IDEMPOTENCY_TTL must be > 0, got %v", c.IdempotencyTTL)
	}
	return nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	// 정수(초) 또는 "15s"/"500ms" 형식 모두 허용.
	if n, err := strconv.Atoi(raw); err == nil {
		return time.Duration(n) * time.Second
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	return def
}

func getIntEnv(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}
	return def
}

func getFloatEnv(key string, def float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	return def
}
