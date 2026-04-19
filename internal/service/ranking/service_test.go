package ranking_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	rankingsvc "github.com/kimsehoon/stagesync/internal/service/ranking"
)

// newSvc — inmem leaderboard + 시드 기반 Service.
func newSvc(t *testing.T) *rankingsvc.Service {
	t.Helper()
	lb := inmem.NewLeaderboard()
	ctx := context.Background()
	for _, s := range []struct {
		player string
		delta  int64
	}{
		{"alice", 500},
		{"bob", 300},
		{"carol", 100},
	} {
		_, err := lb.IncrBy(ctx, "ev1", s.player, s.delta)
		require.NoError(t, err)
	}
	return rankingsvc.NewService(lb)
}

func TestService_Top_DefaultN(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	// n=0 → DefaultTopN (10) 으로 확장, 하지만 실제로는 3 명만 있으니 3 개.
	got, err := svc.Top(context.Background(), "ev1", 0)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "alice", got[0].PlayerID)
}

func TestService_Top_OverMax(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	_, err := svc.Top(context.Background(), "ev1", domain.MaxTopN+1)
	require.ErrorIs(t, err, domain.ErrInvalidLimit)
}

func TestService_Rank(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	e, err := svc.Rank(context.Background(), "ev1", "bob")
	require.NoError(t, err)
	require.Equal(t, 2, e.Rank)
	require.EqualValues(t, 300, e.Score)
}

func TestService_Rank_NotFound(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	_, err := svc.Rank(context.Background(), "ev1", "ghost")
	require.ErrorIs(t, err, domain.ErrPlayerNotRanked)
}

func TestService_Around_DefaultRadius(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	// radius<0 → DefaultAroundRadius (5). 세 명이라 전체 반환.
	got, err := svc.Around(context.Background(), "ev1", "bob", -1)
	require.NoError(t, err)
	require.Len(t, got, 3)
}

func TestService_Around_OverMax(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	_, err := svc.Around(context.Background(), "ev1", "bob", domain.MaxAroundRadius+1)
	require.ErrorIs(t, err, domain.ErrInvalidLimit)
}

// TestService_Around_UnwrappedErrPlayerNotRanked —
// Store 에서 올라온 ErrPlayerNotRanked 는 wrap 없이 그대로 전파되어야 함
// (handler 가 errors.Is 로 404 로 매핑 가능하도록).
func TestService_Around_UnwrappedErrPlayerNotRanked(t *testing.T) {
	t.Parallel()
	svc := newSvc(t)
	_, err := svc.Around(context.Background(), "ev1", "ghost", 3)
	require.True(t, errors.Is(err, domain.ErrPlayerNotRanked))
}
