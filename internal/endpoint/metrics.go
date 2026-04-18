package endpoint

import (
	"net/http"

	"github.com/kimsehoon/stagesync/internal/lifecycle"
	"github.com/kimsehoon/stagesync/internal/room"
)

// MetricsHandler — GET /api/metrics.
// Phase 1 에서 기존 main.go 의 closure factory 패턴에서 구조체 메서드 패턴으로 승격.
// Phase 9 에서 Prometheus Histogram·Counter 추가 시 필드가 여기에 더 붙음.
type MetricsHandler struct {
	Room *room.Room
	Opt  *lifecycle.Optimize
}

// Get — 메트릭 JSON 응답. 추후 TPS·P99 latency 등 확장 예정.
func (h *MetricsHandler) Get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"tps":              0,
		"connectedPlayers": h.Room.Size(),
		"optimized":        h.Opt.On(),
	})
}
