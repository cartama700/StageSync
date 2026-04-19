package endpoint_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/auth"
	"github.com/kimsehoon/stagesync/internal/endpoint"
)

// newLoginServer — AuthHandler 만 마운트한 httptest 서버.
func newLoginServer(t *testing.T) *httptest.Server {
	t.Helper()
	issuer, err := auth.NewIssuer("test-secret-at-least-32-chars!!", 15*time.Minute)
	require.NoError(t, err)

	h := &endpoint.AuthHandler{Issuer: issuer}
	r := chi.NewRouter()
	h.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestAuthHandler_Login_Success(t *testing.T) {
	t.Parallel()
	srv := newLoginServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(`{"player":"p1"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		PlayerID  string    `json:"player_id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.NotEmpty(t, body.Token)
	require.Equal(t, "p1", body.PlayerID)
	require.True(t, body.ExpiresAt.After(time.Now()))
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	t.Parallel()
	srv := newLoginServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(`{"player":""}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAuthHandler_Login_MalformedJSON(t *testing.T) {
	t.Parallel()
	srv := newLoginServer(t)

	resp, err := http.Post(srv.URL+"/api/auth/login", "application/json",
		strings.NewReader(`{not json`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestRequireAuth_NoValidator_Passthrough —
// AUTH_SECRET 빈 상태 (validator=nil) 에서는 미들웨어가 통과시켜야 함.
func TestRequireAuth_NoValidator_Passthrough(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Use(endpoint.RequireAuth(nil))
	r.Get("/private", func(w http.ResponseWriter, req *http.Request) {
		// ctx 에 claims 가 없으므로 PlayerIDFrom 은 false.
		_, ok := auth.PlayerIDFrom(req.Context())
		require.False(t, ok)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/private")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRequireAuth_MissingHeader_401 — Authorization 헤더 없음 → 401.
func TestRequireAuth_MissingHeader_401(t *testing.T) {
	t.Parallel()
	v := auth.NewValidator("s")
	r := chi.NewRouter()
	r.Use(endpoint.RequireAuth(v))
	r.Get("/private", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/private")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, `Bearer realm="stagesync"`, resp.Header.Get("WWW-Authenticate"))
}

// TestRequireAuth_InvalidToken_401 — 말도 안 되는 토큰 → 401.
func TestRequireAuth_InvalidToken_401(t *testing.T) {
	t.Parallel()
	v := auth.NewValidator("s")
	r := chi.NewRouter()
	r.Use(endpoint.RequireAuth(v))
	r.Get("/private", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/private", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestRequireAuth_ValidToken_InjectsClaims —
// 실제 발급한 토큰으로 호출하면 200 + ctx 에 player_id 주입.
func TestRequireAuth_ValidToken_InjectsClaims(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	issuer, _ := auth.NewIssuer(secret, 5*time.Minute)
	validator := auth.NewValidator(secret)

	token, _, err := issuer.Issue("p42")
	require.NoError(t, err)

	var seenPlayerID string
	r := chi.NewRouter()
	r.Use(endpoint.RequireAuth(validator))
	r.Get("/private", func(w http.ResponseWriter, req *http.Request) {
		pid, ok := auth.PlayerIDFrom(req.Context())
		require.True(t, ok)
		seenPlayerID = pid
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "p42", seenPlayerID)
}

// TestRequireAuth_MalformedAuthHeader_401 — "Bearer" 없는 헤더 → 401.
func TestRequireAuth_MalformedAuthHeader_401(t *testing.T) {
	t.Parallel()
	v := auth.NewValidator("s")
	r := chi.NewRouter()
	r.Use(endpoint.RequireAuth(v))
	r.Get("/private", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/private", nil)
	req.Header.Set("Authorization", "token-without-bearer-prefix")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
