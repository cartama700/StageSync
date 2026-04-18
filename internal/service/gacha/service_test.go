package gacha_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	gachasvc "github.com/kimsehoon/stagesync/internal/service/gacha"
)

// fixedPityRepo — 초기 pity 값을 지정할 수 있는 mock.
// 천장 트리거 테스트 전용.
type fixedPityRepo struct {
	pity           int
	insertedRolls  []*domain.Roll
	lastNewCounter int
}

func (r *fixedPityRepo) GetPity(_ context.Context, _, _ string) (int, error) {
	return r.pity, nil
}

func (r *fixedPityRepo) InsertRollsAndUpdatePity(_ context.Context, rolls []*domain.Roll, _, _ string, newCounter int) error {
	r.insertedRolls = append(r.insertedRolls, rolls...)
	r.lastNewCounter = newCounter
	return nil
}

func (r *fixedPityRepo) ListRollsByPlayer(_ context.Context, _ string, _ int) ([]*domain.Roll, error) {
	return r.insertedRolls, nil
}

// failingRepo — InsertRolls... 에서 강제 실패. 트랜잭션 롤백 계열 테스트 (service 단).
type failingRepo struct{}

func (failingRepo) GetPity(_ context.Context, _, _ string) (int, error) { return 0, nil }
func (failingRepo) InsertRollsAndUpdatePity(_ context.Context, _ []*domain.Roll, _, _ string, _ int) error {
	return errors.New("simulated repo failure")
}
func (failingRepo) ListRollsByPlayer(_ context.Context, _ string, _ int) ([]*domain.Roll, error) {
	return nil, nil
}

// newTestService — 결정적 RNG (고정 seed) + inmem repo + 데모 풀.
func newTestService(t *testing.T) *gachasvc.Service {
	t.Helper()
	repo := inmem.NewGachaRepo()
	pools := gachasvc.NewStaticPoolRegistry()
	rng := rand.New(rand.NewPCG(42, 0xCAFEBABE))
	return gachasvc.NewService(repo, pools, gachasvc.WithRand(rng))
}

// TestRoll_InvalidCount — 허용 범위 밖 count 는 ErrInvalidCount.
func TestRoll_InvalidCount(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		count int
	}{
		{name: "zero", count: 0},
		{name: "negative", count: -1},
		{name: "over ten", count: 11},
		{name: "very large", count: 1000},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.Roll(ctx, "p1", gachasvc.DemoPoolID, tc.count)
			require.ErrorIs(t, err, domain.ErrInvalidCount)
		})
	}
}

// TestRoll_UnknownPool — 알 수 없는 풀 → ErrPoolNotFound.
func TestRoll_UnknownPool(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	_, err := svc.Roll(context.Background(), "p1", "unknown-pool", 1)
	require.ErrorIs(t, err, domain.ErrPoolNotFound)
}

// TestRoll_Basic — 정상 10-roll → 10개 결과 + 모두 데모 풀 카드.
func TestRoll_Basic(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	rolls, err := svc.Roll(ctx, "p1", gachasvc.DemoPoolID, 10)
	require.NoError(t, err)
	require.Len(t, rolls, 10)
	for _, r := range rolls {
		require.Equal(t, "p1", r.PlayerID)
		require.Equal(t, gachasvc.DemoPoolID, r.PoolID)
		require.True(t, r.Rarity.IsValid())
		require.NotEmpty(t, r.ID)
		require.False(t, r.PulledAt.IsZero())
	}
}

// TestRoll_PityTrigger — pity=79 상태에서 1 roll → 80번째 = 천장 확정 SSR.
func TestRoll_PityTrigger(t *testing.T) {
	t.Parallel()

	repo := &fixedPityRepo{pity: 79} // 다음 roll 이 80번째 → 데모 풀 threshold=80
	pools := gachasvc.NewStaticPoolRegistry()
	svc := gachasvc.NewService(repo, pools)

	rolls, err := svc.Roll(context.Background(), "p1", gachasvc.DemoPoolID, 1)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
	require.Equal(t, domain.RaritySSR, rolls[0].Rarity, "천장 트리거 시 SSR 확정")
	require.True(t, rolls[0].IsPity, "IsPity 플래그 set")
	require.Equal(t, 0, repo.lastNewCounter, "천장 적용 후 카운터 리셋")
}

// TestRoll_PityNotTriggered — pity 가 threshold 미달이면 일반 roll.
func TestRoll_PityNotTriggered(t *testing.T) {
	t.Parallel()

	repo := &fixedPityRepo{pity: 10}
	pools := gachasvc.NewStaticPoolRegistry()
	// 결정적 RNG 로 첫 roll 이 SSR 이 나오지 않도록 고정 seed 확인 (실패하면 seed 조정).
	rng := rand.New(rand.NewPCG(1, 1))
	svc := gachasvc.NewService(repo, pools, gachasvc.WithRand(rng))

	rolls, err := svc.Roll(context.Background(), "p1", gachasvc.DemoPoolID, 1)
	require.NoError(t, err)
	require.Len(t, rolls, 1)
	require.False(t, rolls[0].IsPity, "일반 roll 이어야 함")
}

// TestRoll_ServiceBubblesRepoError — repo 실패 시 service 가 에러 전파.
func TestRoll_ServiceBubblesRepoError(t *testing.T) {
	t.Parallel()
	pools := gachasvc.NewStaticPoolRegistry()
	svc := gachasvc.NewService(failingRepo{}, pools)

	_, err := svc.Roll(context.Background(), "p1", gachasvc.DemoPoolID, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated repo failure")
}

// TestRoll_Distribution — 10000 unique 플레이어 × 1 roll 로 pity 미개입 상태에서
// rarity 분포가 선언 가중치 (SSR 3%, SR 17%, R 80%) 의 ±5% 이내인지 검증.
func TestRoll_Distribution(t *testing.T) {
	t.Parallel()

	repo := inmem.NewGachaRepo()
	pools := gachasvc.NewStaticPoolRegistry()
	rng := rand.New(rand.NewPCG(1, 2)) // 고정 seed
	svc := gachasvc.NewService(repo, pools, gachasvc.WithRand(rng))

	const N = 10000
	counts := map[domain.Rarity]int{}
	ctx := context.Background()
	for i := 0; i < N; i++ {
		rolls, err := svc.Roll(ctx, fmt.Sprintf("p%d", i), gachasvc.DemoPoolID, 1)
		require.NoError(t, err)
		require.Len(t, rolls, 1)
		counts[rolls[0].Rarity]++
	}

	// 선언 가중치: R 800/1000, SR 170/1000, SSR 30/1000.
	tolerance := 0.05 // ±5%
	checks := []struct {
		rarity   domain.Rarity
		expected float64
	}{
		{domain.RarityR, 0.80},
		{domain.RaritySR, 0.17},
		{domain.RaritySSR, 0.03},
	}
	for _, c := range checks {
		actual := float64(counts[c.rarity]) / float64(N)
		delta := actual - c.expected
		if delta < 0 {
			delta = -delta
		}
		require.Lessf(t, delta, tolerance,
			"%s: actual=%.4f expected=%.4f (허용 ±%.2f)", c.rarity, actual, c.expected, tolerance)
	}
}

// TestListHistory_BasicLimit — 이력 조회 + limit 동작.
func TestListHistory_BasicLimit(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Roll(ctx, "p1", gachasvc.DemoPoolID, 10)
	require.NoError(t, err)

	// 기본 limit (10)
	history, err := svc.ListHistory(ctx, "p1", 10)
	require.NoError(t, err)
	require.Len(t, history, 10)

	// 작은 limit
	history, err = svc.ListHistory(ctx, "p1", 3)
	require.NoError(t, err)
	require.Len(t, history, 3)

	// 다른 플레이어 → 빈 배열
	history, err = svc.ListHistory(ctx, "p_other", 10)
	require.NoError(t, err)
	require.Empty(t, history)
}

// TestGetPity_DefaultZero — 처음 뽑지 않은 플레이어의 pity 는 0.
func TestGetPity_DefaultZero(t *testing.T) {
	t.Parallel()
	svc := newTestService(t)

	counter, err := svc.GetPity(context.Background(), "newbie", gachasvc.DemoPoolID)
	require.NoError(t, err)
	require.Equal(t, 0, counter)
}
