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
	gachasvc "github.com/kimsehoon/stagesync/internal/service/gacha"
)

// newGachaServer — httptest 기반 in-memory 가챠 서버.
func newGachaServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := inmem.NewGachaRepo()
	pools := gachasvc.NewStaticPoolRegistry()
	svc := gachasvc.NewService(repo, pools)
	h := &endpoint.GachaHandler{Service: svc}

	r := chi.NewRouter()
	h.Mount(r)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// rollDTOShape — 응답 파싱용 최소 필드.
type rollDTOShape struct {
	ID       string `json:"id"`
	PlayerID string `json:"player_id"`
	PoolID   string `json:"pool_id"`
	CardID   string `json:"card_id"`
	Rarity   string `json:"rarity"`
	IsPity   bool   `json:"is_pity"`
}

func TestGachaHandler_Roll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantCode int
		wantErr  string // "" = success
	}{
		{
			name:     "valid 10-roll",
			body:     `{"player":"p1","pool":"demo","count":10}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "valid 1-roll",
			body:     `{"player":"p1","pool":"demo","count":1}`,
			wantCode: http.StatusCreated,
		},
		{
			name:     "missing player",
			body:     `{"pool":"demo","count":1}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "missing pool",
			body:     `{"player":"p1","count":1}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "count over 10",
			body:     `{"player":"p1","pool":"demo","count":11}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "count zero",
			body:     `{"player":"p1","pool":"demo","count":0}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "unknown pool",
			body:     `{"player":"p1","pool":"ghost","count":1}`,
			wantCode: http.StatusNotFound,
			wantErr:  "NOT_FOUND",
		},
		{
			name:     "invalid json",
			body:     `not json`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newGachaServer(t)
			resp, err := http.Post(srv.URL+"/api/gacha/roll", "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.wantCode, resp.StatusCode)

			if tc.wantErr != "" {
				var er errResponse
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
				require.Equal(t, tc.wantErr, er.Error.Code)
			}
		})
	}
}

func TestGachaHandler_History_AfterRoll(t *testing.T) {
	t.Parallel()
	srv := newGachaServer(t)

	// 3 roll 실행
	resp, err := http.Post(srv.URL+"/api/gacha/roll", "application/json",
		strings.NewReader(`{"player":"p1","pool":"demo","count":3}`))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// history 조회
	resp, err = http.Get(srv.URL + "/api/gacha/history/p1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var history []rollDTOShape
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&history))
	require.Len(t, history, 3)
	for _, r := range history {
		require.Equal(t, "p1", r.PlayerID)
		require.Equal(t, "demo", r.PoolID)
	}
}

func TestGachaHandler_History_Empty(t *testing.T) {
	t.Parallel()
	srv := newGachaServer(t)

	resp, err := http.Get(srv.URL + "/api/gacha/history/nobody")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var history []rollDTOShape
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&history))
	require.Empty(t, history)
}

func TestGachaHandler_Pity(t *testing.T) {
	t.Parallel()
	srv := newGachaServer(t)

	// 아직 뽑기 전 → counter 0
	resp, err := http.Get(srv.URL + "/api/gacha/pity/p1/demo")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		PlayerID string `json:"player_id"`
		PoolID   string `json:"pool_id"`
		Counter  int    `json:"counter"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "p1", body.PlayerID)
	require.Equal(t, "demo", body.PoolID)
	require.Equal(t, 0, body.Counter)
}
