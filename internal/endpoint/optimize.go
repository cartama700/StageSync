package endpoint

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/kimsehoon/stagesync/internal/apperror"
	"github.com/kimsehoon/stagesync/internal/lifecycle"
)

// OptimizeHandler — POST /api/optimize 로 Naive/Pooled 경로 전환.
type OptimizeHandler struct {
	State *lifecycle.Optimize
}

type optimizeReq struct {
	On bool `json:"on"`
}

// ServeHTTP — body: {"on": true|false} 를 받아 atomic.Bool 에 반영.
func (h *OptimizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req optimizeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apperror.WriteJSON(w, apperror.Validation(`invalid body (expected {"on": bool})`, nil))
		return
	}
	h.State.Set(req.On)
	slog.Info("optimize toggled", "on", req.On)
	w.WriteHeader(http.StatusNoContent)
}
