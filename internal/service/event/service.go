// Package event — イベント 비즈니스 로직.
// 시간 기반 상태 전이 · 점수 누적 · 보상 조회.
package event

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	domain "github.com/kimsehoon/stagesync/internal/domain/event"
)

// LeaderboardWriter — Event 서비스가 요구하는 최소 인터페이스.
// 점수 누적 성공 후 Redis ZSET (또는 inmem fallback) 에 반영하기 위함.
// 완전한 ranking.Store 보다 좁게 선언 — 소비자 측 최소 의존.
type LeaderboardWriter interface {
	IncrBy(ctx context.Context, eventID, playerID string, delta int64) (int64, error)
}

// 정책 상수.
const (
	// maxScoreDelta — 1회 반영 가능한 최대 점수 (악의적 과대 delta 차단).
	maxScoreDelta = 1_000_000
)

// Repository — Service 가 요구하는 저장소 인터페이스.
type Repository interface {
	// CreateEvent — 신규 이벤트 등록. 중복 ID 는 ErrAlreadyExists.
	CreateEvent(ctx context.Context, e *domain.Event) error

	// GetEvent — ID 로 조회. 없으면 ErrNotFound.
	GetEvent(ctx context.Context, id string) (*domain.Event, error)

	// ListCurrentEvents — now 시각에 진행 중인 이벤트 (start <= now <= end).
	ListCurrentEvents(ctx context.Context, now time.Time) ([]*domain.Event, error)

	// AddScore — (event, player) 에 delta 점수 누적. UPSERT 로 원자적.
	AddScore(ctx context.Context, eventID, playerID string, delta int64) error

	// GetScore — (event, player) 누적 점수. 없으면 points=0 으로 반환 (err=nil).
	GetScore(ctx context.Context, eventID, playerID string) (*domain.EventScore, error)

	// ListRewardTiers — 이벤트의 보상 티어 (Tier 오름차순).
	ListRewardTiers(ctx context.Context, eventID string) ([]domain.RewardTier, error)

	// InsertRewardTier — 보상 티어 등록 (이벤트 생성 시 함께).
	InsertRewardTier(ctx context.Context, t domain.RewardTier) error
}

// Service — 이벤트 서비스.
type Service struct {
	repo        Repository
	leaderboard LeaderboardWriter // nil 허용 — 랭킹 기능 비활성 모드
	now         func() time.Time
}

// Option — 테스트에서 시계 주입 등.
type Option func(*Service)

// WithNow — 시계 함수 주입 (테스트 전용).
func WithNow(fn func() time.Time) Option {
	return func(s *Service) { s.now = fn }
}

// WithLeaderboard — 점수 누적 성공 시 랭킹에 반영 (best-effort).
// 주입하지 않으면 AddScore 는 MySQL (또는 inmem) 에만 기록.
// leaderboard 호출 실패는 요청 전체 실패로 처리하지 않음 — MySQL 이 truth,
// 랭킹은 eventual consistency 로 복구 가능 (Phase 10 배치 잡 예정).
func WithLeaderboard(lb LeaderboardWriter) Option {
	return func(s *Service) { s.leaderboard = lb }
}

// NewService — 의존성 주입 + 기본 실시간 clock.
func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

// CreateInput — Create 에 전달할 입력 묶음 (+ 보상 티어).
type CreateInput struct {
	ID      string
	Name    string
	StartAt time.Time
	EndAt   time.Time
	Rewards []domain.RewardTier // 비워둘 수 있음
}

// Create — 이벤트 + 보상 티어를 한 번에 등록.
// 원자성은 리포에 맡김 (inmem 은 mutex, mysql 은 tx 로 처리).
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Event, error) {
	if !in.StartAt.Before(in.EndAt) {
		return nil, domain.ErrInvalidWindow
	}
	e := &domain.Event{
		ID:        in.ID,
		Name:      in.Name,
		StartAt:   in.StartAt,
		EndAt:     in.EndAt,
		CreatedAt: s.now(),
	}
	if err := s.repo.CreateEvent(ctx, e); err != nil {
		return nil, fmt.Errorf("repo.CreateEvent: %w", err)
	}
	for _, t := range in.Rewards {
		t.EventID = in.ID
		if err := s.repo.InsertRewardTier(ctx, t); err != nil {
			return nil, fmt.Errorf("repo.InsertRewardTier: %w", err)
		}
	}
	// DB 에서 쓴 값을 재조회하지 않고 위 로컬 값을 그대로 반환 —
	// CreatedAt 을 s.now() 와 DB 기본값 CURRENT_TIMESTAMP 이 미세하게 달라도
	// 응답에서는 서비스 시계 기준으로 일관되게 보이도록.
	return e, nil
}

// Get — 이벤트 단건 조회 + 현재 시각 기준 status.
func (s *Service) Get(ctx context.Context, id string) (*domain.Event, domain.Status, error) {
	e, err := s.repo.GetEvent(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("repo.GetEvent: %w", err)
	}
	return e, e.StatusAt(s.now()), nil
}

// ListCurrent — 진행 중 이벤트 리스트.
func (s *Service) ListCurrent(ctx context.Context) ([]*domain.Event, error) {
	events, err := s.repo.ListCurrentEvents(ctx, s.now())
	if err != nil {
		return nil, fmt.Errorf("repo.ListCurrentEvents: %w", err)
	}
	return events, nil
}

// AddScore — 진행 중 이벤트에 한해 점수 누적.
// UPCOMING / ENDED 는 ErrNotOngoing 으로 거부.
func (s *Service) AddScore(ctx context.Context, eventID, playerID string, delta int64) (*domain.EventScore, error) {
	if delta <= 0 || delta > maxScoreDelta {
		return nil, domain.ErrInvalidDelta
	}
	e, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("repo.GetEvent: %w", err)
	}
	if e.StatusAt(s.now()) != domain.StatusOngoing {
		return nil, domain.ErrNotOngoing
	}
	if err := s.repo.AddScore(ctx, eventID, playerID, delta); err != nil {
		return nil, fmt.Errorf("repo.AddScore: %w", err)
	}
	sc, err := s.repo.GetScore(ctx, eventID, playerID)
	if err != nil {
		return nil, fmt.Errorf("repo.GetScore: %w", err)
	}
	// Leaderboard 반영 — best-effort.
	// 실패해도 MySQL(=truth) 은 이미 누적됐으므로 응답은 성공으로 반환.
	if s.leaderboard != nil {
		if _, err := s.leaderboard.IncrBy(ctx, eventID, playerID, delta); err != nil {
			slog.Warn("leaderboard IncrBy failed (MySQL 은 반영됨 — 추후 복구 필요)",
				"event", eventID, "player", playerID, "delta", delta, "err", err,
			)
		}
	}
	return sc, nil
}

// GetScore — 플레이어 점수 스냅샷 (없으면 points=0).
func (s *Service) GetScore(ctx context.Context, eventID, playerID string) (*domain.EventScore, error) {
	if _, err := s.repo.GetEvent(ctx, eventID); err != nil {
		return nil, fmt.Errorf("repo.GetEvent: %w", err)
	}
	sc, err := s.repo.GetScore(ctx, eventID, playerID)
	if err != nil {
		return nil, fmt.Errorf("repo.GetScore: %w", err)
	}
	return sc, nil
}

// GetRewards — 현재 점수 기준 획득 가능 보상 + 전체 티어 함께 반환.
// 이벤트 종료 이전이어도 조회 자체는 가능 (미획득 티어를 UI 에 보여주기 위함).
func (s *Service) GetRewards(ctx context.Context, eventID, playerID string) (RewardsView, error) {
	e, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return RewardsView{}, fmt.Errorf("repo.GetEvent: %w", err)
	}
	tiers, err := s.repo.ListRewardTiers(ctx, eventID)
	if err != nil {
		return RewardsView{}, fmt.Errorf("repo.ListRewardTiers: %w", err)
	}
	sc, err := s.repo.GetScore(ctx, eventID, playerID)
	if err != nil {
		return RewardsView{}, fmt.Errorf("repo.GetScore: %w", err)
	}
	return RewardsView{
		Status:    e.StatusAt(s.now()),
		Points:    sc.Points,
		Tiers:     tiers,
		Eligible:  domain.EligibleRewards(tiers, sc.Points),
		Claimable: e.StatusAt(s.now()) == domain.StatusEnded,
	}, nil
}

// RewardsView — 보상 조회 응답.
type RewardsView struct {
	Status    domain.Status
	Points    int64
	Tiers     []domain.RewardTier
	Eligible  []domain.RewardTier
	Claimable bool // 종료 이후에만 true (Phase 8 メール 에서 실제 지급)
}
