package battle

import "errors"

// ErrInvalidDamage — damage 가 허용 범위 밖.
var ErrInvalidDamage = errors.New("damage must be 1..MaxDamagePerRequest")

// ErrNotFound — 플레이어 HP 행 없음 (초기화 전).
var ErrNotFound = errors.New("player hp not found")
