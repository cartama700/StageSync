package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kimsehoon/stagesync/internal/lifecycle"
	"github.com/kimsehoon/stagesync/internal/room"
)

// PrometheusHandler — Prometheus 포맷 `/metrics` 엔드포인트 + 미들웨어가 관측할 histogram 소유.
// 격리된 Registry 를 써서 테스트 간 상태 오염 방지.
type PrometheusHandler struct {
	reg      *prometheus.Registry
	handler  http.Handler
	httpDurH *prometheus.HistogramVec
}

// NewPrometheusHandler — 격리된 Registry 로 생성.
//
// 노출 지표:
//   - stagesync_room_connected_players (GaugeFunc) — Room 접속자 수
//   - stagesync_optimize_on (GaugeFunc) — 최적화 경로 on/off (1/0)
//   - http_request_duration_seconds (HistogramVec) — HTTP 요청 지연 (method × path × status)
//   - + Go 런타임 collector (goroutines, GC 등)
//   - + process collector (cpu, memory rss 등)
func NewPrometheusHandler(rm *room.Room, opt *lifecycle.Optimize) *PrometheusHandler {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "stagesync_room_connected_players",
			Help: "Number of players currently connected to the WebSocket room.",
		},
		func() float64 { return float64(rm.Size()) },
	))

	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "stagesync_optimize_on",
			Help: "1 when the optimized (pooled) AOI path is active, 0 otherwise.",
		},
		func() float64 {
			if opt.On() {
				return 1
			}
			return 0
		},
	))

	// HTTP 요청 지연 히스토그램.
	// 버킷은 Prometheus 기본 (5ms..10s) — 웹 API 레이턴시 분포에 적합.
	// "path" 레이블은 chi RoutePattern (ex. "/api/event/{id}") 을 씀 — IDs 전개되어 high cardinality 가 되지 않게.
	httpDurH := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency partitioned by method, chi route pattern, and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
	reg.MustRegister(httpDurH)

	return &PrometheusHandler{
		reg:      reg,
		handler:  promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}),
		httpDurH: httpDurH,
	}
}

// Mount — 표준 /metrics 경로에 등록.
func (h *PrometheusHandler) Mount(r chi.Router) {
	r.Get("/metrics", h.handler.ServeHTTP)
}

// HTTPDurationHistogram — RequestMetrics 미들웨어에 주입하기 위한 접근자.
func (h *PrometheusHandler) HTTPDurationHistogram() *prometheus.HistogramVec {
	return h.httpDurH
}
