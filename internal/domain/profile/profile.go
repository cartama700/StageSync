// Package profile — 플레이어 프로필 도메인 모델.
// 다른 레이어 (service, persistence, endpoint) 가 이 패키지에 의존하지만,
// 이 패키지는 어디에도 의존하지 않는 순수 도메인.
package profile

import "time"

// Profile — 플레이어 プロフィール 도메인 객체.
type Profile struct {
	ID        string
	Name      string
	CreatedAt time.Time
}
