// Package ranking — ランキング 비즈니스 로직.
// Redis ZSET (또는 inmem fallback) 기반 실시간 랭킹 Top-N · 내 주변 조회.
package ranking

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
)

// Store — Service 가 요구하는 저장소 인터페이스.
// Redis 구현 (prod) 또는 inmem 구현 (REDIS_ADDR 미설정 시 graceful degrade) 이 만족.
// Event 서비스도 IncrBy 만 사용 — 해당 consumer 는 좁은 인터페이스를 직접 선언.
type Store interface {
	// IncrBy — (event, player) 점수를 delta 만큼 증분. 누적 후 총점 반환.
	// delta 가 음수여도 허용 (보정 용도). 호출자 책임으로 검증.
	IncrBy(ctx context.Context, eventID, playerID string, delta int64) (int64, error)

	// Top — Top-N 엔트리 (Rank 1..n, 점수 내림차순).
	// 동점 처리는 구현체 (Redis 는 사전순 · inmem 은 playerID 사전순) — 문서화된 결정적 순서.
	Top(ctx context.Context, eventID string, n int) ([]domain.Entry, error)

	// Rank — 단일 플레이어의 순위 + 점수.
	// 미등재 플레이어는 domain.ErrPlayerNotRanked.
	Rank(ctx context.Context, eventID, playerID string) (*domain.Entry, error)

	// Around — 특정 플레이어 기준 ±radius 엔트리 (본인 포함).
	// 본인이 Top 근처면 상단 윈도우가 좌측 경계 (Rank=1) 에 clamp 됨.
	Around(ctx context.Context, eventID, playerID string, radius int) ([]domain.Entry, error)
}

// Service — 랭킹 서비스. 입력 검증 + Store 호출.
type Service struct {
	store Store
}

// NewService — 의존성 주입.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Top — Top-N 조회. n 범위: [1, MaxTopN], 기본 DefaultTopN.
func (s *Service) Top(ctx context.Context, eventID string, n int) ([]domain.Entry, error) {
	if n <= 0 {
		n = domain.DefaultTopN
	}
	if n > domain.MaxTopN {
		return nil, domain.ErrInvalidLimit
	}
	entries, err := s.store.Top(ctx, eventID, n)
	if err != nil {
		return nil, fmt.Errorf("store.Top: %w", err)
	}
	return entries, nil
}

// Around — 내 주변 랭킹. radius 범위: [0, MaxAroundRadius], 기본 DefaultAroundRadius.
// 미등재 플레이어는 ErrPlayerNotRanked 를 그대로 전파 (Rank 호출 실패 시).
func (s *Service) Around(ctx context.Context, eventID, playerID string, radius int) ([]domain.Entry, error) {
	if radius < 0 {
		radius = domain.DefaultAroundRadius
	}
	if radius > domain.MaxAroundRadius {
		return nil, domain.ErrInvalidLimit
	}
	entries, err := s.store.Around(ctx, eventID, playerID, radius)
	if err != nil {
		// Store 가 ErrPlayerNotRanked 를 포장 없이 반환하면 그대로 전파.
		if errors.Is(err, domain.ErrPlayerNotRanked) {
			return nil, err
		}
		return nil, fmt.Errorf("store.Around: %w", err)
	}
	return entries, nil
}

// Rank — 단일 플레이어 순위 조회.
func (s *Service) Rank(ctx context.Context, eventID, playerID string) (*domain.Entry, error) {
	entry, err := s.store.Rank(ctx, eventID, playerID)
	if err != nil {
		if errors.Is(err, domain.ErrPlayerNotRanked) {
			return nil, err
		}
		return nil, fmt.Errorf("store.Rank: %w", err)
	}
	return entry, nil
}
