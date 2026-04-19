package redis_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
	redisrepo "github.com/kimsehoon/stagesync/internal/persistence/redis"
)

// newMiniredis — 각 테스트용 isolated in-memory Redis + client.
// 실 Redis 대신 miniredis 를 써도 ZSET 명령은 호환.
func newMiniredis(t *testing.T) (*redisrepo.Leaderboard, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t) // t.Cleanup 자동 등록됨
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return redisrepo.NewLeaderboard(client), mr
}

// seed — inmem 테스트와 동일한 픽스처.
func seed(t *testing.T, lb *redisrepo.Leaderboard) {
	t.Helper()
	ctx := context.Background()
	for _, s := range []struct {
		player string
		delta  int64
	}{
		{"p1", 500},
		{"p2", 300},
		{"p3", 300},
		{"p4", 100},
	} {
		_, err := lb.IncrBy(ctx, "ev1", s.player, s.delta)
		require.NoError(t, err)
	}
}

func TestRedisLeaderboard_IncrBy_Accumulates(t *testing.T) {
	t.Parallel()
	lb, _ := newMiniredis(t)
	ctx := context.Background()

	got, err := lb.IncrBy(ctx, "ev1", "p1", 100)
	require.NoError(t, err)
	require.EqualValues(t, 100, got)

	got, err = lb.IncrBy(ctx, "ev1", "p1", 50)
	require.NoError(t, err)
	require.EqualValues(t, 150, got)

	got, err = lb.IncrBy(ctx, "ev2", "p1", 999)
	require.NoError(t, err)
	require.EqualValues(t, 999, got, "다른 이벤트는 독립 ZSET")
}

func TestRedisLeaderboard_Top(t *testing.T) {
	t.Parallel()
	lb, _ := newMiniredis(t)
	seed(t, lb)
	ctx := context.Background()

	top3, err := lb.Top(ctx, "ev1", 3)
	require.NoError(t, err)
	require.Len(t, top3, 3)

	require.Equal(t, "p1", top3[0].PlayerID)
	require.EqualValues(t, 500, top3[0].Score)
	require.Equal(t, 1, top3[0].Rank)
	// 동점 p2/p3 는 ZREVRANGE 기본 (lex DESC) → p3 가 먼저.
	require.Equal(t, "p3", top3[1].PlayerID)
	require.Equal(t, "p2", top3[2].PlayerID)
}

func TestRedisLeaderboard_Top_LargerThanSize(t *testing.T) {
	t.Parallel()
	lb, _ := newMiniredis(t)
	seed(t, lb)
	got, err := lb.Top(context.Background(), "ev1", 100)
	require.NoError(t, err)
	require.Len(t, got, 4)
}

func TestRedisLeaderboard_Rank(t *testing.T) {
	t.Parallel()
	lb, _ := newMiniredis(t)
	seed(t, lb)
	ctx := context.Background()

	// p3 는 동점 p2 보다 lex DESC 로 앞이라 rank 2.
	e, err := lb.Rank(ctx, "ev1", "p3")
	require.NoError(t, err)
	require.Equal(t, "p3", e.PlayerID)
	require.EqualValues(t, 300, e.Score)
	require.Equal(t, 2, e.Rank)

	_, err = lb.Rank(ctx, "ev1", "ghost")
	require.ErrorIs(t, err, domain.ErrPlayerNotRanked)
}

func TestRedisLeaderboard_Around(t *testing.T) {
	t.Parallel()
	lb, _ := newMiniredis(t)
	seed(t, lb)
	ctx := context.Background()

	// 순위: p1(1) · p3(2) · p2(3) · p4(4).
	around, err := lb.Around(ctx, "ev1", "p2", 1)
	require.NoError(t, err)
	require.Equal(t, []string{"p3", "p2", "p4"}, extractIDs(around))

	// Top clamp.
	around, err = lb.Around(ctx, "ev1", "p1", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p3", "p2"}, extractIDs(around))

	// 꼴찌 clamp.
	around, err = lb.Around(ctx, "ev1", "p4", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"p3", "p2", "p4"}, extractIDs(around))

	// radius=0 본인만.
	around, err = lb.Around(ctx, "ev1", "p2", 0)
	require.NoError(t, err)
	require.Equal(t, []string{"p2"}, extractIDs(around))

	// 미등재 → ErrPlayerNotRanked.
	_, err = lb.Around(ctx, "ev1", "ghost", 2)
	require.ErrorIs(t, err, domain.ErrPlayerNotRanked)
}

func extractIDs(entries []domain.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.PlayerID
	}
	return out
}
