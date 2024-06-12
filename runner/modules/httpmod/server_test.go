package httpmod_test

import (
	"errors"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/phanitejak/gopkg/logging/loggingtest"
	"github.com/phanitejak/gopkg/metrics"
	"github.com/phanitejak/gopkg/runner/modules/httpmod"
	"github.com/phanitejak/gopkg/tracing"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name           string
		opts           []httpmod.Opt
		expectedURL    string
		expectedStatus int
	}{
		{
			name:           "NilOpts",
			opts:           nil,
			expectedURL:    "http://[::]:",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "ZeroOpts",
			opts:           []httpmod.Opt{},
			expectedURL:    "http://[::]:",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "WithAddr",
			opts:           []httpmod.Opt{httpmod.WithAddr("127.0.0.1:0")},
			expectedURL:    "http://127.0.0.1:",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "WithServer",
			opts:           []httpmod.Opt{httpmod.WithServer(&http.Server{Addr: "127.0.0.1:0"})},
			expectedURL:    "http://127.0.0.1:",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "WithHandler",
			opts:           []httpmod.Opt{httpmod.WithHandler(StatusHandler(http.StatusOK))},
			expectedURL:    "http://[::]:",
			expectedStatus: http.StatusOK,
		},
		{
			name: "OptsOrder",
			opts: []httpmod.Opt{
				httpmod.WithHandler(StatusHandler(http.StatusNotFound)),
				httpmod.WithHandler(StatusHandler(http.StatusOK)),
				httpmod.WithHandler(StatusHandler(http.StatusBadRequest)),
			},
			expectedURL:    "http://[::]:",
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			srv := httpmod.NewServer(tt.opts...)
			require.NoError(t, srv.Init(tracing.NewLogger(loggingtest.NewTestLogger(t))), "init failed")
			assert.Contains(t, srv.URL(), tt.expectedURL, "URL doesn't match")

			done := make(chan struct{})
			go func() {
				defer close(done)
				assert.NoError(t, srv.Run(), "run failed")
			}()

			r, err := http.Get(srv.URL())
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, r.StatusCode)
			assert.NoError(t, r.Body.Close())

			require.NoError(t, srv.Close(), "close failed")
			<-done
		})
	}
}

func TestServerWithMetrics(t *testing.T) {
	srv := httpmod.NewServer(httpmod.WithAddr("127.0.0.1:0"), httpmod.WithMetrics())
	require.NoError(t, srv.Init(tracing.NewLogger(loggingtest.NewTestLogger(t))))

	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, srv.Run(), "run failed")
	}()

	tests := []struct {
		path string
		body string
	}{
		{
			path: "/debug/pprof/cmdline",
			body: "httpmod.test",
		},
		{
			path: "/status",
			body: "",
		},
		{
			path: metrics.DefaultEndPoint,
			body: "http_server_active_requests_count",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			url := srv.URL() + tt.path
			resp, err := http.Get(url) //nolint: gosec
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.body)
			require.NoError(t, resp.Body.Close())
		})
	}

	require.NoError(t, srv.Close())
	<-done
}

func TestServerInitErr(t *testing.T) {
	srv := httpmod.NewServer(httpmod.WithAddr("not an address"))
	require.Error(t, srv.Init(tracing.NewLogger(loggingtest.NewTestLogger(t))))
}

func TestServerFromEnv(t *testing.T) {
	const addr = "127.0.0.1:51067"
	require.NoError(t, os.Setenv("HTTP_SERVER_ADDR", addr))
	srv := httpmod.NewServer(httpmod.FromEnv())
	require.NoError(t, srv.Init(tracing.NewLogger(loggingtest.NewTestLogger(t))))
	assert.Equal(t, "http://"+addr, srv.URL())
}

func TestServerErrorOpt(t *testing.T) {
	optErr := errors.New("")
	srv := httpmod.NewServer(func(s *httpmod.Server) error { return optErr })
	require.ErrorIs(t, srv.Init(tracing.NewLogger(loggingtest.NewTestLogger(t))), optErr)
}

func StatusHandler(statusCode int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	})
}
