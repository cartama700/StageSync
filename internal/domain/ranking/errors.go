package ranking

import "errors"

// ErrPlayerNotRanked — 해당 플레이어가 아직 랭킹에 등재되지 않음 (점수 반영 전).
// "존재하지 않는 이벤트" 와는 구분 — 이벤트 유무는 event 도메인의 책임.
var ErrPlayerNotRanked = errors.New("player not in ranking")

// ErrInvalidLimit — Top-N 의 n 또는 Around 의 radius 가 허용 범위 밖.
var ErrInvalidLimit = errors.New("invalid limit")
