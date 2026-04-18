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
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	profilesvc "github.com/kimsehoon/stagesync/internal/service/profile"
)

// newProfileServer — httptest 기반 in-memory HTTP 서버 조립.
// 각 테스트가 독립 서버 + 독립 repo 를 가져 t.Parallel 안전.
func newProfileServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := inmem.NewProfileRepo()
	svc := profilesvc.NewService(repo)
	h := &endpoint.ProfileHandler{Service: svc}

	r := chi.NewRouter()
	h.Mount(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// errResponse — apperror.WriteJSON 의 응답 스키마 (테스트 전용 부분 필드).
type errResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Fields  []struct {
			Field string `json:"field"`
			Tag   string `json:"tag"`
		} `json:"fields,omitempty"`
	} `json:"error"`
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	return resp
}

func TestProfileHandler_Create(t *testing.T) {
	t.Parallel()

	longName := strings.Repeat("x", 200)

	tests := []struct {
		name    string
		body    string
		status  int
		errCode string // "" = success
	}{
		{name: "valid", body: `{"id":"p1","name":"sekai"}`, status: http.StatusCreated},
		{name: "empty body", body: `{}`, status: http.StatusBadRequest, errCode: "VALIDATION_FAILED"},
		{name: "missing id", body: `{"name":"sekai"}`, status: http.StatusBadRequest, errCode: "VALIDATION_FAILED"},
		{name: "missing name", body: `{"id":"p1"}`, status: http.StatusBadRequest, errCode: "VALIDATION_FAILED"},
		{name: "invalid json", body: `not json`, status: http.StatusBadRequest, errCode: "VALIDATION_FAILED"},
		{name: "name too long", body: `{"id":"p1","name":"` + longName + `"}`, status: http.StatusBadRequest, errCode: "VALIDATION_FAILED"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newProfileServer(t)
			resp := postJSON(t, srv.URL+"/api/profile", tc.body)
			defer resp.Body.Close()

			require.Equal(t, tc.status, resp.StatusCode)
			if tc.errCode == "" {
				return
			}
			var er errResponse
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
			require.Equal(t, tc.errCode, er.Error.Code)
		})
	}
}

// TestProfileHandler_GetAfterCreate — 생성 후 조회 흐름 (한 서버 내 state 검증).
func TestProfileHandler_GetAfterCreate(t *testing.T) {
	t.Parallel()
	srv := newProfileServer(t)

	// Create
	resp := postJSON(t, srv.URL+"/api/profile", `{"id":"p1","name":"sekai"}`)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get
	resp, err := http.Get(srv.URL + "/api/profile/p1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var p struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&p))
	require.Equal(t, "p1", p.ID)
	require.Equal(t, "sekai", p.Name)
}

func TestProfileHandler_GetNotFound(t *testing.T) {
	t.Parallel()
	srv := newProfileServer(t)

	resp, err := http.Get(srv.URL + "/api/profile/ghost")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var er errResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	require.Equal(t, "NOT_FOUND", er.Error.Code)
}

func TestProfileHandler_DuplicateConflict(t *testing.T) {
	t.Parallel()
	srv := newProfileServer(t)

	// 1차 — 성공
	resp := postJSON(t, srv.URL+"/api/profile", `{"id":"p1","name":"sekai"}`)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 2차 동일 ID — Conflict
	resp = postJSON(t, srv.URL+"/api/profile", `{"id":"p1","name":"other"}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var er errResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	require.Equal(t, "CONFLICT", er.Error.Code)
}
