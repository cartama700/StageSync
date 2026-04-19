package endpoint_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
)

// TestRequestMetrics_ObservesRoutePattern —
// `/api/foo/42` 요청이 RoutePattern `/api/foo/{id}` 로 집계돼야 함 (high cardinality 방지).
// 200 과 404 두 경로 모두 Histogram 에 기록되는지 확인.
func TestRequestMetrics_ObservesRoutePattern(t *testing.T) {
	t.Parallel()

	hist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "test_http_request_duration_seconds",
		Help: "test",
	}, []string{"method", "path", "status"})

	r := chi.NewRouter()
	r.Use(endpoint.RequestMetrics(hist))
	r.Get("/api/foo/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 200 케이스.
	resp, err := http.Get(srv.URL + "/api/foo/42")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 404 케이스 — 미매칭 경로.
	resp2, err := http.Get(srv.URL + "/does-not-exist")
	require.NoError(t, err)
	require.NoError(t, resp2.Body.Close())
	require.Equal(t, http.StatusNotFound, resp2.StatusCode)

	// Histogram 출력 확인 — Prometheus exposition 포맷으로 덤프해서 검사.
	reg := prometheus.NewRegistry()
	reg.MustRegister(hist)

	dump := dumpRegistry(t, reg)
	require.Contains(t, dump, `path="/api/foo/{id}"`, "ID 가 전개되지 않고 pattern 으로 집계되어야 함")
	require.Contains(t, dump, `method="GET"`)
	require.Contains(t, dump, `status="200"`)
	// 404 는 RoutePattern 이 비어 있으므로 "(unknown)" 으로 집계.
	require.Contains(t, dump, `path="(unknown)"`)
	require.Contains(t, dump, `status="404"`)
}

func dumpRegistry(t *testing.T, reg *prometheus.Registry) string {
	t.Helper()
	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	return strings.TrimSpace(string(body))
}
