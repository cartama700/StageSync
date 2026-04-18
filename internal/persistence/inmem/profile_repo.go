// Package inmem — 메모리 기반 저장소 구현 (테스트·개발용).
// Phase 2 에서 persistence/mysql 이 동일 인터페이스를 만족하는 실 DB 구현으로 추가됨.
// 환경변수 기반으로 둘 중 하나를 선택하는 graceful degrade 패턴.
package inmem

import (
	"context"
	"sync"

	domain "github.com/kimsehoon/stagesync/internal/domain/profile"
)

// ProfileRepo — in-memory Profile 저장소.
// service/profile.Repository 인터페이스를 암묵적으로 만족.
type ProfileRepo struct {
	mu    sync.RWMutex
	items map[string]*domain.Profile
}

// NewProfileRepo — 빈 ProfileRepo 생성.
func NewProfileRepo() *ProfileRepo {
	return &ProfileRepo{items: map[string]*domain.Profile{}}
}

// Get — ID 로 프로필 조회. 없으면 domain.ErrNotFound.
// ctx 는 현재 미사용 (메모리라 취소 타이밍 무의미) 이지만 인터페이스 규약 준수 + 후속 구현과 시그니처 통일.
func (r *ProfileRepo) Get(_ context.Context, id string) (*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

// Create — 프로필 생성. 중복 ID 면 domain.ErrAlreadyExists.
func (r *ProfileRepo) Create(_ context.Context, p *domain.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.items[p.ID]; exists {
		return domain.ErrAlreadyExists
	}
	r.items[p.ID] = p
	return nil
}
