package tracing_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/phanitejak/kptgolib/tracing"
	"github.com/phanitejak/kptgolib/tracing/tracingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const deprecatedParentID = 0

func TestHttpRequestAreNotInjectedWhenTracingIsDisabled(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	request, err := http.NewRequest(http.MethodGet, "http://some-url.com", nil)
	require.NoError(t, err)

	r := tracing.RequestWithContext(request, context.Background())

	assert.Equal(t, "", r.Header.Get("Uber-Trace-Id"))
}

func TestHttpRequestHasTraceHeadersInjected(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	span, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")
	spanContext := span.SpanContext()
	require.True(t, spanContext.IsValid())

	request, err := http.NewRequest(http.MethodGet, "http://some-url.com", nil)
	require.NoError(t, err)

	r := tracing.RequestWithContext(request, ctx)

	assert.Equal(t, uberTraceFromSpan(span), r.Header.Get("Uber-Trace-Id"))
	span.End()
}

func uberTraceFromSpan(span tracing.Span) string {
	spanCtx := span.SpanContext()
	sampled := 0
	if spanCtx.IsSampled() {
		sampled = 1
	}
	return fmt.Sprintf("%s:%s:%d:%d", spanCtx.TraceID().String(), spanCtx.SpanID().String(), deprecatedParentID, sampled)
}

func TestContextCreationOnIncomingRequests(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	router := mux.NewRouter()

	ctx := context.Background()

	controllerFunc := func(writer http.ResponseWriter, request *http.Request) {
		ctx = request.Context()
	}
	router.Handle("/", tracing.Wrap(http.HandlerFunc(controllerFunc)))

	rr := httptest.NewRecorder()
	request, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	router.ServeHTTP(rr, request)

	assert.Equal(t, http.StatusOK, rr.Code)
	span := tracing.SpanFromContext(ctx)
	assert.NotNil(t, span)
	assert.True(t, span.SpanContext().IsValid())
}

func TestWrappedServerRequest(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	router := mux.NewRouter()

	var serverSpan tracing.Span

	controllerFunc := func(writer http.ResponseWriter, request *http.Request) {
		serverSpan, _ = tracing.StartSpanFromContext(request.Context(), "test server")
		defer serverSpan.Finish()
	}
	router.Handle("/", tracing.Wrap(http.HandlerFunc(controllerFunc)))

	rr := httptest.NewRecorder()
	emptyRequest, err := http.NewRequest(http.MethodGet, "/", nil)

	span, spanContext := tracing.StartSpanFromContext(context.Background(), "test-op")
	defer span.Finish()

	require.NoError(t, err)
	request := tracing.RequestWithContext(emptyRequest, spanContext)
	router.ServeHTTP(rr, request)

	assert.Equal(t, span.SpanContext().SpanID().String(), strings.Split(request.Header["Uber-Trace-Id"][0], ":")[1])
	assert.Equal(t, span.SpanContext().TraceID().String(), serverSpan.SpanContext().TraceID().String())
}

func TestHttpHandler(t *testing.T) {
	closer, mp := tracingtest.SetUpWithMockProcessor(t)
	defer closer()
	var serverSpan tracing.Span
	handleFunc := func(w http.ResponseWriter, r *http.Request) {
		serverSpan, _ = tracing.StartSpanFromContext(r.Context(), "test-server-serve")
		defer serverSpan.Finish()
	}

	url := "/api/v1/test"
	method := http.MethodPost
	body := bytes.NewBuffer([]byte(`"5"`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, url, body)
	h := tracing.Wrap(http.HandlerFunc(handleFunc))
	h.ServeHTTP(w, r)

	assert.Equal(t, 1, mp.GetSpanAmount("test-server-serve"))
	assert.Equal(t, 1, mp.GetSpanAmount("POST /api/v1/test"))
}
