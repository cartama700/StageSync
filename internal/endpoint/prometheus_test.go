package endpoint_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/api/proto/roompb"
	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/lifecycle"
	"github.com/kimsehoon/stagesync/internal/room"
)

// TestPrometheusHandler_ExposesCustomMetrics —
// Room 크기 + optimize 토글이 지표로 노출되는지 검증.
func TestPrometheusHandler_ExposesCustomMetrics(t *testing.T) {
	t.Parallel()

	rm := room.NewRoom()
	rm.ApplyMove(&roompb.Move{PlayerId: "a"})
	rm.ApplyMove(&roompb.Move{PlayerId: "b"})
	rm.ApplyMove(&roompb.Move{PlayerId: "c"})

	opt := &lifecycle.Optimize{}
	opt.Set(true)

	r := chi.NewRouter()
	endpoint.NewPrometheusHandler(rm, opt).Mount(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(body)

	// Prometheus 포맷: metric_name value \n
	require.Contains(t, text, "stagesync_room_connected_players 3")
	require.Contains(t, text, "stagesync_optimize_on 1")
	// Go collector 도 등록됨 → goroutine 지표 존재.
	require.True(t, strings.Contains(text, "go_goroutines"),
		"go runtime collector 가 등록되어야 함")
}

// TestPrometheusHandler_OptOff — optimize off 상태는 0 으로 노출.
func TestPrometheusHandler_OptOff(t *testing.T) {
	t.Parallel()
	rm := room.NewRoom()
	opt := &lifecycle.Optimize{} // default off

	r := chi.NewRouter()
	endpoint.NewPrometheusHandler(rm, opt).Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	require.Contains(t, string(body), "stagesync_optimize_on 0")
	require.Contains(t, string(body), "stagesync_room_connected_players 0")
}

// TestPrometheusHandler_HistogramRegistered —
// http_request_duration_seconds 가 /metrics 로 노출되어 Prometheus scrape 대상이 되어야 함.
// RequestMetrics 미들웨어 경유로 관측값이 실제로 기록되는지까지 E2E 확인.
func TestPrometheusHandler_HistogramRegistered(t *testing.T) {
	t.Parallel()
	rm := room.NewRoom()
	opt := &lifecycle.Optimize{}

	ph := endpoint.NewPrometheusHandler(rm, opt)

	r := chi.NewRouter()
	r.Use(endpoint.RequestMetrics(ph.HTTPDurationHistogram()))
	ph.Mount(r)
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 먼저 /ping 을 1회 호출해서 histogram 에 관측값을 남김.
	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	// /metrics 스크레이프.
	resp2, err := http.Get(srv.URL + "/metrics")
	require.NoError(t, err)
	defer resp2.Body.Close()
	raw, err := io.ReadAll(resp2.Body)
	require.NoError(t, err)
	text := string(raw)

	// HELP / TYPE 메타데이터가 노출되는지.
	require.Contains(t, text, "http_request_duration_seconds")
	require.Contains(t, text, `path="/ping"`)
	require.Contains(t, text, `method="GET"`)
	require.Contains(t, text, `status="200"`)
}
