package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kimsehoon/stagesync/internal/apperror"
	domain "github.com/kimsehoon/stagesync/internal/domain/battle"
)

// BattleApplier — 이 handler 가 요구하는 서비스 인터페이스 (consumer-defined).
// 실제 구현은 V1Naive 또는 V2UserQueue — BATTLE_IMPL 환경변수로 선택.
type BattleApplier interface {
	Apply(ctx context.Context, playerID string, damage int) (int, error)
}

// BattleHandler — Phase 19 HP 데드락 랩 REST 엔드포인트.
//
// 제출용 MVP 에선 비보호 (데모 · 벤치 편의). 실 게임은 반드시 `RequireAuth` 적용.
type BattleHandler struct {
	Applier BattleApplier
	// ImplLabel — 현재 선택된 구현 이름 (응답 메타데이터, 디버깅용).
	ImplLabel string
}

// Mount — 라우트 등록.
func (h *BattleHandler) Mount(r chi.Router) {
	r.Post("/api/battle/damage", h.ApplyDamage)
}

// ----- DTO -----

type damageReq struct {
	TargetPlayer string `json:"target_player" validate:"required,min=1,max=64"`
	Damage       int    `json:"damage"        validate:"required,min=1,max=100000"`
}

type damageResp struct {
	PlayerID string `json:"player_id"`
	HP       int    `json:"hp"`
	Impl     string `json:"impl"` // 디버그용 — 현재 적용된 구현체 라벨.
}

// ApplyDamage — POST /api/battle/damage.
// body `{"target_player":"p1","damage":10}` → 처리 후 남은 HP 반환.
func (h *BattleHandler) ApplyDamage(w http.ResponseWriter, r *http.Request) {
	var req damageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation("invalid body", []apperror.FieldError{
			{Field: "body", Tag: "json", Message: err.Error()},
		}))
		return
	}
	if err := vldtr.Struct(req); err != nil {
		apperror.WriteJSON(w, toValidationError(err))
		return
	}

	hp, err := h.Applier.Apply(r.Context(), req.TargetPlayer, req.Damage)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidDamage) {
			apperror.WriteJSON(w, apperror.Validation("damage out of range", []apperror.FieldError{
				{Field: "damage", Tag: "range", Message: "1..100000"},
			}))
			return
		}
		apperror.WriteJSON(w, apperror.Internal("apply damage", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, damageResp{
		PlayerID: req.TargetPlayer,
		HP:       hp,
		Impl:     h.ImplLabel,
	})
}
