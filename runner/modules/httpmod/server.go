// Package httpmod provides http.Server as module.
package httpmod

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/kelseyhightower/envconfig"

	"gopkg/metrics"
	"gopkg/tracing"
)

// Opt can be used to modify servers configuration.
type Opt func(s *Server) error

// FromEnv tries to uses value of HTTP_SERVER_ADDR env variable with ':8080' as default address.
func FromEnv() Opt {
	return func(s *Server) error {
		addr := struct {
			Addr string `envconfig:"HTTP_SERVER_ADDR" default:":8080"`
		}{}

		if err := envconfig.Process("", &addr); err != nil {
			return fmt.Errorf("loading server config from env failed: %w", err)
		}

		s.srv.Addr = addr.Addr
		return nil
	}
}

// WithServer allows setting pointer for http.Server to be used.
func WithServer(srv *http.Server) Opt {
	return func(s *Server) error {
		s.srv = srv
		return nil
	}
}

// WithAddr allows setting listening address of http.Server.
func WithAddr(addr string) Opt {
	return func(s *Server) error {
		s.srv.Addr = addr
		return nil
	}
}

// WithHandler allows setting handler of http.Server.
func WithHandler(h http.Handler) Opt {
	return func(s *Server) error {
		s.srv.Handler = h
		return nil
	}
}

// WithManagementServer can be used expose management endpoints
// when the service doesn't have any HTTP API itself.
// It will expose:
//  1. prometheus metrics
//  2. pprof endpoints
//  3. status endpoint
func WithManagementServer() Opt {
	return func(s *Server) error {
		mux := http.NewServeMux()
		mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
		metrics.InstrumentWithPprof(mux)
		mux.HandleFunc("/status", func(http.ResponseWriter, *http.Request) {})
		s.srv.Handler = metrics.InstrumentHTTPHandler(mux)
		return nil
	}
}

// WithMetrics can be used when service doesn't need http server other wise.
// It will expose:
//  1. prometheus metrics
//  2. pprof endpoints
//  3. status endpoint
func WithMetrics() Opt {
	return func(s *Server) error {
		mux := http.NewServeMux()
		mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
		metrics.InstrumentWithPprof(mux)
		mux.HandleFunc("/status", func(http.ResponseWriter, *http.Request) {})
		s.srv.Handler = metrics.InstrumentHTTPHandler(mux)
		return nil
	}
}

// Server wraps http.Server as module.
type Server struct {
	srv  *http.Server
	ln   net.Listener
	opts []Opt
}

// NewServer creates new instance of Server with given options.
func NewServer(opts ...Opt) *Server {
	return &Server{
		opts: opts,
	}
}

// Init applies all options and calls calls net.Listen with Servers address.
func (s *Server) Init(_ *tracing.Logger) error {
	s.srv = &http.Server{}
	for _, opt := range s.opts {
		if err := opt(s); err != nil {
			return fmt.Errorf("failed to applie option for httpmod.Server: %w", err)
		}
	}

	ln, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("failed to init listener: %w", err)
	}

	s.ln = ln
	return nil
}

// URL return http address of server. It can be used only after Init() is called.
func (s *Server) URL() string {
	return "http://" + s.ln.Addr().String()
}

// Run starts serving on initialized listener.
func (s *Server) Run() error {
	err := s.srv.Serve(s.ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Close shutsdown server gracefully.
func (s *Server) Close() error {
	return s.srv.Shutdown(context.Background())
}
