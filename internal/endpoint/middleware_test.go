package endpoint_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"github.com/kimsehoon/stagesync/internal/endpoint"
)

// TestRequestLogger_InjectsLoggerWithRequestID —
// 미들웨어가 ctx 에 logger 를 주입하고 request_id 가 로그에 포함됨.
func TestRequestLogger_InjectsLoggerWithRequestID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(endpoint.RequestLogger(logger))

	var seenReqID string
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		l := endpoint.LoggerFrom(req.Context())
		require.NotNil(t, l)
		l.Info("handler-called")
		seenReqID = middleware.GetReqID(req.Context())
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/ping")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEmpty(t, seenReqID)

	// 로그에 "request_id" 키가 포함되어야 함 (핸들러 로그 + access 로그 둘 다).
	logs := strings.TrimSpace(buf.String())
	require.NotEmpty(t, logs)
	lines := strings.Split(logs, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "handler + access log")
	for _, line := range lines {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m))
		require.Equal(t, seenReqID, m["request_id"])
	}
}

// TestLoggerFrom_Fallback — ctx 에 logger 없으면 slog.Default() 반환 (nil 아님).
func TestLoggerFrom_Fallback(t *testing.T) {
	t.Parallel()
	l := endpoint.LoggerFrom(context.Background())
	require.NotNil(t, l)
}
