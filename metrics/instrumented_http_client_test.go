package metrics_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/metrics"
)

type testCase struct {
	str  string
	name string
}

var (
	metricHTTPClientRequestsDurationName = "http_client_requests_seconds"
	metricHTTPClientResponsesSizeName    = "http_client_responses_size_bytes"
	metricHTTPClientRequestsSizeName     = "http_client_requests_size_bytes"
	clientRequestDurationSum             = metricHTTPClientRequestsDurationName + "_sum"
	clientRequestDurationCount           = metricHTTPClientRequestsDurationName + "_count"
	clientRequestSizeSum                 = metricHTTPClientRequestsSizeName + "_sum"
	clientRequestSizeCount               = metricHTTPClientRequestsSizeName + "_count"
	clientResponseSizeSum                = metricHTTPClientResponsesSizeName + "_sum"
	clientResponseSizeCount              = metricHTTPClientResponsesSizeName + "_count"
	testEndpoint                         = "/client/test"
	testOpsEndpoint                      = "/client/test/ops"
	testFormEndpoint                     = "/client/test/form"
	testDoEndpoint                       = "/client/test/do"
	testDoEndpointTemplate               = "/client/{test}/{op}"
	testDoEndpointTemplateSet            = "/{foobaarii}"
	testDoEndpointTemplate2              = "/client/{test}/{op2}"
	testURLVariableEndpointTemplate      = "/{fooVar}/{barVar}/{id}"
	testURLVariableEndpointExpanded      = "/foo/bar/12345"
	testURLVariableEndpointTemplate2     = "/{fooVar}/{barVar}/{id2}"
	testURLVariableEndpointExpanded2     = "/foo/bar/123456"
	testURLVariableEndpointTemplate3     = "/{fooVar}/{barVar}/{id3}"
	testURLVariableEndpointExpanded3     = "/foo/bar/1234567"
	testEndPointNaN                      = "/client/notfound"
	testFormFieldName                    = "testField"
	testFormFieldValue                   = "foobar"
	testCookieName                       = "testCookie"
	testCookieValue                      = "1234"
	doMethod                             = "DELETE"
	testResponse                         = "OK"
	targetHost                           = "127.0.0.1"
)

// Simple example how to use default HTTP client with instrumentation capabilities.
func ExampleNewInstrumentedDefaultHttpClient() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Execute request by using some of the standard HTTP Client's wrapper methods.
	// Instrumentation will expose metrics for request/response time and sizes
	resp, _ := instrumentedHTTPClient.Get("http://foo.bar.com/foo")

	fmt.Println(resp.Status)
}

// Simple example how to use given HTTP client with instrumentation capabilities.
func ExampleNewInstrumentedHttpClient() {
	// Create standard HTTP Client
	client := &http.Client{Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}, Timeout: time.Second * 10}

	// Instantiate a new instrumented HTTP Client by using a given HTTP client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(client)

	// Execute request by using some of the standard HTTP Client's wrapper methods.
	// Instrumentation will expose metrics for request/response time and sizes
	resp, _ := instrumentedHTTPClient.Get("http://foo.bar.com/foo")

	fmt.Println(resp.Status)
}

// Example how to create HttpRequestTemplate for InstrumentedHttpClient.Do method argument.
func ExampleNewHttpRequestTemplate() {
	// Instantiate a new instrumented HTTP client
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Instantiate new request having URL template support
	request, err := metrics.NewHttpRequestTemplate("DELETE", "http://foo.bar.com/api/{id}", nil, "123456")
	if err != nil {
		panic(err)
	}

	// Use request to execute Do method
	resp, _ := instrumentedHTTPClient.Do(request)

	fmt.Println(resp.Status)
}

// NewHttpRequestTemplateFromRequest.
func ExampleNewHttpRequestTemplateFromRequest() {
	// Instantiate a new instrumented HTTP client
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Instantiate new request having URL template support
	requ, err := http.NewRequest("DELETE", "http://foo.bar.com/api/{id}", nil)
	if err != nil {
		panic(err)
	}
	request, err := metrics.NewHttpRequestTemplateFromRequest(requ, "123456")
	if err != nil {
		panic(err)
	}

	// Use request to execute Do method
	resp, _ := instrumentedHTTPClient.Do(request)

	fmt.Println(resp.Status)
}

// Example how to execute GET request.
func ExampleInstrumentedHttpClient_Get() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Execute GET
	resp, _ := instrumentedHTTPClient.Get("http://foo.bar.com/api/info")

	fmt.Println(resp.Status)
}

// Example how to execute GET request using URL template variables.
func ExampleInstrumentedHttpClient_Get_urlVariables() {
	// Instantiate a new instrumented HTTP Client by using given http.Client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})

	// Execute GET by using URL template variables
	resp, _ := instrumentedHTTPClient.Get("http://foo.bar.com/api/{type}/{id}/details", "foo_type", "1234")

	fmt.Println(resp.Status)
}

// Example how to execute POST request.
func ExampleInstrumentedHttpClient_Post() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Execute POST
	resp, _ := instrumentedHTTPClient.Post("http://foo.bar.com/api/info", "text/plain", strings.NewReader("Something to post"))

	fmt.Println(resp.Status)
}

// Example how to execute POST request using URL template variables.
func ExampleInstrumentedHttpClient_Post_urlVariables() {
	// Instantiate a new instrumented HTTP Client by using given http.Client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})

	// Execute POST by using URL template variables
	resp, _ := instrumentedHTTPClient.Post("http://foo.bar.com/api/{type}/add", "text/plain", strings.NewReader("Something to post"), "foo_type")

	fmt.Println(resp.Status)
}

// Example how to execute POST FORM request.
func ExampleInstrumentedHttpClient_PostForm() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Execute POST FORM
	resp, _ := instrumentedHTTPClient.PostForm("http://foo.bar.com/form-action", url.Values{"form-field": {"value"}})

	fmt.Println(resp.Status)
}

// Example how to execute POST FORM request using URL template variables.
func ExampleInstrumentedHttpClient_PostForm_urlVariables() {
	// Instantiate a new instrumented HTTP Client by using given http.Client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})

	// Execute POST FORM by using URL template variables
	resp, _ := instrumentedHTTPClient.PostForm("http://foo.bar.com/{operation}", url.Values{"form-field": {"value"}}, "form-action")

	fmt.Println(resp.Status)
}

// Example how to execute HEAD request.
func ExampleInstrumentedHttpClient_Head() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Execute HEAD
	resp, _ := instrumentedHTTPClient.Head("http://foo.bar.com/api/info")

	fmt.Println(resp.Status)
}

// Example how to execute HEAD request using URL template variables.
func ExampleInstrumentedHttpClient_Head_urlVariables() {
	// Instantiate a new instrumented HTTP Client by using given http.Client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})

	// Execute HEAD by using URL template variables
	resp, _ := instrumentedHTTPClient.Head("http://foo.bar.com/api/{type}/{id}/details", "foo_type", "1234")

	fmt.Println(resp.Status)
}

// Example how to execute DO request.
func ExampleInstrumentedHttpClient_Do() {
	// Instantiate a new instrumented HTTP Client using http.DefaultClient
	instrumentedHTTPClient := metrics.NewInstrumentedDefaultHttpClient()

	// Instantiate new request (e.g. PUT)
	request, _ := metrics.NewHttpRequestTemplate("PUT", "http://foo.bar.com/api?id=123456", strings.NewReader("Something to put"))

	// Execute PUT request
	resp, _ := instrumentedHTTPClient.Do(request)

	fmt.Println(resp.Status)
}

// Example how to execute DO request using URL template variables.
func ExampleInstrumentedHttpClient_Do_urlVariables() {
	// Instantiate a new instrumented HTTP Client by using given http.Client
	instrumentedHTTPClient := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})

	// Instantiate new request (e.g. PUT) having URL template support
	request, _ := metrics.NewHttpRequestTemplate("PUT", "http://foo.bar.com/api/{id}", strings.NewReader("Something to put"), "123456")

	// Execute PUT by using URL template variables
	resp, _ := instrumentedHTTPClient.Do(request)

	fmt.Println(resp.Status)
}

func TestInstrumentedHttpClientMetrics(t *testing.T) {
	testCases := []testCase{
		promCountRow(clientRequestDurationCount, targetHost, "GET", 200, testEndpoint, 1),
		promRow(clientRequestDurationSum, targetHost, "GET", 200, testEndpoint),
		promCountRow(clientRequestSizeCount, targetHost, "GET", 200, testEndpoint, 1),
		promRow(clientRequestSizeSum, targetHost, "GET", 200, testEndpoint),
		promCountRow(clientResponseSizeCount, targetHost, "GET", 200, testEndpoint, 1),
		promCountRow(clientResponseSizeSum, targetHost, "GET", 200, testEndpoint, len(testResponse)),
		promCountRow(clientRequestDurationCount, targetHost, "GET", 404, testEndPointNaN, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testEndpoint, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedHttpClient(&http.Client{Timeout: time.Second * 10})
	_, err := client.Get(ts.URL + testEndpoint)
	require.NoError(t, err)
	_, err = client.Get(ts.URL + testEndPointNaN)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

func TestInstrumentedHttpClientOperations(t *testing.T) {
	testCases := []testCase{
		promCountRow(clientRequestDurationCount, targetHost, "GET", 200, testOpsEndpoint, 1),
		promCountRow(clientRequestDurationCount, targetHost, "POST", 200, testOpsEndpoint, 1),
		promCountRow(clientRequestDurationCount, targetHost, "HEAD", 200, testOpsEndpoint, 1),
		promCountRow(clientRequestDurationCount, targetHost, "POST", 200, testFormEndpoint, 1),
		promCountRow(clientRequestDurationCount, targetHost, doMethod, 200, testDoEndpoint, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testOpsEndpoint, serveTestResponse)
	mux.HandleFunc(testFormEndpoint, func(w http.ResponseWriter, r *http.Request) { serveFormEndpoint(t, w, r) })
	mux.HandleFunc(testDoEndpoint, func(w http.ResponseWriter, r *http.Request) { serveDoEndpoint(t, w, r) })
	mux.HandleFunc(testURLVariableEndpointExpanded, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedDefaultHttpClient()
	_, err := client.Get(ts.URL + testOpsEndpoint)
	require.NoError(t, err)
	_, err = client.Head(ts.URL + testOpsEndpoint)
	require.NoError(t, err)
	_, err = client.Post(ts.URL+testOpsEndpoint, "text/plain", strings.NewReader(""))
	require.NoError(t, err)
	_, err = client.PostForm(ts.URL+testFormEndpoint, url.Values{testFormFieldName: {testFormFieldValue}})
	require.NoError(t, err)
	request, err := metrics.NewHttpRequestTemplate(doMethod, ts.URL+testDoEndpoint, strings.NewReader(""))
	require.NoError(t, err)
	request.Request.AddCookie(&http.Cookie{Name: testCookieName, Value: testCookieValue})
	_, err = client.Do(request)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

func TestInstrumentHttpClientTemplate(t *testing.T) {
	testCases := []testCase{
		promCountRow(clientRequestDurationCount, targetHost, "GET", 200, testURLVariableEndpointTemplate, 1),
		promCountRow(clientRequestDurationCount, targetHost, doMethod, 200, testDoEndpointTemplate, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testDoEndpoint, func(w http.ResponseWriter, r *http.Request) { serveDoEndpoint(t, w, r) })
	mux.HandleFunc(testURLVariableEndpointExpanded, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedDefaultHttpClient()
	urlPaths := strings.Split(testURLVariableEndpointExpanded, "/")
	_, err := client.Get(ts.URL+testURLVariableEndpointTemplate, urlPaths[1], urlPaths[2], urlPaths[3])
	require.NoError(t, err)
	request, err := metrics.NewHttpRequestTemplate(doMethod, ts.URL+testDoEndpointTemplate, strings.NewReader(""), "test", "do")
	require.NoError(t, err)
	request.Request.AddCookie(&http.Cookie{Name: testCookieName, Value: testCookieValue})
	_, err = client.Do(request)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

func TestInstrumentHttpClientTemplateSet(t *testing.T) {
	testCases := []testCase{
		promRow(clientRequestDurationCount, targetHost, doMethod, 200, testDoEndpointTemplateSet),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testDoEndpoint, func(w http.ResponseWriter, r *http.Request) { serveDoEndpoint(t, w, r) })
	mux.HandleFunc(testURLVariableEndpointExpanded, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedDefaultHttpClient()
	urlPaths := strings.Split(testURLVariableEndpointExpanded, "/")
	_, err := client.Get(ts.URL+testURLVariableEndpointTemplate, urlPaths[1], urlPaths[2], urlPaths[3])
	require.NoError(t, err)
	request, err := metrics.NewHttpRequestTemplate(doMethod, ts.URL+testDoEndpointTemplate, strings.NewReader(""), "test", "do")
	require.NoError(t, err)
	request.UrlTemplate = testDoEndpointTemplateSet
	request.Request.AddCookie(&http.Cookie{Name: testCookieName, Value: testCookieValue})
	_, err = client.Do(request)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

func TestInstrumentHttpClientTemplateFromRequest(t *testing.T) {
	testCases := []testCase{
		promCountRow(clientRequestDurationCount, targetHost, "GET", 200, testURLVariableEndpointTemplate2, 1),
		promCountRow(clientRequestDurationCount, targetHost, doMethod, 200, testDoEndpointTemplate2, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testDoEndpoint, func(w http.ResponseWriter, r *http.Request) { serveDoEndpoint(t, w, r) })
	mux.HandleFunc(testURLVariableEndpointExpanded2, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedDefaultHttpClient()
	urlPaths := strings.Split(testURLVariableEndpointExpanded2, "/")
	_, err := client.Get(ts.URL+testURLVariableEndpointTemplate2, urlPaths[1], urlPaths[2], urlPaths[3])
	require.NoError(t, err)
	req, err := http.NewRequest(doMethod, ts.URL+testDoEndpointTemplate2, strings.NewReader(""))
	require.NoError(t, err)

	req.Header.Set("foo", "bar")
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: testCookieValue})
	request, err := metrics.NewHttpRequestTemplateFromRequest(req, "test", "do")
	require.NoError(t, err)
	assert.Equal(t, req.Header.Get("foo"), request.Request.Header.Get("foo"), "Verify also headers are set")
	_, err = client.Do(request)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

func TestInstrumentHttpClientTemplateFromRequestNoPathVars(t *testing.T) {
	testCases := []testCase{
		promCountRow(clientRequestDurationCount, targetHost, "GET", 200, testURLVariableEndpointTemplate3, 1),
		promCountRow(clientRequestDurationCount, targetHost, doMethod, 200, testURLVariableEndpointExpanded3, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(testDoEndpoint, func(w http.ResponseWriter, r *http.Request) { serveDoEndpoint(t, w, r) })
	mux.HandleFunc(testURLVariableEndpointExpanded3, serveTestResponse)
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := metrics.NewInstrumentedDefaultHttpClient()
	urlPaths := strings.Split(testURLVariableEndpointExpanded3, "/")
	_, err := client.Get(ts.URL+testURLVariableEndpointTemplate3, urlPaths[1], urlPaths[2], urlPaths[3])
	require.NoError(t, err)
	req, err := http.NewRequest(doMethod, ts.URL+testURLVariableEndpointExpanded3, strings.NewReader(""))
	require.NoError(t, err)

	req.Header.Set("foo", "bar")
	req.AddCookie(&http.Cookie{Name: testCookieName, Value: testCookieValue})
	request, err := metrics.NewHttpRequestTemplateFromRequest(req)
	require.NoError(t, err)
	assert.Equal(t, req.Header.Get("foo"), request.Request.Header.Get("foo"), "Verify also headers are set")
	_, err = client.Do(request)
	require.NoError(t, err)
	checkOutput(t, client, ts.URL, testCases)
}

//nolint:errcheck
func serveTestResponse(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(testResponse))
}

func serveDoEndpoint(t *testing.T, w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(testCookieName)
	require.NoError(t, err)

	assert.Contains(t, c.Value, testCookieValue, "Test cookie exists")
	_, err = w.Write([]byte(testResponse))
	require.NoError(t, err)
}

func serveFormEndpoint(t *testing.T, w http.ResponseWriter, r *http.Request) {
	assert.Contains(t, r.FormValue(testFormFieldName), testFormFieldValue, "Form value exists")
	serveTestResponse(w, r)
}

func checkOutput(t *testing.T, client *metrics.InstrumentedHttpClient, url string, cases []testCase) {
	response, err := client.Get(url + metrics.DefaultEndPoint)
	require.NoError(t, err)

	buf, err := ioutil.ReadAll(response.Body)
	require.NoError(t, err)

	body := string(buf)
	for _, testCase := range cases {
		assert.Contains(t, body, testCase.str, testCase.name)
	}
}

func promCountRow(metricName, clientName, method string, statusCode int, uri string, count int) testCase {
	return testCase{
		str:  fmt.Sprintf("%s{clientName=\"%s\",method=\"%s\",status=\"%d\",uri=\"%s\"} %d", metricName, clientName, method, statusCode, uri, count),
		name: fmt.Sprintf("For %s do %s expecting %d", metricName, method, statusCode),
	}
}

func promRow(metricName, clientName, method string, statusCode int, uri string) testCase {
	return testCase{
		str:  fmt.Sprintf("%s{clientName=\"%s\",method=\"%s\",status=\"%d\",uri=\"%s\"} ", metricName, clientName, method, statusCode, uri),
		name: fmt.Sprintf("For %s do %s expecting %d", metricName, method, statusCode),
	}
}
