package battle_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/battle"
	battlesvc "github.com/kimsehoon/stagesync/internal/service/battle"
)

// mockRepo — ApplyDamageNaive 가 sleep 을 포함해서 "느린 DB" 시뮬.
// 동시 실행 감지 위해 inFlight 카운터 + maxInFlight 기록.
type mockRepo struct {
	mu          sync.Mutex
	hpByPlayer  map[string]int
	inFlight    int32
	maxInFlight int32
	sleep       time.Duration
}

func newMockRepo(sleep time.Duration) *mockRepo {
	return &mockRepo{
		hpByPlayer: map[string]int{},
		sleep:      sleep,
	}
}

func (r *mockRepo) ApplyDamageNaive(_ context.Context, playerID string, damage int) (int, error) {
	// inFlight 증가 + 최대치 기록.
	cur := atomic.AddInt32(&r.inFlight, 1)
	defer atomic.AddInt32(&r.inFlight, -1)
	for {
		prev := atomic.LoadInt32(&r.maxInFlight)
		if cur <= prev || atomic.CompareAndSwapInt32(&r.maxInFlight, prev, cur) {
			break
		}
	}

	time.Sleep(r.sleep)

	r.mu.Lock()
	defer r.mu.Unlock()
	hp, ok := r.hpByPlayer[playerID]
	if !ok {
		hp = domain.DefaultInitialHP
	}
	hp -= damage
	if hp < 0 {
		hp = 0
	}
	r.hpByPlayer[playerID] = hp
	return hp, nil
}

func (r *mockRepo) Get(_ context.Context, playerID string) (*domain.PlayerHP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hp, ok := r.hpByPlayer[playerID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.PlayerHP{PlayerID: playerID, HP: hp}, nil
}

func (r *mockRepo) Reset(_ context.Context, playerID string, hp int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hpByPlayer[playerID] = hp
	return nil
}

// ----- 기본 기능 테스트 -----

func TestV1Naive_InvalidDamage(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(0)
	svc := battlesvc.NewV1Naive(repo)

	tests := []int{0, -1, domain.MaxDamagePerRequest + 1}
	for _, dmg := range tests {
		_, err := svc.Apply(context.Background(), "p1", dmg)
		require.ErrorIs(t, err, domain.ErrInvalidDamage, "damage=%d", dmg)
	}
}

func TestV1Naive_Basic(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(0)
	svc := battlesvc.NewV1Naive(repo)
	ctx := context.Background()

	hp, err := svc.Apply(ctx, "p1", 100)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultInitialHP-100, hp)

	hp, err = svc.Apply(ctx, "p1", 50)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultInitialHP-150, hp)
}

func TestV2UserQueue_Basic(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(0)
	svc := battlesvc.NewV2UserQueue(repo)
	ctx := context.Background()

	hp, err := svc.Apply(ctx, "p1", 100)
	require.NoError(t, err)
	require.Equal(t, domain.DefaultInitialHP-100, hp)
}

// ----- 핵심 서사 테스트 -----

// TestV1Naive_AllowsConcurrentDBCalls —
// V1 은 모든 요청을 동시에 DB 로 보냄 → inFlight > 1 이 관찰됨.
// 즉, **DB 레벨의 락** 에 의존한다 — 이게 FOR UPDATE 패턴의 본질.
func TestV1Naive_AllowsConcurrentDBCalls(t *testing.T) {
	t.Parallel()
	// sleep 으로 동시성 가시화 (짧은 race 구간 확보).
	repo := newMockRepo(30 * time.Millisecond)
	svc := battlesvc.NewV1Naive(repo)
	ctx := context.Background()

	const N = 20
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.Apply(ctx, "target", 10)
		}()
	}
	wg.Wait()

	// V1 은 동시에 다수의 DB 호출이 inFlight — 실제 MySQL 이라면 여기서 락 경합.
	// 테스트 mock 은 락 없이 단순 sleep 이므로 inFlight 가 쉽게 2+ 를 돌파함.
	require.Greater(t, atomic.LoadInt32(&repo.maxInFlight), int32(1),
		"V1-naive 는 동시 DB 호출을 허용 — 이게 MySQL 에선 락 경합의 원인")
}

// TestV2UserQueue_SerializesDBCalls —
// V2 는 같은 playerID 의 요청을 **단일 워커** 로 직렬화 →
// repo 의 inFlight 는 절대 1 을 넘지 않음 (DB 에 한 번에 한 요청만).
//
// **이것이 v1 vs v2 의 핵심 차이** — 락 경합을 DB 레벨 → Go 레벨로 이동.
func TestV2UserQueue_SerializesDBCalls(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(10 * time.Millisecond)
	svc := battlesvc.NewV2UserQueue(repo)
	ctx := context.Background()

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = svc.Apply(ctx, "target", 1)
		}()
	}
	wg.Wait()

	require.Equal(t, int32(1), atomic.LoadInt32(&repo.maxInFlight),
		"V2-queue 는 playerID 별 단일 워커 → DB 호출이 항상 순차")

	// 데미지 N 회 누적 확인 — 순차 처리라도 최종 HP 는 N 번 차감된 상태여야 함.
	hp, err := repo.Get(ctx, "target")
	require.NoError(t, err)
	require.Equal(t, domain.DefaultInitialHP-N, hp.HP)
}

// TestV2UserQueue_IndependentPlayers —
// 다른 playerID 는 독립 워커 → 병렬 처리 가능 (inFlight 최대 2+).
func TestV2UserQueue_IndependentPlayers(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(20 * time.Millisecond)
	svc := battlesvc.NewV2UserQueue(repo)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_, _ = svc.Apply(ctx, "p1", 1)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_, _ = svc.Apply(ctx, "p2", 1)
		}
	}()
	wg.Wait()

	require.Greater(t, atomic.LoadInt32(&repo.maxInFlight), int32(1),
		"다른 playerID 는 독립 워커 → 병렬 실행 허용")
}

// TestV2UserQueue_CtxCancel —
// ctx 취소 시 enqueue / await 단계에서 ctx.Err 전파.
func TestV2UserQueue_CtxCancel(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(200 * time.Millisecond)
	svc := battlesvc.NewV2UserQueue(repo)

	// 한 요청 실행 중 (sleep 200ms) — 취소 ctx 로 두 번째 보내면 빨리 실패.
	go svc.Apply(context.Background(), "p1", 1)
	time.Sleep(20 * time.Millisecond) // 첫 요청이 워커에 잡혔는지 확인 대기.

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := svc.Apply(ctx, "p1", 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"ctx deadline 이 원인이어야 함: got %v", err)
}

// TestBuild_ImplementationMapping — env 매핑.
func TestBuild_ImplementationMapping(t *testing.T) {
	t.Parallel()
	repo := newMockRepo(0)

	a := battlesvc.Build(battlesvc.ImplNaive, repo)
	require.IsType(t, &battlesvc.V1Naive{}, a)

	b := battlesvc.Build(battlesvc.ImplQueue, repo)
	require.IsType(t, &battlesvc.V2UserQueue{}, b)

	c := battlesvc.Build("unknown", repo)
	require.IsType(t, &battlesvc.V1Naive{}, c, "알 수 없는 값은 naive 로 fallback")
}

// sanity — mockRepo 자체가 문제 없는지 확인.
func TestMockRepo_Sanity(t *testing.T) {
	t.Parallel()
	r := newMockRepo(0)
	ctx := context.Background()
	require.NoError(t, r.Reset(ctx, "p", 100))
	hp, err := r.ApplyDamageNaive(ctx, "p", 30)
	require.NoError(t, err)
	require.Equal(t, 70, hp)

	got, err := r.Get(ctx, "p")
	require.NoError(t, err)
	require.Equal(t, 70, got.HP)

	_, err = r.Get(ctx, "ghost")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
