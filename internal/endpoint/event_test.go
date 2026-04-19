package endpoint_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
	"github.com/kimsehoon/stagesync/internal/persistence/inmem"
	eventsvc "github.com/kimsehoon/stagesync/internal/service/event"
)

func newEventServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := inmem.NewEventRepo()
	svc := eventsvc.NewService(repo)
	h := &endpoint.EventHandler{Service: svc}
	r := chi.NewRouter()
	h.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// ongoingEventBody — 지금 기준으로 진행 중인 이벤트 JSON.
func ongoingEventBody(id string) string {
	start := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	end := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	return fmt.Sprintf(
		`{"id":%q,"name":"E","start_at":%q,"end_at":%q,"rewards":[{"tier":1,"min_points":100,"reward_name":"r1"}]}`,
		id, start, end,
	)
}

func TestEventHandler_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		wantCode int
		wantErr  string
	}{
		{
			name:     "valid ongoing",
			body:     ongoingEventBody("e1"),
			wantCode: http.StatusCreated,
		},
		{
			name:     "missing id",
			body:     `{"name":"E","start_at":"2026-04-19T10:00:00Z","end_at":"2026-04-19T11:00:00Z"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "invalid window",
			body:     `{"id":"e1","name":"E","start_at":"2026-04-19T11:00:00Z","end_at":"2026-04-19T10:00:00Z"}`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
		{
			name:     "invalid json",
			body:     `{nope`,
			wantCode: http.StatusBadRequest,
			wantErr:  "VALIDATION_FAILED",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := newEventServer(t)
			resp, err := http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(tc.body))
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

func TestEventHandler_Duplicate(t *testing.T) {
	t.Parallel()
	srv := newEventServer(t)
	body := ongoingEventBody("dup")

	resp, err := http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestEventHandler_GetAndCurrent(t *testing.T) {
	t.Parallel()
	srv := newEventServer(t)

	resp, err := http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(ongoingEventBody("e1")))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// GET /api/event/e1
	resp, err = http.Get(srv.URL + "/api/event/e1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	require.Equal(t, "e1", got.ID)
	require.Equal(t, "ONGOING", got.Status)

	// GET /api/event/current
	resp, err = http.Get(srv.URL + "/api/event/current")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
	require.Len(t, list, 1)
	require.Equal(t, "e1", list[0].ID)
}

func TestEventHandler_GetNotFound(t *testing.T) {
	t.Parallel()
	srv := newEventServer(t)
	resp, err := http.Get(srv.URL + "/api/event/missing")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestEventHandler_AddScoreAndRewards(t *testing.T) {
	t.Parallel()
	srv := newEventServer(t)

	resp, err := http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(ongoingEventBody("e1")))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// 점수 누적.
	resp, err = http.Post(srv.URL+"/api/event/e1/score", "application/json",
		strings.NewReader(`{"player":"p1","delta":150}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var sc struct {
		Points int64 `json:"points"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sc))
	require.EqualValues(t, 150, sc.Points)

	// 점수 조회.
	resp, err = http.Get(srv.URL + "/api/event/e1/score/p1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 보상 조회 (진행 중이므로 claimable=false, 100 티어 달성).
	resp, err = http.Get(srv.URL + "/api/event/e1/rewards/p1")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rw struct {
		Status    string `json:"status"`
		Points    int64  `json:"points"`
		Tiers     []any  `json:"tiers"`
		Eligible  []any  `json:"eligible"`
		Claimable bool   `json:"claimable"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&rw))
	require.Equal(t, "ONGOING", rw.Status)
	require.EqualValues(t, 150, rw.Points)
	require.Len(t, rw.Tiers, 1)
	require.Len(t, rw.Eligible, 1)
	require.False(t, rw.Claimable)
}

func TestEventHandler_AddScore_InvalidDelta(t *testing.T) {
	t.Parallel()
	srv := newEventServer(t)

	resp, err := http.Post(srv.URL+"/api/event", "application/json", strings.NewReader(ongoingEventBody("e1")))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// delta=0 → validator min=1 → 400.
	resp, err = http.Post(srv.URL+"/api/event/e1/score", "application/json",
		strings.NewReader(`{"player":"p1","delta":0}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
