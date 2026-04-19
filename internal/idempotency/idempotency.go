// Package idempotency — `Idempotency-Key` 헤더 기반 중복 요청 감지.
//
// 핵심 패턴:
//  1. 클라이언트가 `Idempotency-Key: <uuid>` 헤더 포함.
//  2. 서버가 해당 키로 cache lookup → 히트면 저장된 응답 리플레이 (실제 핸들러 재실행 안 함).
//  3. 미스면 핸들러 실행 → 응답을 캡처해서 TTL 로 저장.
//
// 사용 케이스: 네트워크 지연으로 클라이언트가 "따닥" (밀리초 단위 중복 클릭) 했을 때
// DB 까지 도달하는 중복 처리를 차단. Stripe API 의 idempotency 패턴과 동일.
package idempotency

import "context"

// Entry — 캐시된 응답 스냅샷.
// Headers 는 일부러 생략 — request_id 같은 per-request 헤더가 캐시된 채 재사용되면 혼란스러움.
type Entry struct {
	Status int    `json:"status"`
	Body   []byte `json:"body"`
}

// Store — 캐시 저장소 인터페이스.
// 미들웨어는 이 인터페이스를 통해서만 호출. Redis · inmem 두 구현.
//
// 의미 (Redis `SET NX EX` 동작에 정렬):
//   - Get: 키가 존재하고 미만료면 entry + true. 없거나 만료면 nil + false.
//   - Set: 키가 **이미 존재하면 no-op**. 없으면 TTL 과 함께 저장.
//     (동시 요청 중 첫 번째만 저장되고 나머지는 silent skip — Race 안전성의 핵심)
type Store interface {
	Get(ctx context.Context, key string) (*Entry, bool, error)
	Set(ctx context.Context, key string, entry Entry) error
}
