package inmem_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
)

// seedLeaderboard — 공통 테스트 픽스처.
// 점수 순: p1(500) > p3(300) > p2(300) > p4(100)
// 동점 p2/p3 는 Redis ZREVRANGE 규칙 (lex DESC) 에 맞춰 p3 가 먼저.
func seedLeaderboard(t *testing.T) *inmem.Leaderboard {
	t.Helper()
	lb := inmem.NewLeaderboard()
	ctx := context.Background()
	for _, seed := range []struct {
		player string
		delta  int64
	}{
		{"p1", 500},
		{"p2", 300},
		{"p3", 300},
		{"p4", 100},
	} {
		_, err := lb.IncrBy(ctx, "ev1", seed.player, seed.delta)
		require.NoError(t, err)
	}
	return lb
}

func TestInmemLeaderboard_IncrBy_Accumulates(t *testing.T) {
	t.Parallel()
	lb := inmem.NewLeaderboard()
	ctx := context.Background()

	got, err := lb.IncrBy(ctx, "ev1", "p1", 100)
	require.NoError(t, err)
	require.EqualValues(t, 100, got)

	got, err = lb.IncrBy(ctx, "ev1", "p1", 50)
	require.NoError(t, err)
	require.EqualValues(t, 150, got, "누적되어야 함")

	// 다른 이벤트는 독립.
	got, err = lb.IncrBy(ctx, "ev2", "p1", 999)
	require.NoError(t, err)
	require.EqualValues(t, 999, got)
}

func TestInmemLeaderboard_Top(t *testing.T) {
	t.Parallel()
	lb := seedLeaderboard(t)
	ctx := context.Background()

	top3, err := lb.Top(ctx, "ev1", 3)
	require.NoError(t, err)
	require.Len(t, top3, 3)

	require.Equal(t, "p1", top3[0].PlayerID)
	require.EqualValues(t, 500, top3[0].Score)
	require.Equal(t, 1, top3[0].Rank)

	// p2 / p3 동점 — ZREVRANGE 규칙 (lex DESC) 으로 p3 가 2위.
	require.Equal(t, "p3", top3[1].PlayerID)
	require.Equal(t, 2, top3[1].Rank)
	require.Equal(t, "p2", top3[2].PlayerID)
	require.Equal(t, 3, top3[2].Rank)
}

func TestInmemLeaderboard_Top_LargerThanSize(t *testing.T) {
	t.Parallel()
	lb := seedLeaderboard(t)
	got, err := lb.Top(context.Background(), "ev1", 100)
	require.NoError(t, err)
	require.Len(t, got, 4, "사이즈 이상 요청해도 전체만 반환")
}

func TestInmemLeaderboard_Top_EmptyEvent(t *testing.T) {
	t.Parallel()
	lb := inmem.NewLeaderboard()
	got, err := lb.Top(context.Background(), "no-such-event", 10)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestInmemLeaderboard_Rank(t *testing.T) {
	t.Parallel()
	lb := seedLeaderboard(t)
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

func TestInmemLeaderboard_Around(t *testing.T) {
	t.Parallel()
	lb := seedLeaderboard(t)
	ctx := context.Background()

	// 순위: p1(1) · p3(2) · p2(3) · p4(4).
	// p2 (rank 3) 주변 ±1 → [p3, p2, p4].
	around, err := lb.Around(ctx, "ev1", "p2", 1)
	require.NoError(t, err)
	require.Equal(t, []string{"p3", "p2", "p4"}, extractIDs(around))

	// p1 (rank 1) ±2 → [p1, p3, p2] (좌측 clamp).
	around, err = lb.Around(ctx, "ev1", "p1", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"p1", "p3", "p2"}, extractIDs(around))

	// p4 (rank 4, 마지막) ±2 → [p3, p2, p4] (우측 clamp).
	around, err = lb.Around(ctx, "ev1", "p4", 2)
	require.NoError(t, err)
	require.Equal(t, []string{"p3", "p2", "p4"}, extractIDs(around))

	// radius=0 → 본인만.
	around, err = lb.Around(ctx, "ev1", "p2", 0)
	require.NoError(t, err)
	require.Equal(t, []string{"p2"}, extractIDs(around))
}

func TestInmemLeaderboard_Around_NotRanked(t *testing.T) {
	t.Parallel()
	lb := seedLeaderboard(t)
	_, err := lb.Around(context.Background(), "ev1", "ghost", 2)
	require.ErrorIs(t, err, domain.ErrPlayerNotRanked)
}

func extractIDs(entries []domain.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.PlayerID
	}
	return out
}
