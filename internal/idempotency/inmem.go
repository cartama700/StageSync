package idempotency

import (
	"context"
	"sync"
	"time"
)

// InmemStore — 메모리 기반 Store. 개발 · 단일 프로세스 운영용.
// Redis 없는 환경에서 graceful degrade 목적으로 사용.
//
// 만료 정책: lazy expiration (Get 시 체크) + periodic sweep (Cleanup 고루틴).
// 단일 프로세스 기준으로만 동작 — 다중 Pod 환경에선 Redis 필수.
type InmemStore struct {
	mu      sync.Mutex
	entries map[string]inmemEntry
	ttl     time.Duration
	now     func() time.Time
}

type inmemEntry struct {
	entry     Entry
	expiresAt time.Time
}

// NewInmemStore — ttl 기반 inmem 스토어 생성.
// ttl 이 0 이하면 5 분 디폴트.
func NewInmemStore(ttl time.Duration) *InmemStore {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &InmemStore{
		entries: map[string]inmemEntry{},
		ttl:     ttl,
		now:     time.Now,
	}
}

// Get — 키가 있고 미만료면 반환. 만료된 엔트리는 lazy 삭제.
func (s *InmemStore) Get(_ context.Context, key string) (*Entry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return nil, false, nil
	}
	if s.now().After(e.expiresAt) {
		delete(s.entries, key)
		return nil, false, nil
	}
	return &e.entry, true, nil
}

// Set — 이미 키가 존재하고 미만료면 no-op (Race 방어 — Redis SET NX 와 동등).
func (s *InmemStore) Set(_ context.Context, key string, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.entries[key]; ok && s.now().Before(e.expiresAt) {
		return nil
	}
	s.entries[key] = inmemEntry{
		entry:     entry,
		expiresAt: s.now().Add(s.ttl),
	}
	return nil
}

// Sweep — 만료된 엔트리 일괄 삭제. 주기적 호출용 (메인 고루틴 · ticker).
// 반환: 삭제된 엔트리 수.
func (s *InmemStore) Sweep() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	removed := 0
	for k, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, k)
			removed++
		}
	}
	return removed
}
