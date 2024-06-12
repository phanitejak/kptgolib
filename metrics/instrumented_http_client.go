package metrics

import (
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	clientMetricHTTPRequestsDurationName = "http_client_requests_seconds"
	clientMetricHTTPResponsesSizeName    = "http_client_responses_size_bytes"
	clientMetricHTTPRequestsSizeName     = "http_client_requests_size_bytes"
)

var (
	clientDuration = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: clientMetricHTTPRequestsDurationName,
			Help: "Total time and count of http requests by status code, " +
				"method, URI and host in seconds.",
		},
		[]string{"status", "method", "uri", "clientName"},
	)
	clientRespSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: clientMetricHTTPResponsesSizeName,
			Help: "Total size and count of http responses by status code, " +
				"method, URI and host in bytes.",
		},
		[]string{"status", "method", "uri", "clientName"},
	)
	clientRequestSize = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: clientMetricHTTPRequestsSizeName,
			Help: "Total size and count of http requests by status code, " +
				"method, URI and host in bytes.",
		},
		[]string{"status", "method", "uri", "clientName"},
	)
)

//nolint:gochecknoinits
func init() {
	prometheus.MustRegister(clientDuration, clientRespSize, clientRequestSize)
}

// A InstrumentedHttpClient represents standard http.Client with metrics instrumentation capabilities.
type InstrumentedHttpClient struct {
	client *http.Client
	rules  []InstrumentRule
}

// A HttpRequestTemplate represents standard http.Request with URL templating capabilities.
type HttpRequestTemplate struct {
	Request     *http.Request
	UrlTemplate string
}

// NewInstrumentedHttpClient returns given http client with instrumentation capabilities.
func NewInstrumentedHttpClient(httpClient *http.Client) *InstrumentedHttpClient {
	return &InstrumentedHttpClient{httpClient, nil}
}

// NewInstrumentedDefaultHttpClient returns default http client with instrumentation capabilities.
func NewInstrumentedDefaultHttpClient() *InstrumentedHttpClient {
	return &InstrumentedHttpClient{http.DefaultClient, nil}
}

// NewHttpRequestTemplate returns a new HttpRequestTemplate given a method, URL, optional body and urlVariables.
// This can be then used as an argument for InstrumentedHttpClient.Do method.
func NewHttpRequestTemplate(method, urlTemplate string, body io.Reader, urlVariables ...string) (*HttpRequestTemplate, error) {
	req, err := http.NewRequest(method, expandURL(urlTemplate, urlVariables), body)
	return &HttpRequestTemplate{req, urlTemplate}, err
}

// NewHttpRequestTemplateFromRequest returns a new HttpRequestTemplate given request and urlVariables.
// This can be then used as an argument for InstrumentedHttpClient.Do method.
func NewHttpRequestTemplateFromRequest(request *http.Request, urlVariables ...string) (*HttpRequestTemplate, error) {
	rawURL, err := url.QueryUnescape(request.URL.String())
	if len(urlVariables) > 0 {
		if err != nil {
			return nil, err
		}
		req := new(http.Request)
		*req = *request
		req.URL, err = url.Parse(expandURL(rawURL, urlVariables))
		return &HttpRequestTemplate{req, rawURL}, err
	}
	return &HttpRequestTemplate{request, rawURL}, err
}

// SetRules sets global URI templating rules for the InstrumentedHttpClient across all requests
func (hc *InstrumentedHttpClient) SetRules(rules ...InstrumentRule) {
	hc.rules = rules
}

// Get is a metric instrumentation wrapper for Client.Get with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Get method documentation for details.
func (hc *InstrumentedHttpClient) Get(urlTemplate string, urlVariables ...string) (resp *http.Response, err error) {
	now := time.Now()
	response, error := hc.client.Get(expandURL(urlTemplate, urlVariables))
	hc.Instrument(response, urlTemplate, now)
	return response, error
}

// Post is a metric instrumentation wrapper for Client.Post with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Post method documentation for details.
func (hc *InstrumentedHttpClient) Post(urlTemplate string, contentType string, body io.Reader, urlVariables ...string) (resp *http.Response, err error) {
	now := time.Now()
	response, error := hc.client.Post(expandURL(urlTemplate, urlVariables), contentType, body)
	hc.Instrument(response, urlTemplate, now)
	return response, error
}

// Do is a metric instrumentation wrapper for Client.Do with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Do method documentation for details.
func (hc *InstrumentedHttpClient) Do(req *HttpRequestTemplate) (*http.Response, error) {
	now := time.Now()
	response, error := hc.client.Do(req.Request)
	hc.Instrument(response, req.UrlTemplate, now)
	return response, error
}

// PostForm is a metric instrumentation wrapper for Client.PostForm with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.PostForm method documentation for details.
func (hc *InstrumentedHttpClient) PostForm(urlTemplate string, data url.Values, urlVariables ...string) (resp *http.Response, err error) {
	now := time.Now()
	response, error := hc.client.PostForm(expandURL(urlTemplate, urlVariables), data)
	hc.Instrument(response, urlTemplate, now)
	return response, error
}

// Head is a metric instrumentation wrapper for Client.Head with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Head method documentation for details.
func (hc *InstrumentedHttpClient) Head(urlTemplate string, urlVariables ...string) (resp *http.Response, err error) {
	now := time.Now()
	response, error := hc.client.Head(expandURL(urlTemplate, urlVariables))
	hc.Instrument(response, urlTemplate, now)
	return response, error
}

// Instrument instruments response. Usually this is not needed by the library consumers, just use actual HTTP client operations and instrumentation is happening automatically.
func (hc *InstrumentedHttpClient) Instrument(response *http.Response, urlTemplate string, start time.Time) {
	if response != nil {
		url, err := url.Parse(urlTemplate)
		if err != nil {
			panic(err)
		}
		hc.instrumentDuration(response, url, start)
		hc.instrumentResponseSize(response, url)
		hc.instrumentRequestSize(response, url)
	}
}

func expandURL(urlTemplate string, urlVariables []string) string {
	expandedURL := urlTemplate
	if len(urlVariables) > 0 {
		if len(urlVariables) == strings.Count(urlTemplate, "{") && len(urlVariables) == strings.Count(urlTemplate, "}") {
			for _, v := range urlVariables {
				runes := []rune(expandedURL)
				start := strings.Index(expandedURL, "{")
				end := strings.Index(expandedURL, "}") + 1
				expandedURL = string(runes[0:start]) + url.PathEscape(v) + string(runes[end:])
			}
		} else {
			panic("Count mismatch between given URL variable(s) and variable {placeholder}(s)!")
		}
	}
	return expandedURL
}

func (hc *InstrumentedHttpClient) instrumentDuration(response *http.Response, urlTemplate *url.URL, start time.Time) {
	clientDuration.WithLabelValues(strconv.Itoa(response.StatusCode), response.Request.Method, getURIApplyingRules(urlTemplate, hc.rules), response.Request.URL.Hostname()).Observe(
		time.Since(start).Seconds())
}

func (hc *InstrumentedHttpClient) instrumentResponseSize(response *http.Response, urlTemplate *url.URL) {
	length := response.ContentLength
	if length > -1 {
		clientRespSize.WithLabelValues(strconv.Itoa(response.StatusCode), response.Request.Method, getURIApplyingRules(urlTemplate, hc.rules), response.Request.URL.Hostname()).Observe(
			float64(length))
	}
}

func (hc *InstrumentedHttpClient) instrumentRequestSize(response *http.Response, urlTemplate *url.URL) {
	clientRequestSize.WithLabelValues(strconv.Itoa(response.StatusCode), response.Request.Method, getURIApplyingRules(urlTemplate, hc.rules), response.Request.URL.Hostname()).Observe(
		float64(computeApproximateRequestSize(response.Request)))
}
