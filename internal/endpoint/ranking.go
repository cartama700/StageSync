package endpoint

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/apperror"
	domain "github.com/kimsehoon/stagesync/internal/domain/ranking"
)

// RankingService — 이 handler 가 요구하는 서비스 인터페이스 (consumer-defined).
type RankingService interface {
	Top(ctx context.Context, eventID string, n int) ([]domain.Entry, error)
	Around(ctx context.Context, eventID, playerID string, radius int) ([]domain.Entry, error)
	Rank(ctx context.Context, eventID, playerID string) (*domain.Entry, error)
}

// RankingHandler — ランキング REST 엔드포인트.
type RankingHandler struct {
	Service RankingService
}

// Mount — 랭킹 라우트 등록.
//
//	GET /api/ranking/{eventId}/top?n=10
//	GET /api/ranking/{eventId}/me/{playerId}?radius=5
func (h *RankingHandler) Mount(r chi.Router) {
	r.Get("/api/ranking/{eventId}/top", h.Top)
	r.Get("/api/ranking/{eventId}/me/{playerId}", h.Around)
}

// ----- DTO -----

type rankingEntryDTO struct {
	PlayerID string `json:"player_id"`
	Score    int64  `json:"score"`
	Rank     int    `json:"rank"`
}

type topResponse struct {
	EventID string            `json:"event_id"`
	Count   int               `json:"count"`
	Entries []rankingEntryDTO `json:"entries"`
}

type aroundResponse struct {
	EventID  string            `json:"event_id"`
	PlayerID string            `json:"player_id"`
	Rank     int               `json:"rank"`
	Score    int64             `json:"score"`
	Radius   int               `json:"radius"`
	Entries  []rankingEntryDTO `json:"entries"`
}

// ----- handlers -----

// Top — GET /api/ranking/{eventId}/top?n=10
func (h *RankingHandler) Top(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	n := parseIntQueryDefault(r, "n", domain.DefaultTopN)

	entries, err := h.Service.Top(r.Context(), eventID, n)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidLimit) {
			apperror.WriteJSON(w, apperror.Validation("n out of range", []apperror.FieldError{
				{Field: "n", Tag: "range", Message: "must be 1..100"},
			}))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("ranking top failed", err))
		return
	}
	writeJSON(w, topResponse{
		EventID: eventID,
		Count:   len(entries),
		Entries: toDTOs(entries),
	})
}

// Around — GET /api/ranking/{eventId}/me/{playerId}?radius=5
func (h *RankingHandler) Around(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventId")
	playerID := chi.URLParam(r, "playerId")
	radius := parseIntQueryDefault(r, "radius", domain.DefaultAroundRadius)

	entries, err := h.Service.Around(r.Context(), eventID, playerID, radius)
	if err != nil {
		if errors.Is(err, domain.ErrPlayerNotRanked) {
			apperror.WriteJSON(w, apperror.NotFound("ranking entry", playerID))
			return
		}
		if errors.Is(err, domain.ErrInvalidLimit) {
			apperror.WriteJSON(w, apperror.Validation("radius out of range", []apperror.FieldError{
				{Field: "radius", Tag: "range", Message: "must be 0..25"},
			}))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("ranking around failed", err))
		return
	}
	// 본인 엔트리를 메타로 함께 반환 (response 선두 lookup 으로 재계산 불필요).
	var selfRank int
	var selfScore int64
	for _, e := range entries {
		if e.PlayerID == playerID {
			selfRank = e.Rank
			selfScore = e.Score
			break
		}
	}
	writeJSON(w, aroundResponse{
		EventID:  eventID,
		PlayerID: playerID,
		Rank:     selfRank,
		Score:    selfScore,
		Radius:   radius,
		Entries:  toDTOs(entries),
	})
}

// ----- helpers -----

func parseIntQueryDefault(r *http.Request, key string, def int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}

func toDTOs(entries []domain.Entry) []rankingEntryDTO {
	out := make([]rankingEntryDTO, len(entries))
	for i, e := range entries {
		out[i] = rankingEntryDTO{
			PlayerID: e.PlayerID,
			Score:    e.Score,
			Rank:     e.Rank,
		}
	}
	return out
}
