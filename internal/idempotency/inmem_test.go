package idempotency_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/idempotency"
)

func TestInmemStore_GetSet_Roundtrip(t *testing.T) {
	t.Parallel()
	s := idempotency.NewInmemStore(5 * time.Minute)
	ctx := context.Background()

	// 미스.
	e, ok, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, e)

	// Set + Get.
	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 201, Body: []byte(`{"ok":true}`)}))
	e, ok, err = s.Get(ctx, "k1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 201, e.Status)
	require.JSONEq(t, `{"ok":true}`, string(e.Body))
}

func TestInmemStore_SetIsNX(t *testing.T) {
	t.Parallel()
	s := idempotency.NewInmemStore(5 * time.Minute)
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 200, Body: []byte("first")}))
	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 500, Body: []byte("second")}))

	e, ok, _ := s.Get(ctx, "k1")
	require.True(t, ok)
	require.Equal(t, 200, e.Status, "첫 번째 Set 이 유지되어야 함 (NX 시맨틱)")
	require.Equal(t, []byte("first"), e.Body)
}

func TestInmemStore_ExpiredEntry_LazyDeleted(t *testing.T) {
	t.Parallel()
	s := idempotency.NewInmemStore(100 * time.Millisecond)
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k1", idempotency.Entry{Status: 200, Body: []byte("x")}))
	time.Sleep(150 * time.Millisecond)

	_, ok, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.False(t, ok, "TTL 초과 엔트리는 Get 에서 lazy 삭제")

	// Sweep 을 한 번 더 호출해도 에러 없음.
	_ = s.Sweep()
}

func TestInmemStore_Sweep(t *testing.T) {
	t.Parallel()
	s := idempotency.NewInmemStore(50 * time.Millisecond)
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "a", idempotency.Entry{Status: 200}))
	require.NoError(t, s.Set(ctx, "b", idempotency.Entry{Status: 200}))

	time.Sleep(100 * time.Millisecond)
	removed := s.Sweep()
	require.Equal(t, 2, removed)

	_, ok, _ := s.Get(ctx, "a")
	require.False(t, ok)
}
