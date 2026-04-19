package event

import "errors"

// Sentinel errors — service / endpoint 가 errors.Is() 로 식별 후 HTTP 매핑.
var (
	// ErrNotFound — 존재하지 않는 이벤트 ID.
	ErrNotFound = errors.New("event not found")

	// ErrAlreadyExists — 이벤트 생성 시 ID 중복.
	ErrAlreadyExists = errors.New("event already exists")

	// ErrInvalidWindow — start_at >= end_at 와 같이 기간이 유효하지 않음.
	ErrInvalidWindow = errors.New("event time window is invalid (start must be before end)")

	// ErrNotOngoing — 진행 중이 아닌 이벤트에 점수 반영 시도.
	ErrNotOngoing = errors.New("event is not ongoing — cannot accumulate points")

	// ErrInvalidDelta — delta 값이 허용 범위 밖 (음수 또는 과도하게 큼).
	ErrInvalidDelta = errors.New("invalid score delta")
)
