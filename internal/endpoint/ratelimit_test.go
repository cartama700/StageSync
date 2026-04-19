package endpoint_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/ratelimit"
)

// TestRateLimit_NilLimiter_Passthrough —
// limiter=nil 이면 통과 (rate-limit 비활성 모드).
func TestRateLimit_NilLimiter_Passthrough(t *testing.T) {
	t.Parallel()
	r := chi.NewRouter()
	r.Use(endpoint.RateLimit(nil))
	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	for i := 0; i < 10; i++ {
		resp, err := http.Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
}

// TestRateLimit_AllowsBurst_Then429 —
// burst=2 → 처음 2 개는 200, 3 번째는 429 + Retry-After.
func TestRateLimit_AllowsBurst_Then429(t *testing.T) {
	t.Parallel()
	// 같은 IP (127.0.0.1) 에서 연속 호출되는 testServer → identity 동일.
	limiter := ratelimit.New(0.001, 2) // rps 거의 0 + burst 2 → 버스트 후 전부 거절.
	r := chi.NewRouter()
	r.Use(endpoint.RateLimit(limiter))
	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 처음 2 회는 성공.
	for i := 0; i < 2; i++ {
		resp, err := http.Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusOK, resp.StatusCode, "#%d", i)
	}
	// 3 번째는 429.
	resp, err := http.Get(srv.URL + "/foo")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "1", resp.Header.Get("Retry-After"))
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
}

// TestRateLimit_XForwardedFor_Isolation —
// 다른 XFF 는 별도 bucket → 독립적으로 허용.
func TestRateLimit_XForwardedFor_Isolation(t *testing.T) {
	t.Parallel()
	limiter := ratelimit.New(0.001, 1)
	r := chi.NewRouter()
	r.Use(endpoint.RateLimit(limiter))
	r.Get("/foo", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// IP A 로 1 회 → OK.
	reqA, _ := http.NewRequest(http.MethodGet, srv.URL+"/foo", nil)
	reqA.Header.Set("X-Forwarded-For", "10.0.0.1")
	respA, _ := http.DefaultClient.Do(reqA)
	require.Equal(t, http.StatusOK, respA.StatusCode)
	_ = respA.Body.Close()

	// IP A 로 다시 → 429.
	reqA2, _ := http.NewRequest(http.MethodGet, srv.URL+"/foo", nil)
	reqA2.Header.Set("X-Forwarded-For", "10.0.0.1")
	respA2, _ := http.DefaultClient.Do(reqA2)
	require.Equal(t, http.StatusTooManyRequests, respA2.StatusCode)
	_ = respA2.Body.Close()

	// IP B 로 → OK (독립 bucket).
	reqB, _ := http.NewRequest(http.MethodGet, srv.URL+"/foo", nil)
	reqB.Header.Set("X-Forwarded-For", "10.0.0.2")
	respB, _ := http.DefaultClient.Do(reqB)
	require.Equal(t, http.StatusOK, respB.StatusCode)
	_ = respB.Body.Close()
}

// TestRateLimit_429_BodyFormat — 429 응답 바디에 code/message JSON.
func TestRateLimit_429_BodyFormat(t *testing.T) {
	t.Parallel()
	limiter := ratelimit.New(0.001, 1)
	r := chi.NewRouter()
	r.Use(endpoint.RateLimit(limiter))
	r.Post("/foo", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// 첫 번째는 OK.
	resp1, _ := http.Post(srv.URL+"/foo", "application/json", strings.NewReader(""))
	_ = resp1.Body.Close()
	// 두 번째는 429.
	resp, err := http.Post(srv.URL+"/foo", "application/json", strings.NewReader(""))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "RATE_LIMITED", body["code"])
	require.NotEmpty(t, body["message"])
}
