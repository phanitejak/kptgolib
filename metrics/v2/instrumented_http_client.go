// Package metrics v2 provides unified functions to expose application level
// metrics from the NEO services.
package metrics

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"gopkg/metrics"
)

type contextKey string

const (
	contextKeyURLTemplate = contextKey("com_nokia_neo_metrics_url_template")
)

// InstrumentedHTTPClient represents standard http.Client with metrics instrumentation capabilities.
type InstrumentedHTTPClient struct {
	hClient *http.Client
	iClient *metrics.InstrumentedHttpClient
}

// InstrumentedTransport is a http.RoundTripper with metrics instrumentation capabilities.
type InstrumentedTransport struct {
	rt      http.RoundTripper
	iClient *metrics.InstrumentedHttpClient
}

// NewInstrumentedHTTPClient returns given http client with instrumentation capabilities based on given rules for URI templating.
func NewInstrumentedHTTPClient(httpClient *http.Client, rules ...metrics.InstrumentRule) *InstrumentedHTTPClient {
	c := metrics.NewInstrumentedHttpClient(httpClient)
	c.SetRules(rules...)
	return &InstrumentedHTTPClient{httpClient, c}
}

// NewInstrumentedDefaultHTTPClient returns default http client with instrumentation capabilities based on given rules for URI templating.
func NewInstrumentedDefaultHTTPClient(rules ...metrics.InstrumentRule) *InstrumentedHTTPClient {
	return NewInstrumentedHTTPClient(http.DefaultClient, rules...)
}

// NewInstrumentedTransport returns given RoundTripper with instrumentation capabilities.
func NewInstrumentedTransport(rt http.RoundTripper) http.RoundTripper {
	return NewInstrumentedTransportWithRules(rt)
}

// NewInstrumentedTransportForKubeAPI returns given RoundTripper with instrumentation capabilities including URI templating for Kube API.
func NewInstrumentedTransportForKubeAPI(rt http.RoundTripper) http.RoundTripper {
	return NewInstrumentedTransportWithRules(rt, *kubeAPIRules...)
}

// NewInstrumentedTransportWithRules returns given RoundTripper with instrumentation capabilities based on given rules for URI templating.
func NewInstrumentedTransportWithRules(rt http.RoundTripper, rules ...metrics.InstrumentRule) http.RoundTripper {
	c := metrics.NewInstrumentedDefaultHttpClient()
	c.SetRules(rules...)
	return &InstrumentedTransport{rt, c}
}

// NewInstrumentedDefaultTransport returns given RoundTripper with instrumentation capabilities based on given rules for URI templating.
func NewInstrumentedDefaultTransport(rules ...metrics.InstrumentRule) http.RoundTripper {
	c := metrics.NewInstrumentedDefaultHttpClient()
	c.SetRules(rules...)
	return &InstrumentedTransport{http.DefaultTransport, c}
}

// NewHTTPRequest returns a new HTTPRequest given a method, URL, optional body and urlVariables.
// This can be then used as an argument for InstrumentedHTTPClient.Do method.
func NewHTTPRequest(method, urlTemplate string, body io.Reader, urlVariables ...string) (*http.Request, error) {
	tmpl, err := metrics.NewHttpRequestTemplate(method, urlTemplate, body, urlVariables...)
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(tmpl.Request.Context(), contextKeyURLTemplate, urlTemplate)
	return tmpl.Request.WithContext(ctx), nil
}

// NewHTTPRequestFromRequest returns a new HTTPRequest with given request and urlVariables.
// This can be then used as an argument for InstrumentedHTTPClient.Do method.
func NewHTTPRequestFromRequest(request *http.Request, urlVariables ...string) (*http.Request, error) {
	rawURL, err := url.QueryUnescape(request.URL.String())
	if err != nil {
		return nil, err
	}
	tmpl, err := metrics.NewHttpRequestTemplateFromRequest(request, urlVariables...)
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(tmpl.Request.Context(), contextKeyURLTemplate, rawURL)
	return tmpl.Request.WithContext(ctx), nil
}

// Get is a metric instrumentation wrapper for Client.Get with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Get method documentation for details.
func (hc2 *InstrumentedHTTPClient) Get(urlTemplate string, urlVariables ...string) (resp *http.Response, err error) {
	response, error := hc2.iClient.Get(urlTemplate, urlVariables...)
	return response, error
}

// Post is a metric instrumentation wrapper for Client.Post with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Post method documentation for details.
func (hc2 *InstrumentedHTTPClient) Post(urlTemplate string, contentType string, body io.Reader, urlVariables ...string) (resp *http.Response, err error) {
	response, error := hc2.iClient.Post(urlTemplate, contentType, body, urlVariables...)
	return response, error
}

// Do is a metric instrumentation wrapper for Client.Do with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Do method documentation for details.
func (hc2 *InstrumentedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	now := time.Now()
	response, error := hc2.hClient.Do(req)
	keyVal := req.Context().Value(contextKeyURLTemplate)
	var template string
	if keyVal == nil {
		template = req.URL.Path
	} else {
		template = keyVal.(string)
	}
	hc2.iClient.Instrument(response, template, now)
	return response, error
}

// PostForm is a metric instrumentation wrapper for Client.PostForm with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.PostForm method documentation for details.
func (hc2 *InstrumentedHTTPClient) PostForm(urlTemplate string, data url.Values, urlVariables ...string) (resp *http.Response, err error) {
	response, error := hc2.iClient.PostForm(urlTemplate, data, urlVariables...)
	return response, error
}

// Head is a metric instrumentation wrapper for Client.Head with URL template support.
// Instrumentation exposes metrics for request/response time and sizes.
// See the Client.Head method documentation for details.
func (hc2 *InstrumentedHTTPClient) Head(urlTemplate string, urlVariables ...string) (resp *http.Response, err error) {
	response, error := hc2.iClient.Head(urlTemplate, urlVariables...)
	return response, error
}

// RoundTrip implements http.RoundTripper. It forwards the request to the
// next RoundTripper and instruments request.
func (it *InstrumentedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	now := time.Now()
	resp, err := it.rt.RoundTrip(req)
	keyVal := req.Context().Value(contextKeyURLTemplate)
	var template string
	if keyVal == nil {
		template = req.URL.Path
	} else {
		template = keyVal.(string)
	}
	it.iClient.Instrument(resp, template, now)
	return resp, err
}
