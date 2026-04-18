// Package profile — 프로필 비즈니스 로직 서비스.
package profile

import (
	"context"
	"fmt"
	"time"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
)

// Repository — Service 가 요구하는 저장소 인터페이스.
// 이 인터페이스는 **사용자 측 (service)** 에서 선언되며,
// 실제 구현은 persistence/inmem (Phase 1) 또는 persistence/mysql (Phase 2) 에서.
// Go 관용: interface at consumer side.
type Repository interface {
	Get(ctx context.Context, id string) (*domain.Profile, error)
	Create(ctx context.Context, p *domain.Profile) error
}

// Service — 프로필 비즈니스 로직 구현.
type Service struct {
	repo Repository
}

// NewService — Service 생성. 의존성 (Repository) 는 생성 시 주입.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetProfile — ID 로 프로필 조회.
// repo 에러는 %w 로 래핑하여 호출자가 errors.Is() 로 식별 가능하게.
func (s *Service) GetProfile(ctx context.Context, id string) (*domain.Profile, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("repo get: %w", err)
	}
	return p, nil
}

// CreateProfile — 새 프로필 생성.
func (s *Service) CreateProfile(ctx context.Context, id, name string) (*domain.Profile, error) {
	p := &domain.Profile{
		ID:        id,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("repo create: %w", err)
	}
	return p, nil
}
