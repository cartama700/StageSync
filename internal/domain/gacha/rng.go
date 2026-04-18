package gacha

// RandIntN — 테스트 가능성을 위해 RNG 를 함수 타입으로 추상화.
// 서비스 레이어가 자신의 RNG (mutex 보호) 또는 stdlib 의 `rand.IntN` 을 전달.
// 반환값은 [0, n) 범위.
type RandIntN func(n int) int

// WeightedPick — 카드 리스트에서 가중치 기반 1개 선택.
// cards 가 빈 리스트면 zero Card 반환 (호출자가 사전 검증 책임).
// 복잡도 O(N) — 풀 크기 N 은 통상 수십~수백 수준이라 이분 탐색 불필요.
func WeightedPick(rngInt RandIntN, cards []Card) Card {
	if len(cards) == 0 {
		return Card{}
	}
	total := 0
	for _, c := range cards {
		total += c.Weight
	}
	if total <= 0 {
		return Card{}
	}
	pick := rngInt(total)
	acc := 0
	for _, c := range cards {
		acc += c.Weight
		if pick < acc {
			return c
		}
	}
	// 이론적으로 도달 불가 (pick < total 보장). 안전망.
	return cards[len(cards)-1]
}

// FilterByRarity — 주어진 rarity 에 해당하는 카드만 필터링 (새 슬라이스).
func FilterByRarity(cards []Card, rarity Rarity) []Card {
	out := make([]Card, 0, len(cards))
	for _, c := range cards {
		if c.Rarity == rarity {
			out = append(out, c)
		}
	}
	return out
}
