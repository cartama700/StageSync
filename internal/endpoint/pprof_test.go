package endpoint_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
)

// TestPprofMount — chi middleware.Profiler() 가 /debug/pprof/ 하위 경로들을 올바르게 제공하는지
// smoke test. 상세 프로파일 내용은 검증하지 않고 "접근 가능" 까지만 확인.
//
// main.go 와 동일한 마운트 방식 (r.Mount("/debug", middleware.Profiler())) 을 재현.
func TestPprofMount(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	r.Mount("/debug", chimw.Profiler())

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	tests := []struct {
		name    string
		path    string
		wantOK  bool
		wantSub string // 응답 본문에 포함되어야 하는 서브스트링 (GET 200 일 때만)
	}{
		{name: "index", path: "/debug/pprof/", wantOK: true, wantSub: "profiles"},
		{name: "cmdline", path: "/debug/pprof/cmdline", wantOK: true},
		{name: "goroutine (debug=1)", path: "/debug/pprof/goroutine?debug=1", wantOK: true, wantSub: "goroutine profile"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := http.Get(srv.URL + tc.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			if tc.wantOK {
				require.Equal(t, http.StatusOK, resp.StatusCode, "path=%s", tc.path)
				if tc.wantSub != "" {
					body, _ := io.ReadAll(resp.Body)
					require.True(t, strings.Contains(string(body), tc.wantSub),
						"body missing %q:\n%s", tc.wantSub, string(body))
				}
			}
		})
	}
}
