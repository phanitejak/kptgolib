// nolint
package tracing_test

import (
	"context"
	"fmt"
	"testing"

	"gopkg/tracing/tracingtest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/logging"
	"gopkg/tracing"
)

func TestGetTraceIDFromContext(t *testing.T) {
	logger := logging.NewLogger()
	closer, err := tracing.InitGlobalTracer(tracing.WithLogger(logger))
	assert.NoError(t, err)
	defer func() {
		err = closer.Close()
		require.NoError(t, err)
	}()
	span, spanContext := tracing.StartSpan("Test-operation", tracing.FollowsFrom(context.Background()))
	span.Finish()
	traceID, err := tracing.GetTraceIDFromContext(spanContext)
	assert.NoError(t, err)
	assert.Len(t, traceID, 32)

	traceID, err = tracing.GetTraceIDFromContext(nil) // nolint: staticcheck
	assert.Error(t, err)
	assert.Len(t, traceID, 0)

	traceID, err = tracing.GetTraceIDFromContext(context.TODO())
	assert.Error(t, err)
	assert.Len(t, traceID, 0)
}

func TestSetOperationName(t *testing.T) {
	closer, processor := tracingtest.SetUpWithMockProcessor(t)
	defer closer()
	originalName := "origName"
	newName := "newName"

	span, _ := tracing.StartSpan(originalName, tracing.FollowsFrom(context.Background()))
	span.SetOperationName(newName)
	span.Finish()

	assert.Falsef(t, processor.SpanNameExist(originalName), fmt.Sprintf("span name %s existed while it should not", originalName))
	assert.Truef(t, processor.SpanNameExist(newName), fmt.Sprintf("span name %s did not exit", newName))
}

var tests = []struct {
	name     string
	fields   []tracing.Field
	key      string
	expected string
}{
	{
		name: "log a string",
		fields: []tracing.Field{
			tracing.String("stringkey", "val"),
		},
		key:      "stringkey",
		expected: "val",
	},
	{
		name: "log a Bool",
		fields: []tracing.Field{
			tracing.Bool("boolkey", true),
		},
		key:      "boolkey",
		expected: "true",
	},
	{
		name: "log an Object",
		fields: []tracing.Field{
			tracing.Object("objectkey", "Just a string"),
		},
		key:      "objectkey",
		expected: "Just a string",
	},
	{
		name: "log an Int",
		fields: []tracing.Field{
			tracing.Int("intkey", 555),
		},
		key:      "intkey",
		expected: "555",
	},
	{
		name: "log an Error",
		fields: []tracing.Field{
			tracing.Error(fmt.Errorf("error message for test")),
		},
		key:      "error.object",
		expected: "error message for test",
	},
}

func TestLogFields(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traceCloser, traceReporter := tracingtest.SetUpWithMockProcessor(t)
			ctx := context.Background()
			spanName := "op"
			span, ctx := tracing.StartSpanFromContext(ctx, spanName)
			require.NotNil(t, span)
			require.NotNil(t, ctx)

			span.LogFields(tt.fields...)
			span.Finish()

			str, found := traceReporter.FindEventAttributeValue(spanName, tt.key)
			assert.True(t, found)
			assert.Equal(t, tt.expected, str)

			traceCloser()
		})
	}
}

func TestLogFieldsWithAttrs(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traceCloser, traceReporter := tracingtest.SetUpWithMockProcessor(t)
			ctx := context.Background()
			spanName := "op"
			span, ctx := tracing.StartSpanFromContext(ctx, spanName)
			span.SetTag("string", "val")
			span.SetTag("int64", int64(64))
			span.SetTag("uint64", uint64(6464))
			span.SetTag("int32", int32(32))
			span.SetTag("uint32", uint32(32))
			span.SetTag("float64", float64(64.64))
			span.SetTag("float32", float32(32.32))
			span.SetTag("uint", uint(8))
			span.SetTag("something", t)

			require.NotNil(t, span)
			require.NotNil(t, ctx)

			span.LogFields(tt.fields...)
			span.Finish()

			attrs, found := traceReporter.FindEventAttribute(spanName, tt.key)
			assert.True(t, found)
			assert.NotNil(t, attrs)
			t.Log(tt.fields[0].String())
			traceCloser()
		})
	}
}
