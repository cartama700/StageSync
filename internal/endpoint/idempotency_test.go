package endpoint_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/idempotency"
)

// TestIdempotency_NilStore_Passthrough — store=nil 이면 아무것도 안 함.
func TestIdempotency_NilStore_Passthrough(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Use(endpoint.Idempotency(nil))
	r.Post("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL+"/foo", "application/json", strings.NewReader(""))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Empty(t, resp.Header.Get("Idempotency-Replayed"))
}

// TestIdempotency_GET_Passthrough — GET 은 캐시 대상 아님.
func TestIdempotency_GET_Passthrough(t *testing.T) {
	t.Parallel()
	store := idempotency.NewInmemStore(5 * time.Minute)
	var calls int32

	r := chi.NewRouter()
	r.Use(endpoint.Idempotency(store))
	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 같은 키로 2 회 GET 호출 → 핸들러 2 회 실행.
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/foo", nil)
		req.Header.Set("Idempotency-Key", "k-1")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "GET 은 캐시 대상 아님")
}

// TestIdempotency_MissingKey_Passthrough — 헤더 없음 → 일반 처리.
func TestIdempotency_MissingKey_Passthrough(t *testing.T) {
	t.Parallel()
	store := idempotency.NewInmemStore(5 * time.Minute)
	var calls int32

	r := chi.NewRouter()
	r.Use(endpoint.Idempotency(store))
	r.Post("/foo", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	for i := 0; i < 2; i++ {
		resp, err := http.Post(srv.URL+"/foo", "application/json", strings.NewReader(""))
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	require.Equal(t, int32(2), atomic.LoadInt32(&calls), "헤더 없으면 매번 실행")
}

// TestIdempotency_Replay — 같은 키로 2 회 호출 → 두 번째는 캐시 리플레이.
func TestIdempotency_Replay(t *testing.T) {
	t.Parallel()
	store := idempotency.NewInmemStore(5 * time.Minute)
	var calls int32

	r := chi.NewRouter()
	r.Use(endpoint.Idempotency(store))
	r.Post("/roll", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"roll":"ssr"}`))
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 1 차 요청.
	req1, _ := http.NewRequest(http.MethodPost, srv.URL+"/roll", strings.NewReader(""))
	req1.Header.Set("Idempotency-Key", "abc-123")
	resp1, err := http.DefaultClient.Do(req1)
	require.NoError(t, err)
	body1, _ := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	require.Empty(t, resp1.Header.Get("Idempotency-Replayed"))

	// 2 차 요청 (같은 key) → 리플레이.
	req2, _ := http.NewRequest(http.MethodPost, srv.URL+"/roll", strings.NewReader(""))
	req2.Header.Set("Idempotency-Key", "abc-123")
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()

	require.Equal(t, http.StatusCreated, resp2.StatusCode)
	require.Equal(t, "true", resp2.Header.Get("Idempotency-Replayed"))
	require.Equal(t, body1, body2, "바디 동일")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "핸들러는 1 회만 실행")
}

// TestIdempotency_DifferentKey_DifferentResponse —
// 다른 키는 독립 — 각각 별도 핸들러 실행.
func TestIdempotency_DifferentKey_DifferentResponse(t *testing.T) {
	t.Parallel()
	store := idempotency.NewInmemStore(5 * time.Minute)
	var calls int32

	r := chi.NewRouter()
	r.Use(endpoint.Idempotency(store))
	r.Post("/roll", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	for _, k := range []string{"a", "b", "c"} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/roll", strings.NewReader(""))
		req.Header.Set("Idempotency-Key", k)
		resp, _ := http.DefaultClient.Do(req)
		_ = resp.Body.Close()
	}
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
}
