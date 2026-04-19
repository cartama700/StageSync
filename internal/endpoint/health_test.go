package endpoint_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/lifecycle"
)

// newHealthRouter — 테스트 용 헬퍼. State 주입 여부에 따라 Mount.
func newHealthRouter(state *lifecycle.Readiness) http.Handler {
	r := chi.NewRouter()
	h := &endpoint.HealthHandler{State: state}
	h.Mount(r)
	return r
}

// TestHealthHandler_Live_Always200 — /health/live 는 drain 중에도 200.
// liveness 는 "프로세스 살아 있는가" 만 판단 — drain 중에도 pod 재시작 당하면 안 됨.
func TestHealthHandler_Live_Always200(t *testing.T) {
	t.Parallel()

	state := lifecycle.NewReadiness()
	state.SetDraining() // drain 상태여도 live 는 200.

	srv := httptest.NewServer(newHealthRouter(state))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/health/live")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestHealthHandler_Ready_NilState — State 주입 안 한 경우 Ready 는 항상 200 (fallback).
func TestHealthHandler_Ready_NilState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newHealthRouter(nil))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/health/ready")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestHealthHandler_Ready_Live — State.Ready()==true 면 200.
func TestHealthHandler_Ready_Live(t *testing.T) {
	t.Parallel()

	state := lifecycle.NewReadiness()
	srv := httptest.NewServer(newHealthRouter(state))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/health/ready")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestHealthHandler_Ready_Draining — SetDraining 후엔 503 + {"ready": false}.
func TestHealthHandler_Ready_Draining(t *testing.T) {
	t.Parallel()

	state := lifecycle.NewReadiness()
	state.SetDraining()

	srv := httptest.NewServer(newHealthRouter(state))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/health/ready")
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, false, payload["ready"])
}
