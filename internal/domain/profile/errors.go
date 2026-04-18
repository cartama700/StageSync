package profile

import "errors"

// Sentinel errors — 도메인 수준 에러.
// 호출자는 errors.Is(err, ErrNotFound) 로 식별하여 적절히 응답 (HTTP 404 등).
var (
	// ErrNotFound — 프로필이 저장소에 없음.
	ErrNotFound = errors.New("profile not found")

	// ErrAlreadyExists — 동일 ID 프로필이 이미 존재.
	ErrAlreadyExists = errors.New("profile already exists")
)
