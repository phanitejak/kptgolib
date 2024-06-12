package metrics_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg/metrics"
	metricsv2 "gopkg/metrics/v2"
)

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
	targetHost                           = "127.0.0.1"
)

type testEndpointDef struct {
	name       string
	handleFunc func(w http.ResponseWriter, r *http.Request)
}

func TestInstrumentedDefaultHTTPClient_WithoutTemplating(t *testing.T) {
	testEndpointName := "/v2/TestInstrumentedDefaultHTTPClient_WithoutTemplating"
	testEndpointNameNotFound := "/v2/not_found"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedDefaultHTTPClient()

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusOK), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointName)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointNameNotFound)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointNameNotFound, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointName, "text/plain", strings.NewReader("Something to post"))
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointNameNotFound, "text/plain", strings.NewReader("Something to post"))
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointNameNotFound, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusOK), func(t *testing.T) {
		resp, err := client.Head(ts.URL + testEndpointName)
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Head(ts.URL + testEndpointNameNotFound)
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointNameNotFound, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithoutTemplatingPostForm(t *testing.T) {
	testEndpointName := "/v2/TestInstrumentedHTTPClient_WithoutTemplatingPostForm"
	testEndpointNameNotFound := "/v2/post_form_not_found"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		resp, err := client.PostForm(ts.URL+testEndpointName, url.Values{"form-field": {"value"}})
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.PostForm(ts.URL+testEndpointNameNotFound, url.Values{"form-field": {"value"}})
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointNameNotFound, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithTemplating(t *testing.T) {
	testEndpointTemplate := "/v2/TestInstrumentedTTPClient_WithTemplating/{id1}/{id2}"
	testEndpointName := "/v2/TestInstrumentedTTPClient_WithTemplating/1/2"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusOK), func(t *testing.T) {
		resp, err := client.Get(ts.URL+testEndpointTemplate, "1", "2")
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Get(ts.URL+testEndpointTemplate, "not", "found")
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointTemplate, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointTemplate, "text/plain", strings.NewReader("Something to post"), "1", "2")
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointTemplate, "text/plain", strings.NewReader("Something to post"), "not", "found")
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusOK), func(t *testing.T) {
		resp, err := client.Head(ts.URL+testEndpointTemplate, "1", "2")
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Head(ts.URL+testEndpointTemplate, "not", "found")
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointTemplate, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithoutTemplatingDoFromNewHTTPRequest(t *testing.T) {
	testEndpointName := "/v2/TestInstrumentedHTTPClient_WithoutTemplatingDoFromNewHTTPRequest"
	testEndpointNameNotFound := "/v2/do_from_new_http_request_not_found"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d_%s", http.MethodPost, http.StatusOK, "NewHTTPRequest"), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointName, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d_%s", http.MethodPost, http.StatusNotFound, "NewHTTPRequest"), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointNameNotFound, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointNameNotFound, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequest(t *testing.T) {
	testEndpointTemplate := "/v2/TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequest/{id1}/{id2}"
	testEndpointName := "/v2/TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequest/1/2"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"), "1", "2")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"), "not", "found")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithoutTemplatingDoFromNewHTTPRequestFromRequest(t *testing.T) {
	testEndpointName := "/v2/TestInstrumentedHTTPClient_WithoutTemplatingDoFromNewHTTPRequestFromRequest"
	testEndpointNameNotFound := "/v2/do_from_new_http_request_from_request_not_found"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+testEndpointName, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		req, err = metricsv2.NewHTTPRequestFromRequest(req)
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+testEndpointNameNotFound, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		req, err = metricsv2.NewHTTPRequestFromRequest(req)
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointNameNotFound, http.StatusNotFound)
	})
}

func TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequestFromRequest(t *testing.T) {
	testEndpointTemplate := "/v2/TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequestFromRequest/{id1}/{id2}"
	testEndpointName := "/v2/TestInstrumentedHTTPClient_WithTemplatingDoFromNewHTTPRequestFromRequest/1/2"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := metricsv2.NewInstrumentedHTTPClient(&http.Client{Timeout: time.Second * 10})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		req, err = metricsv2.NewHTTPRequestFromRequest(req, "1", "2")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"))
		assert.NotNil(t, req)
		assert.Nil(t, err)
		req, err = metricsv2.NewHTTPRequestFromRequest(req, "not", "found")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusNotFound)
	})
}

func TestInstrumentedDefaultTransport_WithoutTemplating(t *testing.T) {
	testEndpointName := "/v2/TestInstrumentedDefaultTransport_WithoutTemplating/1/2"
	testEndpointNameNotFound := "/v2/not_found_2"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := http.Client{Transport: metricsv2.NewInstrumentedDefaultTransport()}
	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusOK), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointName)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointNameNotFound)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointNameNotFound, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointName, "text/plain", strings.NewReader("Something to post"))
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Post(ts.URL+testEndpointNameNotFound, "text/plain", strings.NewReader("Something to post"))
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointNameNotFound, http.StatusNotFound)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusOK), func(t *testing.T) {
		resp, err := client.Head(ts.URL + testEndpointName)
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointName, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodHead, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Head(ts.URL + testEndpointNameNotFound)
		verifyResponse(t, ts.URL, resp, err, http.MethodHead, testEndpointNameNotFound, http.StatusNotFound)
	})
}

func TestInstrumentedTransport_WithTemplatingDoFromNewHTTPRequest(t *testing.T) {
	testEndpointTemplate := "/v2/TestInstrumentedTransport_WithTemplatingDoFromNewHTTPRequest/{id1}/{id2}"
	testEndpointName := "/v2/TestInstrumentedTransport_WithTemplatingDoFromNewHTTPRequest/1/2"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := http.Client{Transport: metricsv2.NewInstrumentedTransport(&http.Transport{})}
	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusOK), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"), "1", "2")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodPost, http.StatusNotFound), func(t *testing.T) {
		req, err := metricsv2.NewHTTPRequest(http.MethodPost, ts.URL+testEndpointTemplate, strings.NewReader("Something to post"), "not", "found")
		assert.NotNil(t, req)
		assert.Nil(t, err)
		resp, err := client.Do(req)
		verifyResponse(t, ts.URL, resp, err, http.MethodPost, testEndpointTemplate, http.StatusNotFound)
	})
}

func TestInstrumentedTransport_WithTemplatingKubeAPIRules(t *testing.T) {
	testEndpointTemplate := "/api/v1/namespaces/{namespace}/pods/{name}/status"
	testEndpointName := "/api/v1/namespaces/neo/pods/foobar-pod-123/status"
	testEndpointNameNotFound := "/api/v1/namespaces/neo/pods/foobar-pod-not-found/status"
	ts := startTestServer(testEndpointDef{name: testEndpointName})
	defer ts.Close()
	client := http.Client{Transport: metricsv2.NewInstrumentedTransportForKubeAPI(&http.Transport{})}
	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusOK), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointName)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointTemplate, http.StatusOK)
	})

	t.Run(fmt.Sprintf("%s_%d", http.MethodGet, http.StatusNotFound), func(t *testing.T) {
		resp, err := client.Get(ts.URL + testEndpointNameNotFound)
		verifyResponse(t, ts.URL, resp, err, http.MethodGet, testEndpointTemplate, http.StatusNotFound)
	})
}

func verifyResponse(t *testing.T, serverURL string, resp *http.Response, err error, requiredMethod, requiredURI string, requiredStatus int) {
	assert.NotNil(t, resp)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	assert.Nil(t, err)
	assert.Equal(t, resp.StatusCode, requiredStatus)
	metricsResponse := strings.Split(getMetricResponse(t, serverURL+metrics.DefaultEndPoint), "\n")
	verifyClientMetrics(t, metricsResponse, requiredMethod, requiredURI, requiredStatus, resp.ContentLength > -1)
}

func verifyClientMetrics(t *testing.T, metricsResponse []string, requiredMethod, requiredURI string, requiredStatus int, checkResponseSize bool) {
	t.Run(clientRequestDurationSum, func(t *testing.T) {
		verifyClientMetric(t, metricsResponse, targetHost, clientRequestDurationSum, requiredMethod, requiredURI, ".*", requiredStatus)
	})
	t.Run(clientRequestDurationCount, func(t *testing.T) {
		verifyClientMetric(t, metricsResponse, targetHost, clientRequestDurationCount, requiredMethod, requiredURI, "1", requiredStatus)
	})
	t.Run(clientRequestSizeSum, func(t *testing.T) {
		verifyClientMetric(t, metricsResponse, targetHost, clientRequestSizeSum, requiredMethod, requiredURI, ".*", requiredStatus)
	})
	t.Run(clientRequestSizeCount, func(t *testing.T) {
		verifyClientMetric(t, metricsResponse, targetHost, clientRequestSizeCount, requiredMethod, requiredURI, "1", requiredStatus)
	})
	if checkResponseSize {
		t.Run(clientResponseSizeSum, func(t *testing.T) {
			verifyClientMetric(t, metricsResponse, targetHost, clientResponseSizeSum, requiredMethod, requiredURI, ".*", requiredStatus)
		})
		t.Run(clientResponseSizeCount, func(t *testing.T) {
			verifyClientMetric(t, metricsResponse, targetHost, clientResponseSizeCount, requiredMethod, requiredURI, "1", requiredStatus)
		})
	}
}

func verifyClientMetric(t *testing.T, metricsResponse []string, requiredClient, requiredMetric, requiredMethod, requiredURI, valueReqex string, requiredStatus int) {
	assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`%s{clientName="%s",method="%s",status="%d",uri="%s"} %s`, requiredMetric, requiredClient, requiredMethod, requiredStatus, requiredURI, valueReqex)), metricsResponse)
}

func startTestServer(endpoints ...testEndpointDef) *httptest.Server {
	mux := http.NewServeMux()
	for _, endpoint := range endpoints {
		if endpoint.handleFunc != nil {
			mux.HandleFunc(endpoint.name, endpoint.handleFunc)
		} else {
			mux.HandleFunc(endpoint.name, func(http.ResponseWriter, *http.Request) {})
		}
	}
	mux.Handle(metrics.DefaultEndPoint, metrics.GetMetricsHandler())
	return httptest.NewServer(mux)
}

func getMetricResponse(t *testing.T, url string) string {
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	metrics.GetMetricsHandler().(http.HandlerFunc)(w, req)
	resp := w.Result()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return string(buf)
}
