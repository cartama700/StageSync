package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/idempotency"
)

func newMiniredisStore(t *testing.T, ttl time.Duration) (*idempotency.RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return idempotency.NewRedisStore(client, ttl), mr
}

func TestRedisStore_GetSet_Roundtrip(t *testing.T) {
	t.Parallel()
	s, _ := newMiniredisStore(t, 5*time.Minute)
	ctx := context.Background()

	_, ok, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 201, Body: []byte(`{"ok":true}`)}))
	e, ok, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 201, e.Status)
	require.JSONEq(t, `{"ok":true}`, string(e.Body))
}

func TestRedisStore_SetIsNX(t *testing.T) {
	t.Parallel()
	s, _ := newMiniredisStore(t, 5*time.Minute)
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 200, Body: []byte("first")}))
	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 500, Body: []byte("second")}))

	e, ok, _ := s.Get(ctx, "k1")
	require.True(t, ok)
	require.Equal(t, 200, e.Status, "SET NX 는 첫 번째 값 유지")
	require.Equal(t, []byte("first"), e.Body)
}

func TestRedisStore_Expiration(t *testing.T) {
	t.Parallel()
	s, mr := newMiniredisStore(t, 1*time.Second)
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 200, Body: []byte("x")}))

	// miniredis 의 FastForward 로 TTL 가속.
	mr.FastForward(2 * time.Second)

	_, ok, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.False(t, ok, "TTL 초과 후 키 없음")
}
