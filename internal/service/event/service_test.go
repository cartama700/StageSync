package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/event"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	eventsvc "github.com/kimsehoon/stagesync/internal/service/event"
)

// fixedClock — 테스트에서 Now() 를 통제.
func fixedClock(ts time.Time) func() time.Time { return func() time.Time { return ts } }

// newSvcAt — 지정된 "현재" 시각으로 서비스 생성.
func newSvcAt(now time.Time) *eventsvc.Service {
	repo := inmem.NewEventRepo()
	return eventsvc.NewService(repo, eventsvc.WithNow(fixedClock(now)))
}

func TestCreate_InvalidWindow(t *testing.T) {
	t.Parallel()
	svc := newSvcAt(time.Now())
	start := time.Now()
	_, err := svc.Create(context.Background(), eventsvc.CreateInput{
		ID: "e1", Name: "E1", StartAt: start, EndAt: start, // start == end
	})
	require.ErrorIs(t, err, domain.ErrInvalidWindow)
}

func TestCreate_Duplicate(t *testing.T) {
	t.Parallel()
	svc := newSvcAt(time.Now())
	ctx := context.Background()
	in := eventsvc.CreateInput{
		ID: "e1", Name: "E1",
		StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
	}
	_, err := svc.Create(ctx, in)
	require.NoError(t, err)
	_, err = svc.Create(ctx, in)
	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestCreate_WithRewards(t *testing.T) {
	t.Parallel()
	svc := newSvcAt(time.Now())
	ctx := context.Background()
	_, err := svc.Create(ctx, eventsvc.CreateInput{
		ID: "e1", Name: "E1",
		StartAt: time.Now(), EndAt: time.Now().Add(time.Hour),
		Rewards: []domain.RewardTier{
			{Tier: 1, MinPoints: 100, RewardName: "gem x10"},
			{Tier: 2, MinPoints: 500, RewardName: "gem x50"},
		},
	})
	require.NoError(t, err)
	view, err := svc.GetRewards(ctx, "e1", "p1")
	require.NoError(t, err)
	require.Len(t, view.Tiers, 2)
}

func TestStatusTransition(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	tests := []struct {
		name   string
		now    time.Time
		expect domain.Status
	}{
		{name: "before start", now: start.Add(-time.Minute), expect: domain.StatusUpcoming},
		{name: "at start", now: start, expect: domain.StatusOngoing},
		{name: "middle", now: start.Add(30 * time.Minute), expect: domain.StatusOngoing},
		{name: "at end", now: end, expect: domain.StatusOngoing},
		{name: "after end", now: end.Add(time.Minute), expect: domain.StatusEnded},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newSvcAt(tc.now)
			ctx := context.Background()
			_, err := svc.Create(ctx, eventsvc.CreateInput{
				ID: "e1", Name: "E1", StartAt: start, EndAt: end,
			})
			require.NoError(t, err)
			_, st, err := svc.Get(ctx, "e1")
			require.NoError(t, err)
			require.Equal(t, tc.expect, st)
		})
	}
}

func TestAddScore_OngoingOnly(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	tests := []struct {
		name    string
		now     time.Time
		wantErr error
	}{
		{name: "upcoming", now: start.Add(-time.Minute), wantErr: domain.ErrNotOngoing},
		{name: "ongoing", now: start.Add(30 * time.Minute), wantErr: nil},
		{name: "ended", now: end.Add(time.Minute), wantErr: domain.ErrNotOngoing},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := newSvcAt(tc.now)
			ctx := context.Background()
			_, err := svc.Create(ctx, eventsvc.CreateInput{
				ID: "e1", Name: "E1", StartAt: start, EndAt: end,
			})
			require.NoError(t, err)
			_, err = svc.AddScore(ctx, "e1", "p1", 100)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestAddScore_Accumulates(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	svc := newSvcAt(start.Add(10 * time.Minute))
	ctx := context.Background()
	_, err := svc.Create(ctx, eventsvc.CreateInput{
		ID: "e1", Name: "E1", StartAt: start, EndAt: start.Add(time.Hour),
	})
	require.NoError(t, err)

	_, err = svc.AddScore(ctx, "e1", "p1", 100)
	require.NoError(t, err)
	sc, err := svc.AddScore(ctx, "e1", "p1", 250)
	require.NoError(t, err)
	require.EqualValues(t, 350, sc.Points)
}

func TestAddScore_InvalidDelta(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	svc := newSvcAt(start.Add(10 * time.Minute))
	ctx := context.Background()
	_, err := svc.Create(ctx, eventsvc.CreateInput{
		ID: "e1", Name: "E1", StartAt: start, EndAt: start.Add(time.Hour),
	})
	require.NoError(t, err)

	for _, d := range []int64{0, -1, 2_000_000} {
		_, err := svc.AddScore(ctx, "e1", "p1", d)
		require.ErrorIs(t, err, domain.ErrInvalidDelta, "delta=%d", d)
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	svc := newSvcAt(time.Now())
	_, _, err := svc.Get(context.Background(), "missing")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetRewards_EligibilityEndToEnd(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	// 공유 repo + 서로 다른 clock 두 개.
	repo := inmem.NewEventRepo()
	liveSvc := eventsvc.NewService(repo, eventsvc.WithNow(fixedClock(start.Add(5*time.Minute))))
	postSvc := eventsvc.NewService(repo, eventsvc.WithNow(fixedClock(end.Add(time.Minute))))
	ctx := context.Background()

	_, err := liveSvc.Create(ctx, eventsvc.CreateInput{
		ID: "e1", Name: "E1", StartAt: start, EndAt: end,
		Rewards: []domain.RewardTier{
			{Tier: 1, MinPoints: 100, RewardName: "r1"},
			{Tier: 2, MinPoints: 500, RewardName: "r2"},
			{Tier: 3, MinPoints: 1000, RewardName: "r3"},
		},
	})
	require.NoError(t, err)

	_, err = liveSvc.AddScore(ctx, "e1", "p1", 600)
	require.NoError(t, err)

	view, err := postSvc.GetRewards(ctx, "e1", "p1")
	require.NoError(t, err)
	require.Equal(t, domain.StatusEnded, view.Status)
	require.True(t, view.Claimable)
	require.EqualValues(t, 600, view.Points)
	require.Len(t, view.Tiers, 3)
	require.Len(t, view.Eligible, 2) // 100, 500 만 달성
}

func TestListCurrent(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	svc := newSvcAt(now)
	ctx := context.Background()

	// 진행 중.
	_, err := svc.Create(ctx, eventsvc.CreateInput{
		ID: "ongoing", Name: "O", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour),
	})
	require.NoError(t, err)
	// 예정.
	_, err = svc.Create(ctx, eventsvc.CreateInput{
		ID: "upcoming", Name: "U", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour),
	})
	require.NoError(t, err)
	// 종료.
	_, err = svc.Create(ctx, eventsvc.CreateInput{
		ID: "ended", Name: "E", StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour),
	})
	require.NoError(t, err)

	list, err := svc.ListCurrent(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "ongoing", list[0].ID)
}
