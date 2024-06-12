// Package metrics provides unified functions to expose application level
// metrics from the NEO services.
package metrics

import (
	"encoding/json"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const metricNamespace = "com_metrics"

// DefaultEndPoint is the default endpoint for the exposed metrics.
const (
	DefaultEndPoint                = "/application/prometheus"
	statusEndPoint                 = "/status"
	metricHTTPActiveRequestsName   = "http_server_active_requests_count"
	metricHTTPRequestsDurationName = "http_server_requests_duration_seconds"
	metricHTTPResponsesSizeName    = "http_server_responses_size_bytes"
	metricHTTPRequestsSizeName     = "http_server_requests_size_bytes"
	plainMetricNameKey             = "_plain_metric_name"
)

var (
	rulePattern = regexp.MustCompile(`(?s)(\{[^}]*\})`)

	gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: metricHTTPActiveRequestsName,
		Help: "Count of http requests currently being served by method and URI.",
	}, []string{"method", "uri"})
	obs = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: metricHTTPRequestsDurationName,
			Help: "Total time and count of http requests by status code, " +
				"method and URI in seconds.",
		},
		[]string{"status", "method", "uri"},
	)
	obsResponseSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: metricHTTPResponsesSizeName,
			Help: "Total size and count of http responses by status code, " +
				"method and URI in bytes.",
		},
		[]string{"status", "method", "uri"},
	)
	obsRequestSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: metricHTTPRequestsSizeName,
			Help: "Total size and count of http requests by status code, " +
				"method and URI in bytes.",
		},
		[]string{"status", "method", "uri"},
	)
	commonMetricsCollector = newDefaultCollector()
)

//nolint:gochecknoinits
func init() {
	prometheus.MustRegister(gauge, obs, obsResponseSize, obsRequestSize, commonMetricsCollector)
}

// CustomMetric is a provider for collector.
type CustomMetric interface {
	GetCollector() prometheus.Collector
}

// InstrumentRule combines regexp trigger condition and matching value,.
type InstrumentRule struct {
	Condition *regexp.Regexp
	URIPath   string
}

type loggingStatusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	length     int64
}

// ManagementServer type is for gracefully stop the management server.
type ManagementServer struct {
	server *http.Server
	wg     *sync.WaitGroup
}

type swaggerSpecURLPaths struct {
	BasePath string                     `json:"basePath"`
	Paths    map[string]json.RawMessage `json:"paths"`
}

// Router is an interface used for pprof instrumentation.
type Router interface {
	Handle(pattern string, handler http.Handler)
}

// Close closes and waits until the ManagementServer is gracefully closed.
func (managementServer *ManagementServer) Close() {
	managementServer.server.Close()
	managementServer.wg.Wait()
}

func (lrw *loggingStatusCodeResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingStatusCodeResponseWriter) Flush() {
	f, ok := lrw.ResponseWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (n int, err error) {
	n, err = lrw.ResponseWriter.Write(b)
	lrw.length += int64(n)
	return
}

func (lrw *loggingResponseWriter) Flush() {
	f, ok := lrw.ResponseWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

// GetMetricsHandler gets metric handler in case you want embed metrics endpoint
// to your existing HTTP server.
func GetMetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartManagementServer starts HTTP server for metric endpoint, pprof endpoints
// and optionally for health-check endpoint. Use this in case you don't want to
// embed these to your service's business endpoints.
// Function returns ManagementServer for stopping management server gracefully.
func StartManagementServer(listenAddress string, healthCheckFunc func(http.ResponseWriter, *http.Request)) (managementServer *ManagementServer) {
	mux := http.NewServeMux()
	mux.Handle(DefaultEndPoint, GetMetricsHandler())
	InstrumentWithPprof(mux)
	if healthCheckFunc != nil {
		mux.HandleFunc(statusEndPoint, healthCheckFunc)
	}
	managementServer = &ManagementServer{
		server: &http.Server{
			Addr:    listenAddress,
			Handler: InstrumentHTTPHandler(mux),
		},
		wg: &sync.WaitGroup{},
	}
	listener, err := net.Listen("tcp", managementServer.server.Addr)
	if err != nil {
		panic("Management server error: " + err.Error())
	}
	managementServer.wg.Add(1)
	go func() {
		defer managementServer.wg.Done()
		err := managementServer.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			panic("Management server error: " + err.Error())
		}
	}()
	return
}

// InstrumentWithPprof instruments given Router with pprof profiling endpoints.
func InstrumentWithPprof(mux Router) {
	mux.Handle("/debug/pprof/", http.HandlerFunc(httppprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(httppprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(httppprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(httppprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(httppprof.Trace))
	mux.Handle("/debug/pprof/goroutine", httppprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", httppprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", httppprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", httppprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", httppprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", httppprof.Handler("allocs"))
}

// InstrumentHTTPHandler instruments HTTP handler to expose metrics related to
// request/response count, size and times.
func InstrumentHTTPHandler(next http.Handler) http.Handler {
	var noRules []InstrumentRule
	handler := InstrumentHTTPHandlerWithRules(next, noRules)
	return handler
}

// InstrumentHTTPHandlerWithRules instruments HTTP handler to expose metrics related to
// request/response count, size and times.
// Applies routings according to the given rules.
func InstrumentHTTPHandlerWithRules(handler http.Handler, rules []InstrumentRule) http.Handler {
	handler = instrumentHTTPHandlerInFlight(gauge, handler, rules)
	handler = instrumentHTTPHandlerDuration(obs, handler, rules)
	handler = instrumentHTTPHandlerResponseSize(obsResponseSize, handler, rules)
	handler = instrumentHTTPHandlerRequestSize(obsRequestSize, handler, rules)
	return handler
}

// MustInstrumentHTTPHandlerWithSwaggerSpec instruments HTTP handler to expose metrics related to
// request/response count, size and times. Rest API URL path variable parts are normalized to
// variable names by using given Swagger JSON object. In case Swagger JSON is invalid, a panic will happen.
func MustInstrumentHTTPHandlerWithSwaggerSpec(next http.Handler, swaggerSpec json.RawMessage) http.Handler {
	rules, err := BuildRulesFromSwaggerSpec(swaggerSpec)
	if err != nil {
		panic(err)
	}

	handler := InstrumentHTTPHandlerWithRules(next, rules)

	return handler
}

// InstrumentHTTPHandlerUsingSwaggerSpec instruments HTTP handler to expose metrics related to
// request/response count, size and times. Rest API URL path variable parts are normalized to
// variable names by using given Swagger JSON object. In case Swagger JSON is invalid, a panic
// will happen.
//
// Deprecated: In case you want panic happen when Swagger JSON is invalid, use
// MustInstrumentHTTPHandlerWithSwaggerSpec. In case you want to handle error,
// use function InstrumentHTTPHandlerWithSwaggerSpec.
func InstrumentHTTPHandlerUsingSwaggerSpec(next http.Handler, swaggerSpec json.RawMessage) http.Handler {
	return MustInstrumentHTTPHandlerWithSwaggerSpec(next, swaggerSpec)
}

// InstrumentHTTPHandlerWithSwaggerSpec instruments HTTP handler to expose metrics related to
// request/response count, size and times. Rest API URL path variable parts are normalized to
// variable names by using given Swagger JSON object. In case Swagger JSON is invalid, an error is returned.
func InstrumentHTTPHandlerWithSwaggerSpec(next http.Handler, swaggerSpec json.RawMessage) (http.Handler, error) {
	rules, err := BuildRulesFromSwaggerSpec(swaggerSpec)
	if err != nil {
		return nil, err
	}

	handler := InstrumentHTTPHandlerWithRules(next, rules)

	return handler, nil
}

// BuildRulesFromSwaggerSpec builds rules from the paths unmarshaled from swagger json. Returns error
// if the unmarshalling of url paths from swagger psec fails.
func BuildRulesFromSwaggerSpec(swaggerSpec json.RawMessage) ([]InstrumentRule, error) {
	paths := &swaggerSpecURLPaths{}
	if err := json.Unmarshal(swaggerSpec, paths); err != nil {
		return nil, err
	}

	var rules []InstrumentRule
	for uri := range paths.Paths {
		if strings.Contains(uri, "{") {
			uriPath := path.Join(paths.BasePath, uri)
			rules = append(rules, InstrumentRule{Condition: regexp.MustCompile(rulePattern.ReplaceAllString("^"+uriPath+"$", `[^/]+`)), URIPath: uriPath})
		}
	}
	return rules, nil
}

func instrumentHTTPHandlerInFlight(gauge *prometheus.GaugeVec,
	next http.Handler, rules []InstrumentRule) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g := gauge.WithLabelValues(r.Method, getURIApplyingRules(r.URL, rules))
		g.Inc()
		defer g.Dec()
		next.ServeHTTP(w, r)
	})
}

func instrumentHTTPHandlerDuration(obs prometheus.ObserverVec,
	next http.Handler, rules []InstrumentRule) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 is the default code if w.WriteHeader() isn't called explicitly
		now := time.Now()
		lrw := &loggingStatusCodeResponseWriter{w, 200}
		next.ServeHTTP(lrw, r)
		obs.WithLabelValues(strconv.Itoa(lrw.statusCode), r.Method, getURIApplyingRules(r.URL, rules)).Observe(
			time.Since(now).Seconds())
	})
}

func instrumentHTTPHandlerResponseSize(obs prometheus.ObserverVec,
	next http.Handler, rules []InstrumentRule) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 is the default code if w.WriteHeader() isn't called explicitly
		lrw := &loggingResponseWriter{w, 200, 0}
		next.ServeHTTP(lrw, r)
		obs.WithLabelValues(strconv.Itoa(lrw.statusCode), r.Method, getURIApplyingRules(r.URL, rules)).Observe(
			float64(lrw.length))
	})
}

func instrumentHTTPHandlerRequestSize(obs prometheus.ObserverVec,
	next http.Handler, rules []InstrumentRule) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 is the default code if w.WriteHeader() isn't called explicitly
		lrw := &loggingStatusCodeResponseWriter{w, 200}
		next.ServeHTTP(lrw, r)
		size := computeApproximateRequestSize(r)
		obs.WithLabelValues(strconv.Itoa(lrw.statusCode), r.Method, getURIApplyingRules(r.URL, rules)).Observe(
			float64(size))
	})
}

// Builds URI applying given rules.
func getURIApplyingRules(url *url.URL, rules []InstrumentRule) string {
	var path string
	if url.RawPath != "" {
		path = url.RawPath
	} else {
		path = url.Path
	}
	for _, rule := range rules {
		if rule.Condition.MatchString(path) {
			return rule.URIPath
		}
	}
	return url.Path
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
