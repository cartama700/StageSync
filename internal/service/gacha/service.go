// Package gacha — ガチャ 비즈니스 로직.
// 가중치 RNG · 천장 시스템 · 원자적 트랜잭션.
package gacha

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
)

// Repository — Service 가 요구하는 저장소 인터페이스.
// InsertRollsAndUpdatePity 는 **원자적** 이어야 함 (MySQL 은 tx, inmem 은 mutex).
type Repository interface {
	// GetPity — (playerID, poolID) 의 천장 카운터. 없으면 0 반환.
	GetPity(ctx context.Context, playerID, poolID string) (int, error)

	// InsertRollsAndUpdatePity — rolls 를 모두 INSERT + pity 를 newCounter 로 UPSERT.
	// 반드시 단일 원자 트랜잭션 (실패 시 전체 롤백).
	InsertRollsAndUpdatePity(ctx context.Context, rolls []*domain.Roll, playerID, poolID string, newCounter int) error

	// ListRollsByPlayer — 최신 순으로 최대 limit 건.
	ListRollsByPlayer(ctx context.Context, playerID string, limit int) ([]*domain.Roll, error)
}

// PoolRegistry — 풀 조회. Phase 5 는 하드코딩, Phase 5b 에서 YAML 로 전환.
type PoolRegistry interface {
	GetPool(poolID string) (*domain.Pool, error)
}

// Service — 가챠 서비스.
type Service struct {
	repo  Repository
	pools PoolRegistry

	rngMu sync.Mutex
	rng   *rand.Rand
	now   func() time.Time
}

// Option — 선택적 생성자 파라미터 (테스트에서 RNG · 시계 주입).
type Option func(*Service)

// WithRand — 결정적 RNG 주입 (테스트 전용).
func WithRand(r *rand.Rand) Option {
	return func(s *Service) { s.rng = r }
}

// WithNow — 시계 함수 주입 (테스트 전용).
func WithNow(fn func() time.Time) Option {
	return func(s *Service) { s.now = fn }
}

// NewService — 의존성 주입 + 기본 RNG (time.Now 시드 PCG) + 실시간 clock.
func NewService(repo Repository, pools PoolRegistry, opts ...Option) *Service {
	// #nosec G404 — 게임 가챠는 암호학적 RNG 불필요. PCG 면 충분.
	defaultRng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0xDEADBEEF))
	s := &Service{
		repo:  repo,
		pools: pools,
		rng:   defaultRng,
		now:   time.Now,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Roll — count 회 뽑기 실행 + 천장 적용 + 원자적 저장.
func (s *Service) Roll(ctx context.Context, playerID, poolID string, count int) ([]*domain.Roll, error) {
	if count < 1 || count > 10 {
		return nil, domain.ErrInvalidCount
	}

	pool, err := s.pools.GetPool(poolID)
	if err != nil {
		return nil, fmt.Errorf("pools.GetPool: %w", err)
	}
	if len(pool.Cards) == 0 {
		return nil, domain.ErrEmptyPool
	}

	pity, err := s.repo.GetPity(ctx, playerID, poolID)
	if err != nil {
		return nil, fmt.Errorf("repo.GetPity: %w", err)
	}

	rolls := make([]*domain.Roll, 0, count)
	now := s.now()

	for i := 0; i < count; i++ {
		pity++
		var (
			card   domain.Card
			isPity bool
		)
		if pool.PityThreshold > 0 && pity >= pool.PityThreshold {
			// 천장 — SSR 중에서 가중치 뽑기.
			ssrCards := domain.FilterByRarity(pool.Cards, domain.RaritySSR)
			if len(ssrCards) == 0 {
				// SSR 카드 자체가 없는 풀 — 정상 뽑기로 대체.
				card = domain.WeightedPick(s.pick, pool.Cards)
			} else {
				card = domain.WeightedPick(s.pick, ssrCards)
				isPity = true
			}
			pity = 0
		} else {
			card = domain.WeightedPick(s.pick, pool.Cards)
			if card.Rarity == domain.RaritySSR {
				// 자연 SSR — 천장 리셋.
				pity = 0
			}
		}

		id, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("uuid.NewV7: %w", err)
		}
		rolls = append(rolls, &domain.Roll{
			ID:       id.String(),
			PlayerID: playerID,
			PoolID:   poolID,
			CardID:   card.ID,
			Rarity:   card.Rarity,
			IsPity:   isPity,
			PulledAt: now,
		})
	}

	if err := s.repo.InsertRollsAndUpdatePity(ctx, rolls, playerID, poolID, pity); err != nil {
		return nil, fmt.Errorf("repo.InsertRollsAndUpdatePity: %w", err)
	}
	return rolls, nil
}

// ListHistory — 플레이어 최근 뽑기 이력.
func (s *Service) ListHistory(ctx context.Context, playerID string, limit int) ([]*domain.Roll, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	rolls, err := s.repo.ListRollsByPlayer(ctx, playerID, limit)
	if err != nil {
		return nil, fmt.Errorf("repo.ListRollsByPlayer: %w", err)
	}
	return rolls, nil
}

// GetPity — 천장 카운터 조회.
func (s *Service) GetPity(ctx context.Context, playerID, poolID string) (int, error) {
	pity, err := s.repo.GetPity(ctx, playerID, poolID)
	if err != nil {
		return 0, fmt.Errorf("repo.GetPity: %w", err)
	}
	return pity, nil
}

// pick — mutex 로 보호된 RNG. domain.WeightedPick 에 `RandIntN` 으로 전달.
func (s *Service) pick(n int) int {
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.IntN(n)
}
