package endpoint_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	rankingsvc "github.com/kimsehoon/stagesync/internal/service/ranking"
)

// newRankingServer — inmem leaderboard 에 시드 + httptest 서버 1 줄 반환.
func newRankingServer(t *testing.T) *httptest.Server {
	t.Helper()
	lb := inmem.NewLeaderboard()
	ctx := context.Background()
	for _, s := range []struct {
		player string
		delta  int64
	}{
		{"alice", 500},
		{"bob", 300},
		{"carol", 100},
	} {
		_, err := lb.IncrBy(ctx, "ev1", s.player, s.delta)
		require.NoError(t, err)
	}
	svc := rankingsvc.NewService(lb)
	h := &endpoint.RankingHandler{Service: svc}

	r := chi.NewRouter()
	h.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func TestRankingHandler_Top_Default(t *testing.T) {
	t.Parallel()
	srv := newRankingServer(t)

	resp, err := http.Get(srv.URL + "/api/ranking/ev1/top")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		EventID string `json:"event_id"`
		Count   int    `json:"count"`
		Entries []struct {
			PlayerID string `json:"player_id"`
			Score    int64  `json:"score"`
			Rank     int    `json:"rank"`
		} `json:"entries"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	require.Equal(t, "ev1", body.EventID)
	require.Equal(t, 3, body.Count)
	require.Equal(t, "alice", body.Entries[0].PlayerID)
	require.EqualValues(t, 500, body.Entries[0].Score)
	require.Equal(t, 1, body.Entries[0].Rank)
}

func TestRankingHandler_Top_NOverMax(t *testing.T) {
	t.Parallel()
	srv := newRankingServer(t)

	resp, err := http.Get(srv.URL + "/api/ranking/ev1/top?n=200")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRankingHandler_Around_Self(t *testing.T) {
	t.Parallel()
	srv := newRankingServer(t)

	resp, err := http.Get(srv.URL + "/api/ranking/ev1/me/bob?radius=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		EventID  string `json:"event_id"`
		PlayerID string `json:"player_id"`
		Rank     int    `json:"rank"`
		Score    int64  `json:"score"`
		Radius   int    `json:"radius"`
		Entries  []struct {
			PlayerID string `json:"player_id"`
			Rank     int    `json:"rank"`
		} `json:"entries"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	require.Equal(t, "bob", body.PlayerID)
	require.Equal(t, 2, body.Rank)
	require.EqualValues(t, 300, body.Score)
	// ±1 → alice, bob, carol.
	require.Len(t, body.Entries, 3)
	require.Equal(t, "alice", body.Entries[0].PlayerID)
	require.Equal(t, "bob", body.Entries[1].PlayerID)
	require.Equal(t, "carol", body.Entries[2].PlayerID)
}

func TestRankingHandler_Around_NotFound(t *testing.T) {
	t.Parallel()
	srv := newRankingServer(t)

	resp, err := http.Get(srv.URL + "/api/ranking/ev1/me/ghost")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	raw, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(raw), "NOT_FOUND")
}

func TestRankingHandler_Around_RadiusOutOfRange(t *testing.T) {
	t.Parallel()
	srv := newRankingServer(t)

	resp, err := http.Get(srv.URL + "/api/ranking/ev1/me/bob?radius=999")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
