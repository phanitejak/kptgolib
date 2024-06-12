// Package tracingtest is about setting up environment for tracing
// nolint
package tracingtest

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	tracesdk "go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/require"
	"gopkg/tracing"
)

// SetUp sets tracing related env vars and inits global tracer.
// CleanUp function closes global tracer.
func SetUp(t testing.TB) (cleanUp func()) {
	setEnv(t)
	closer, err := tracing.InitGlobalTracer()
	require.NoError(t, err)

	return func() {
		_ = closer.Close()
	}
}

// SetUpNoopTracerProvider sets up a tracerProvider that returns Tracer and Spans that perform no operations.
// The spans created will not be valid.
func SetUpNoopTracerProvider() {
	otel.SetTracerProvider(trace.NewNoopTracerProvider())
}

func SetUpWithMockProcessor(t testing.TB) (cleanUp func(), processor *MockProcessor) {
	setEnv(t)
	processor = NewMockProcessor()
	closer, err := tracing.InitGlobalTracer(tracing.WithProcessor(processor))
	require.NoError(t, err)

	return func() {
		_ = closer.Close()
	}, processor
}

func SetUpWithMockExporter(t testing.TB) (cleanUp func(), exporter MockExporter) {
	setEnv(t)
	exporter = NewMockExporter()
	pros := tracesdk.NewSimpleSpanProcessor(&exporter)
	closer, err := tracing.InitGlobalTracer(tracing.WithProcessor(pros))
	require.NoError(t, err)

	return func() {
		_ = closer.Close()
	}, exporter
}

func setEnv(t testing.TB) {
	t.Setenv("JAEGER_ENDPOINT", "http://127.0.0.1:14268/api/traces")
	t.Setenv("JAEGER_SERVICE_NAME", "testService")
	t.Setenv("JAEGER_SAMPLER_TYPE", "probabilistic")
	t.Setenv("JAEGER_SAMPLER_PARAM", "1")
	t.Setenv("STANDALONE", "true")
}

type MockProcessor struct {
	spanstorage map[string][]tracesdk.ReadOnlySpan
	mu          *sync.RWMutex
}

func NewMockProcessor() *MockProcessor {
	return &MockProcessor{
		spanstorage: map[string][]tracesdk.ReadOnlySpan{},
		mu:          &sync.RWMutex{},
	}
}

func (m *MockProcessor) OnStart(context.Context, tracesdk.ReadWriteSpan) {
}

// OnEnd immediately exports a ReadOnlySpan.
func (m *MockProcessor) OnEnd(s tracesdk.ReadOnlySpan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.spanstorage[s.Name()]
	if ok {
		m.spanstorage[s.Name()] = append(m.spanstorage[s.Name()], s)
		return
	}
	m.spanstorage[s.Name()] = []tracesdk.ReadOnlySpan{s}
}

// Shutdown shuts down the exporter this SimpleSpanProcessor exports to.
func (m *MockProcessor) Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush does nothing as there is no data to flush.
func (m *MockProcessor) ForceFlush(context.Context) error {
	return nil
}

func (m *MockProcessor) GetAttributes(spanName string) (map[string][]attribute.KeyValue, error) {
	retMap := map[string][]attribute.KeyValue{}
	spans, ok := m.getSpansOK(spanName)
	if ok {
		for _, v := range spans {
			retMap[v.SpanContext().SpanID().String()] = v.Attributes()
		}
		return retMap, nil
	}
	return nil, fmt.Errorf("no span with name %s found", spanName)
}

func (m *MockProcessor) GetSpanAmount(spanName string) int {
	spans, ok := m.getSpansOK(spanName)
	if !ok {
		return 0
	}
	return len(spans)
}

func (m *MockProcessor) CheckSpansNotNil(t testing.TB, spanName string) error {
	spans, ok := m.getSpansOK(spanName)
	if ok {
		for _, span := range spans {
			assert.NotNil(t, span)
		}
		return nil
	}
	return fmt.Errorf("no span of name %s found", spanName)
}

func (m *MockProcessor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.spanstorage = map[string][]tracesdk.ReadOnlySpan{}
}

func (m *MockProcessor) getSpans(spanName string) []tracesdk.ReadOnlySpan {
	spans, _ := m.getSpansOK(spanName)
	return spans
}

func (m *MockProcessor) getSpansOK(spanName string) ([]tracesdk.ReadOnlySpan, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	spans, ok := m.spanstorage[spanName]
	return spans, ok
}

func (m *MockProcessor) FindAttribute(spanName, key string) (string, bool) {
	spans := m.getSpans(spanName)
	for _, span := range spans {
		for _, attr := range span.Attributes() {
			if string(attr.Key) == key {
				return attr.Value.AsString(), true
			}
		}
	}
	return "", false
}

func (m *MockProcessor) FindEventAttributeValue(spanName, key string) (string, bool) {
	spans := m.getSpans(spanName)
	for _, span := range spans {
		for _, event := range span.Events() {
			for _, attr := range event.Attributes {
				if string(attr.Key) == key {
					r := attr.Value.Emit()
					return r, true
				}
			}
		}
	}
	return "", false
}

func (m *MockProcessor) FindEventAttribute(spanName, key string) (attribute.KeyValue, bool) {
	spans := m.getSpans(spanName)
	for _, span := range spans {
		for _, event := range span.Events() {
			for _, attr := range event.Attributes {
				if string(attr.Key) == key {
					return attr, true
				}
			}
		}
	}
	return attribute.KeyValue{}, false
}

func (m *MockProcessor) SpanNameExist(spanName string) bool {
	_, ok := m.getSpansOK(spanName)
	return ok
}

func KeyValueToMap(kvList []attribute.KeyValue) map[attribute.Key]attribute.Value {
	retlist := map[attribute.Key]attribute.Value{}
	for _, kv := range kvList {
		retlist[kv.Key] = kv.Value
	}
	return retlist
}

type MockExporter struct {
	spanstorage map[string]tracesdk.ReadOnlySpan
	mu          *sync.RWMutex
}

func NewMockExporter() MockExporter {
	return MockExporter{
		spanstorage: map[string]tracesdk.ReadOnlySpan{},
		mu:          &sync.RWMutex{},
	}
}

func (e *MockExporter) ExportSpans(ctx context.Context, spans []tracesdk.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, span := range spans {
		e.spanstorage[span.Name()] = span
	}
	return nil
}

func (e *MockExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *MockExporter) GetAttributes(spanName string) ([]attribute.KeyValue, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.spanstorage[spanName]
	if ok {
		return val.Attributes(), nil
	}
	return nil, fmt.Errorf("no span with name %s found", spanName)
}
