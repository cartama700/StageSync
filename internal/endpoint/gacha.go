package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/apperror"
	domain "github.com/kimsehoon/stagesync/internal/domain/gacha"
)

// GachaService — 이 handler 가 요구하는 서비스 인터페이스 (consumer-defined).
type GachaService interface {
	Roll(ctx context.Context, playerID, poolID string, count int) ([]*domain.Roll, error)
	ListHistory(ctx context.Context, playerID string, limit int) ([]*domain.Roll, error)
	GetPity(ctx context.Context, playerID, poolID string) (int, error)
}

// GachaHandler — ガチャ REST 엔드포인트 묶음.
type GachaHandler struct {
	Service GachaService
}

// Mount — 3개 가챠 라우트 등록.
func (h *GachaHandler) Mount(r chi.Router) {
	r.Post("/api/gacha/roll", h.Roll)
	r.Get("/api/gacha/history/{playerId}", h.History)
	r.Get("/api/gacha/pity/{playerId}/{poolId}", h.Pity)
}

// ----- DTO -----

type rollReq struct {
	PlayerID string `json:"player" validate:"required,min=1,max=64"`
	PoolID   string `json:"pool"   validate:"required,min=1,max=64"`
	Count    int    `json:"count"  validate:"required,min=1,max=10"`
}

type rollDTO struct {
	ID       string    `json:"id"`
	PlayerID string    `json:"player_id"`
	PoolID   string    `json:"pool_id"`
	CardID   string    `json:"card_id"`
	Rarity   string    `json:"rarity"`
	IsPity   bool      `json:"is_pity"`
	PulledAt time.Time `json:"pulled_at"`
}

func rollToDTO(r *domain.Roll) rollDTO {
	return rollDTO{
		ID:       r.ID,
		PlayerID: r.PlayerID,
		PoolID:   r.PoolID,
		CardID:   r.CardID,
		Rarity:   string(r.Rarity),
		IsPity:   r.IsPity,
		PulledAt: r.PulledAt,
	}
}

func rollsToDTO(rs []*domain.Roll) []rollDTO {
	out := make([]rollDTO, 0, len(rs))
	for _, r := range rs {
		out = append(out, rollToDTO(r))
	}
	return out
}

// ----- 핸들러 -----

// Roll — POST /api/gacha/roll.
func (h *GachaHandler) Roll(w http.ResponseWriter, r *http.Request) {
	var req rollReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid JSON body", nil))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}

	rolls, err := h.Service.Roll(r.Context(), req.PlayerID, req.PoolID, req.Count)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPoolNotFound):
			apperror.WriteJSON(w, apperror.NotFound("gacha pool", req.PoolID))
		case errors.Is(err, domain.ErrInvalidCount):
			apperror.WriteJSON(w, apperror.Validation(err.Error(), nil))
		case errors.Is(err, domain.ErrEmptyPool):
			apperror.WriteJSON(w, apperror.Internal("pool has no cards", err))
		default:
			apperror.WriteJSON(w, apperror.Internal("gacha roll", err))
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, rollsToDTO(rolls))
}

// History — GET /api/gacha/history/{playerId}?limit=N.
func (h *GachaHandler) History(w http.ResponseWriter, r *http.Request) {
	playerID := chi.URLParam(r, "playerId")
	if playerID == "" {
		apperror.WriteJSON(w, apperror.Validation("playerId path param is required", nil))
		return
	}
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	rolls, err := h.Service.ListHistory(r.Context(), playerID, limit)
	if err != nil {
		apperror.WriteJSON(w, apperror.Internal("gacha history", err))
		return
	}
	writeJSON(w, rollsToDTO(rolls))
}

// Pity — GET /api/gacha/pity/{playerId}/{poolId}.
func (h *GachaHandler) Pity(w http.ResponseWriter, r *http.Request) {
	playerID := chi.URLParam(r, "playerId")
	poolID := chi.URLParam(r, "poolId")
	if playerID == "" || poolID == "" {
		apperror.WriteJSON(w, apperror.Validation("playerId and poolId are required", nil))
		return
	}
	pity, err := h.Service.GetPity(r.Context(), playerID, poolID)
	if err != nil {
		apperror.WriteJSON(w, apperror.Internal("gacha pity", err))
		return
	}
	writeJSON(w, map[string]any{
		"player_id": playerID,
		"pool_id":   poolID,
		"counter":   pity,
	})
}
