package gacha

import "errors"

// Sentinel errors — service / endpoint 가 errors.Is() 로 식별 후 HTTP 매핑.
var (
	// ErrPoolNotFound — 존재하지 않는 풀 ID 요청.
	ErrPoolNotFound = errors.New("gacha pool not found")

	// ErrInvalidCount — roll 횟수가 허용 범위 밖 (1-10 권장).
	ErrInvalidCount = errors.New("invalid roll count (must be 1..10)")

	// ErrEmptyPool — 풀에 카드가 없어 뽑기 불가 (운영 설정 오류).
	ErrEmptyPool = errors.New("gacha pool has no cards")
)
