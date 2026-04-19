// Package battle — Phase 19 HP 데드락 랩 서비스 레이어.
//
// **목적**: "한 플레이어 row 에 쏠리는 동시 쓰기" 라는 프로덕션 장애 패턴을 재현 · 해결하는
// 3 단 비교 (v1-naive → v2-queue → v3-redis-wb).
//
// 각 구현은 `Applier` 인터페이스를 만족 → 환경변수 `BATTLE_IMPL` 또는 테스트 코드에서 swap 가능.
package battle

import (
	"context"
	"fmt"
	"sync"

	domain "github.com/kimsehoon/stagesync/internal/domain/battle"
)

// Repo — 서비스가 요구하는 저장소 인터페이스 (consumer-defined).
// 구현은 MySQL 또는 inmem.
type Repo interface {
	// ApplyDamageNaive — `SELECT ... FOR UPDATE` + `UPDATE` 의 조합.
	// 같은 playerID 에 동시 호출이 쏠리면 락 경합 발생 — 본 랩의 타겟.
	ApplyDamageNaive(ctx context.Context, playerID string, damage int) (remainingHP int, err error)
	Reset(ctx context.Context, playerID string, hp int) error
	Get(ctx context.Context, playerID string) (*domain.PlayerHP, error)
}

// Applier — 전투 데미지 적용 추상화. 3 개 구현체 중 선택.
type Applier interface {
	Apply(ctx context.Context, playerID string, damage int) (int, error)
}

// ----- v1-naive — 그대로 DB FOR UPDATE 에 의존 -----

// V1Naive — 핸들러가 들어오는 모든 요청을 그대로 DB 로 흘려보냄.
// MySQL 의 `innodb_lock_wait_timeout` + `FOR UPDATE` 가 락 큐 역할.
// 평시엔 동작하지만, 한 player 에 폭발적 동시 요청 시 **lock wait timeout** 에러 · 데드락 · p99 폭증.
//
// **이 구현이 재현하는 장애**: "한 유저 row 에 트래픽이 쏠리면 전체 RPS 가 떨어진다"
type V1Naive struct {
	repo Repo
}

// NewV1Naive — 의존성 주입.
func NewV1Naive(repo Repo) *V1Naive { return &V1Naive{repo: repo} }

// Apply — 그냥 repo 호출. 검증만 추가.
func (s *V1Naive) Apply(ctx context.Context, playerID string, damage int) (int, error) {
	if damage < 1 || damage > domain.MaxDamagePerRequest {
		return 0, domain.ErrInvalidDamage
	}
	hp, err := s.repo.ApplyDamageNaive(ctx, playerID, damage)
	if err != nil {
		return 0, fmt.Errorf("apply damage naive: %w", err)
	}
	return hp, nil
}

// ----- v2-queue — playerID 별 단일 워커로 Go 레벨 직렬화 -----

// V2UserQueue — 같은 `playerID` 의 요청을 단일 채널에 enqueue →
// 전용 워커 고루틴 하나가 순차 처리 → DB 에는 **한 번에 한 요청만** 내려감 → 락 경합 0.
//
// 트레이드오프:
//   - 장점: lock wait / 데드락 완벽 차단 · p99 안정.
//   - 단점: 단일 프로세스 스코프 (다중 Pod 면 Pod 별 워커 존재 → 각 Pod → DB 다시 경합).
//     분산 환경에서는 Redis Stream · NATS 같은 **분산 큐** 필요 (v3 에서 보강 예정).
//   - 메모리: player 별 map entry 남음. 프로덕션은 idle TTL sweep 추가 필요.
//
// queue 용량: 플레이어별 bounded (cap=`QueueCapacity`). 초과 시 ctx 타임아웃으로 자연 실패.
type V2UserQueue struct {
	repo Repo

	mu     sync.Mutex
	queues map[string]chan queueReq
}

// queueReq — 단일 공격 요청을 워커로 전달하기 위한 봉투.
type queueReq struct {
	ctx      context.Context //nolint:containedctx // 큐 패턴상 ctx 를 메시지에 담아야 함
	playerID string
	damage   int
	result   chan queueResp
}

type queueResp struct {
	hp  int
	err error
}

// QueueCapacity — 플레이어당 bounded channel 크기.
// 초과 시 send 블로킹 → ctx 타임아웃으로 backpressure.
const QueueCapacity = 64

// NewV2UserQueue — 의존성 주입 + 내부 map 초기화.
func NewV2UserQueue(repo Repo) *V2UserQueue {
	return &V2UserQueue{
		repo:   repo,
		queues: map[string]chan queueReq{},
	}
}

// Apply — playerID 의 큐에 enqueue + 응답 대기.
func (s *V2UserQueue) Apply(ctx context.Context, playerID string, damage int) (int, error) {
	if damage < 1 || damage > domain.MaxDamagePerRequest {
		return 0, domain.ErrInvalidDamage
	}
	q := s.getOrCreate(playerID)
	resp := make(chan queueResp, 1)
	req := queueReq{ctx: ctx, playerID: playerID, damage: damage, result: resp}

	// 큐 쌓이면 ctx 로 timeout.
	select {
	case q <- req:
	case <-ctx.Done():
		return 0, fmt.Errorf("enqueue: %w", ctx.Err())
	}

	select {
	case r := <-resp:
		return r.hp, r.err
	case <-ctx.Done():
		return 0, fmt.Errorf("await: %w", ctx.Err())
	}
}

// getOrCreate — playerID 의 큐 + 워커가 없으면 생성.
func (s *V2UserQueue) getOrCreate(playerID string) chan queueReq {
	s.mu.Lock()
	defer s.mu.Unlock()
	if q, ok := s.queues[playerID]; ok {
		return q
	}
	q := make(chan queueReq, QueueCapacity)
	s.queues[playerID] = q
	go s.worker(q)
	return q
}

// worker — 하나의 playerID 에 대한 모든 요청을 순차 처리. DB 에 직렬 쓰기.
// 현재는 종료 경로 없음 (channel close 기반 shutdown 은 후속 작업 — prod 에선 필요).
func (s *V2UserQueue) worker(q chan queueReq) {
	for req := range q {
		hp, err := s.repo.ApplyDamageNaive(req.ctx, req.playerID, req.damage)
		req.result <- queueResp{hp: hp, err: err}
	}
}

// ----- 구현 선택 헬퍼 -----

// Implementation — 환경변수 BATTLE_IMPL 매핑.
type Implementation string

const (
	ImplNaive Implementation = "naive" // v1
	ImplQueue Implementation = "queue" // v2
	// ImplWriteBehind Implementation = "wb" — v3 (제출 후 별도 PR, Redis 의존)
)

// Build — 지정된 구현으로 Applier 생성.
// 알 수 없는 값이면 ImplNaive 로 fallback.
func Build(impl Implementation, repo Repo) Applier {
	switch impl {
	case ImplQueue:
		return NewV2UserQueue(repo)
	default:
		return NewV1Naive(repo)
	}
}
