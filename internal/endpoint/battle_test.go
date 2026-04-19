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
	battlesvc "github.com/kimsehoon/stagesync/internal/service/battle"
)

func newBattleServer(t *testing.T, impl battlesvc.Implementation) *httptest.Server {
	t.Helper()
	repo := inmem.NewBattleRepo()
	applier := battlesvc.Build(impl, repo)
	h := &endpoint.BattleHandler{Applier: applier, ImplLabel: string(impl)}

	r := chi.NewRouter()
	h.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestBattleHandler_ApplyDamage_Naive(t *testing.T) {
	t.Parallel()
	srv := newBattleServer(t, battlesvc.ImplNaive)

	resp, err := http.Post(srv.URL+"/api/battle/damage", "application/json",
		strings.NewReader(`{"target_player":"p1","damage":100}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		PlayerID string `json:"player_id"`
		HP       int    `json:"hp"`
		Impl     string `json:"impl"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "p1", body.PlayerID)
	require.Equal(t, "naive", body.Impl)
	require.Equal(t, 9900, body.HP) // DefaultInitialHP (10000) - 100
}

func TestBattleHandler_ApplyDamage_Queue(t *testing.T) {
	t.Parallel()
	srv := newBattleServer(t, battlesvc.ImplQueue)

	resp, err := http.Post(srv.URL+"/api/battle/damage", "application/json",
		strings.NewReader(`{"target_player":"p1","damage":100}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Impl string `json:"impl"`
		HP   int    `json:"hp"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Equal(t, "queue", body.Impl)
	require.Equal(t, 9900, body.HP)
}

func TestBattleHandler_InvalidBody(t *testing.T) {
	t.Parallel()
	srv := newBattleServer(t, battlesvc.ImplNaive)

	tests := []struct {
		name string
		body string
	}{
		{"malformed", `{not json`},
		{"missing player", `{"damage":10}`},
		{"missing damage", `{"target_player":"p1"}`},
		{"damage zero", `{"target_player":"p1","damage":0}`},
		{"damage overflow", `{"target_player":"p1","damage":999999}`},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := http.Post(srv.URL+"/api/battle/damage", "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}
