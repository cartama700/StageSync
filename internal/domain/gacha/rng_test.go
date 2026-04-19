package gacha_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
)

// TestWeightedPick_Empty — 빈 슬라이스는 zero Card.
func TestWeightedPick_Empty(t *testing.T) {
	t.Parallel()
	got := domain.WeightedPick(func(int) int { return 0 }, nil)
	require.Equal(t, domain.Card{}, got)
}

// TestWeightedPick_ZeroTotalWeight — 총 가중치 0 → zero Card.
func TestWeightedPick_ZeroTotalWeight(t *testing.T) {
	t.Parallel()
	cards := []domain.Card{
		{ID: "a", Weight: 0},
		{ID: "b", Weight: 0},
	}
	got := domain.WeightedPick(func(int) int { return 0 }, cards)
	require.Equal(t, domain.Card{}, got)
}

// TestWeightedPick_Deterministic — 주입된 pick 값에 따라 정확한 카드 선택.
func TestWeightedPick_Deterministic(t *testing.T) {
	t.Parallel()
	cards := []domain.Card{
		{ID: "a", Weight: 10}, // [0, 10)
		{ID: "b", Weight: 20}, // [10, 30)
		{ID: "c", Weight: 70}, // [30, 100)
	}

	tests := []struct {
		name   string
		pickFn func(int) int
		wantID string
	}{
		{name: "lowest picks first", pickFn: func(int) int { return 0 }, wantID: "a"},
		{name: "boundary a/b", pickFn: func(int) int { return 9 }, wantID: "a"},
		{name: "first of b", pickFn: func(int) int { return 10 }, wantID: "b"},
		{name: "boundary b/c", pickFn: func(int) int { return 29 }, wantID: "b"},
		{name: "first of c", pickFn: func(int) int { return 30 }, wantID: "c"},
		{name: "last of c", pickFn: func(int) int { return 99 }, wantID: "c"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := domain.WeightedPick(tc.pickFn, cards)
			require.Equal(t, tc.wantID, got.ID)
		})
	}
}

// TestWeightedPick_TotalPassedToRng — pick 함수는 총 가중치 만큼의 n 을 받는다.
func TestWeightedPick_TotalPassedToRng(t *testing.T) {
	t.Parallel()
	cards := []domain.Card{
		{ID: "a", Weight: 3},
		{ID: "b", Weight: 7},
	}
	var receivedN int
	domain.WeightedPick(func(n int) int {
		receivedN = n
		return 0
	}, cards)
	require.Equal(t, 10, receivedN)
}

// TestFilterByRarity — 특정 rarity 카드만 포함된 새 슬라이스.
func TestFilterByRarity(t *testing.T) {
	t.Parallel()
	cards := []domain.Card{
		{ID: "r1", Rarity: domain.RarityR},
		{ID: "sr1", Rarity: domain.RaritySR},
		{ID: "ssr1", Rarity: domain.RaritySSR},
		{ID: "r2", Rarity: domain.RarityR},
		{ID: "ssr2", Rarity: domain.RaritySSR},
	}

	tests := []struct {
		name    string
		rarity  domain.Rarity
		wantIDs []string
	}{
		{name: "R", rarity: domain.RarityR, wantIDs: []string{"r1", "r2"}},
		{name: "SR", rarity: domain.RaritySR, wantIDs: []string{"sr1"}},
		{name: "SSR", rarity: domain.RaritySSR, wantIDs: []string{"ssr1", "ssr2"}},
		{name: "unknown", rarity: domain.Rarity("XXX"), wantIDs: []string{}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := domain.FilterByRarity(cards, tc.rarity)
			ids := make([]string, 0, len(got))
			for _, c := range got {
				ids = append(ids, c.ID)
			}
			require.Equal(t, tc.wantIDs, ids)
		})
	}
}

// TestFilterByRarity_ReturnsNewSlice — 원본 슬라이스는 변형되지 않는다.
func TestFilterByRarity_ReturnsNewSlice(t *testing.T) {
	t.Parallel()
	cards := []domain.Card{
		{ID: "a", Rarity: domain.RarityR},
		{ID: "b", Rarity: domain.RaritySSR},
	}
	original := append([]domain.Card{}, cards...)
	_ = domain.FilterByRarity(cards, domain.RaritySSR)
	require.Equal(t, original, cards, "입력 슬라이스가 변형되어서는 안 됨")
}

// TestRarity_IsValid — 알려진 값만 true.
func TestRarity_IsValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rarity domain.Rarity
		want   bool
	}{
		{domain.RarityR, true},
		{domain.RaritySR, true},
		{domain.RaritySSR, true},
		{domain.Rarity(""), false},
		{domain.Rarity("unknown"), false},
		{domain.Rarity("r"), false}, // case-sensitive
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, tc.rarity.IsValid(), "rarity=%q", tc.rarity)
	}
}
