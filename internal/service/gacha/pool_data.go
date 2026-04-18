package gacha

import (
	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
)

// DemoPoolID — Phase 5 MVP 데모 풀 ID.
// Phase 5b 에서 YAML 설정 파일 로드로 교체 예정.
const DemoPoolID = "demo"

// StaticPoolRegistry — 하드코딩된 풀 맵. PoolRegistry 인터페이스 구현.
type StaticPoolRegistry struct {
	pools map[string]*domain.Pool
}

// NewStaticPoolRegistry — Phase 5 기본 데모 풀 1개 포함 레지스트리 생성.
//
// 데모 풀 확률 설계 (총 1000 가중치):
//   - SSR 3% — 10 + 10 + 10 = 30
//   - SR  17% — 50 + 50 + 70 = 170
//   - R   80% — 300 + 300 + 200 = 800
//   - 천장 80 roll — 리듬게임 업계 평균치
func NewStaticPoolRegistry() *StaticPoolRegistry {
	demo := &domain.Pool{
		ID:            DemoPoolID,
		Name:          "Demo Pool",
		PityThreshold: 80,
		Cards: []domain.Card{
			// SSR — 3%
			{ID: "c_miku_ssr", Name: "Miku SSR", Rarity: domain.RaritySSR, Weight: 10},
			{ID: "c_rin_ssr", Name: "Rin SSR", Rarity: domain.RaritySSR, Weight: 10},
			{ID: "c_len_ssr", Name: "Len SSR", Rarity: domain.RaritySSR, Weight: 10},
			// SR — 17%
			{ID: "c_miku_sr", Name: "Miku SR", Rarity: domain.RaritySR, Weight: 50},
			{ID: "c_rin_sr", Name: "Rin SR", Rarity: domain.RaritySR, Weight: 50},
			{ID: "c_len_sr", Name: "Len SR", Rarity: domain.RaritySR, Weight: 70},
			// R — 80%
			{ID: "c_basic_a", Name: "Basic A", Rarity: domain.RarityR, Weight: 300},
			{ID: "c_basic_b", Name: "Basic B", Rarity: domain.RarityR, Weight: 300},
			{ID: "c_basic_c", Name: "Basic C", Rarity: domain.RarityR, Weight: 200},
		},
	}
	return &StaticPoolRegistry{
		pools: map[string]*domain.Pool{
			DemoPoolID: demo,
		},
	}
}

// GetPool — 풀 조회. 없으면 domain.ErrPoolNotFound.
func (r *StaticPoolRegistry) GetPool(poolID string) (*domain.Pool, error) {
	p, ok := r.pools[poolID]
	if !ok {
		return nil, domain.ErrPoolNotFound
	}
	return p, nil
}
