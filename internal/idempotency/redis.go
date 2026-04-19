package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisStore — go-redis/v9 기반 Store.
// 다중 Pod 배포에서도 공유 상태 보장.
//
// Redis 명령:
//   - Get:   GET <key>     → JSON 역직렬화
//   - Set:   SET <key> <json> NX EX <ttl>   (NX = 키 없을 때만 저장 → race-safe)
type RedisStore struct {
	client *goredis.Client
	ttl    time.Duration
}

// NewRedisStore — 이미 연결된 client 와 ttl 로 생성.
// ttl 이 0 이하면 5 분 디폴트.
func NewRedisStore(client *goredis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &RedisStore{client: client, ttl: ttl}
}

// Get — 키가 있으면 Entry 반환.
// 키 없음 (`redis.Nil`) → nil, false, nil.
// JSON 파싱 실패 등 실제 에러는 wrap 해서 반환.
func (s *RedisStore) Get(ctx context.Context, key string) (*Entry, bool, error) {
	raw, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("redis get: %w", err)
	}
	var e Entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, false, fmt.Errorf("unmarshal idempotency entry: %w", err)
	}
	return &e, true, nil
}

// Set — `SET NX EX` 로 원자적 쓰기. 키가 이미 있으면 no-op (반환 nil).
func (s *RedisStore) Set(ctx context.Context, key string, entry Entry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal idempotency entry: %w", err)
	}
	// SetNX: set if not exists. 반환 bool 은 실제 쓰기 여부 — 쓰이지 않아도 에러 아님.
	if _, err := s.client.SetNX(ctx, key, raw, s.ttl).Result(); err != nil {
		return fmt.Errorf("redis setnx: %w", err)
	}
	return nil
}
