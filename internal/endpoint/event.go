package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/apperror"
	domain "github.com/kimsehoon/stagesync/internal/domain/event"
	eventsvc "github.com/kimsehoon/stagesync/internal/service/event"
)

// EventService — 이 handler 가 요구하는 서비스 인터페이스 (consumer-defined).
type EventService interface {
	Create(ctx context.Context, in eventsvc.CreateInput) (*domain.Event, error)
	Get(ctx context.Context, id string) (*domain.Event, domain.Status, error)
	ListCurrent(ctx context.Context) ([]*domain.Event, error)
	AddScore(ctx context.Context, eventID, playerID string, delta int64) (*domain.EventScore, error)
	GetScore(ctx context.Context, eventID, playerID string) (*domain.EventScore, error)
	GetRewards(ctx context.Context, eventID, playerID string) (eventsvc.RewardsView, error)
}

// EventHandler — イベント REST 엔드포인트 묶음.
type EventHandler struct {
	Service EventService
}

// Mount — 6개 라우트 등록.
func (h *EventHandler) Mount(r chi.Router) {
	r.Post("/api/event", h.Create)
	r.Get("/api/event/current", h.ListCurrent)
	r.Get("/api/event/{id}", h.Get)
	r.Post("/api/event/{id}/score", h.AddScore)
	r.Get("/api/event/{id}/score/{playerId}", h.GetScore)
	r.Get("/api/event/{id}/rewards/{playerId}", h.GetRewards)
}

// ----- DTO -----

type rewardTierReq struct {
	Tier       int    `json:"tier"        validate:"required,min=1,max=100"`
	MinPoints  int64  `json:"min_points"  validate:"min=0"`
	RewardName string `json:"reward_name" validate:"required,min=1,max=128"`
}

type createEventReq struct {
	ID      string          `json:"id"       validate:"required,min=1,max=64"`
	Name    string          `json:"name"     validate:"required,min=1,max=128"`
	StartAt time.Time       `json:"start_at" validate:"required"`
	EndAt   time.Time       `json:"end_at"   validate:"required"`
	Rewards []rewardTierReq `json:"rewards"  validate:"dive"`
}

type addScoreReq struct {
	PlayerID string `json:"player" validate:"required,min=1,max=64"`
	Delta    int64  `json:"delta"  validate:"required,min=1"`
}

type eventDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status,omitempty"`
}

type scoreDTO struct {
	EventID   string    `json:"event_id"`
	PlayerID  string    `json:"player_id"`
	Points    int64     `json:"points"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type rewardTierDTO struct {
	Tier       int    `json:"tier"`
	MinPoints  int64  `json:"min_points"`
	RewardName string `json:"reward_name"`
}

type rewardsDTO struct {
	EventID   string          `json:"event_id"`
	PlayerID  string          `json:"player_id"`
	Status    string          `json:"status"`
	Points    int64           `json:"points"`
	Tiers     []rewardTierDTO `json:"tiers"`
	Eligible  []rewardTierDTO `json:"eligible"`
	Claimable bool            `json:"claimable"`
}

func eventToDTO(e *domain.Event, status domain.Status) eventDTO {
	return eventDTO{
		ID:        e.ID,
		Name:      e.Name,
		StartAt:   e.StartAt,
		EndAt:     e.EndAt,
		CreatedAt: e.CreatedAt,
		Status:    string(status),
	}
}

func tiersToDTO(ts []domain.RewardTier) []rewardTierDTO {
	out := make([]rewardTierDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, rewardTierDTO{
			Tier:       t.Tier,
			MinPoints:  t.MinPoints,
			RewardName: t.RewardName,
		})
	}
	return out
}

// ----- 핸들러 -----

// Create — POST /api/event.
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEventReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid JSON body", nil))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}

	rewards := make([]domain.RewardTier, 0, len(req.Rewards))
	for _, rt := range req.Rewards {
		rewards = append(rewards, domain.RewardTier{
			Tier:       rt.Tier,
			MinPoints:  rt.MinPoints,
			RewardName: rt.RewardName,
		})
	}
	e, err := h.Service.Create(r.Context(), eventsvc.CreateInput{
		ID:      req.ID,
		Name:    req.Name,
		StartAt: req.StartAt,
		EndAt:   req.EndAt,
		Rewards: rewards,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			apperror.WriteJSON(w, apperror.Conflict("event already exists"))
		case errors.Is(err, domain.ErrInvalidWindow):
			apperror.WriteJSON(w, apperror.Validation(err.Error(), nil))
		default:
			apperror.WriteJSON(w, apperror.Internal("create event", err))
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, eventToDTO(e, e.StatusAt(time.Now())))
}

// Get — GET /api/event/{id}.
func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		apperror.WriteJSON(w, apperror.Validation("id is required", nil))
		return
	}
	e, st, err := h.Service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			apperror.WriteJSON(w, apperror.NotFound("event", id))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("get event", err))
		return
	}
	writeJSON(w, eventToDTO(e, st))
}

// ListCurrent — GET /api/event/current.
func (h *EventHandler) ListCurrent(w http.ResponseWriter, r *http.Request) {
	events, err := h.Service.ListCurrent(r.Context())
	if err != nil {
		apperror.WriteJSON(w, apperror.Internal("list current events", err))
		return
	}
	now := time.Now()
	out := make([]eventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, eventToDTO(e, e.StatusAt(now)))
	}
	writeJSON(w, out)
}

// AddScore — POST /api/event/{id}/score.
func (h *EventHandler) AddScore(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		apperror.WriteJSON(w, apperror.Validation("id is required", nil))
		return
	}
	var req addScoreReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid JSON body", nil))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}
	sc, err := h.Service.AddScore(r.Context(), eventID, req.PlayerID, req.Delta)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			apperror.WriteJSON(w, apperror.NotFound("event", eventID))
		case errors.Is(err, domain.ErrNotOngoing):
			apperror.WriteJSON(w, apperror.Conflict(err.Error()))
		case errors.Is(err, domain.ErrInvalidDelta):
			apperror.WriteJSON(w, apperror.Validation(err.Error(), nil))
		default:
			apperror.WriteJSON(w, apperror.Internal("add score", err))
		}
		return
	}
	writeJSON(w, scoreDTO{
		EventID:   sc.EventID,
		PlayerID:  sc.PlayerID,
		Points:    sc.Points,
		UpdatedAt: sc.UpdatedAt,
	})
}

// GetScore — GET /api/event/{id}/score/{playerId}.
func (h *EventHandler) GetScore(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	playerID := chi.URLParam(r, "playerId")
	if eventID == "" || playerID == "" {
		apperror.WriteJSON(w, apperror.Validation("id and playerId are required", nil))
		return
	}
	sc, err := h.Service.GetScore(r.Context(), eventID, playerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			apperror.WriteJSON(w, apperror.NotFound("event", eventID))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("get score", err))
		return
	}
	writeJSON(w, scoreDTO{
		EventID:   sc.EventID,
		PlayerID:  sc.PlayerID,
		Points:    sc.Points,
		UpdatedAt: sc.UpdatedAt,
	})
}

// GetRewards — GET /api/event/{id}/rewards/{playerId}.
func (h *EventHandler) GetRewards(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "id")
	playerID := chi.URLParam(r, "playerId")
	if eventID == "" || playerID == "" {
		apperror.WriteJSON(w, apperror.Validation("id and playerId are required", nil))
		return
	}
	view, err := h.Service.GetRewards(r.Context(), eventID, playerID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			apperror.WriteJSON(w, apperror.NotFound("event", eventID))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("get rewards", err))
		return
	}
	writeJSON(w, rewardsDTO{
		EventID:   eventID,
		PlayerID:  playerID,
		Status:    string(view.Status),
		Points:    view.Points,
		Tiers:     tiersToDTO(view.Tiers),
		Eligible:  tiersToDTO(view.Eligible),
		Claimable: view.Claimable,
	})
}
