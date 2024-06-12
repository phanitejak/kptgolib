// nolint
package tracingtest

import (
	"context"
	"testing"

	"github.com/phanitejak/kptgolib/tracing"
	"github.com/phanitejak/kptgolib/tracing/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

const (
	OperationID = "afa0a9f161f6442e8c0cb626697f5393"
)

func TestBasicSetup(t *testing.T) {
	cleanUp := SetUp(t)
	defer cleanUp()

	require.NotNil(t, cleanUp, "Setup must be completed successfully")
}

func TestMockProcessor(t *testing.T) {
	expectedAttrs := []attribute.KeyValue{
		attribute.Key("key1").String("value1"),
		attribute.Key("key2").String("value2"),
		attribute.Key("key3").String("value3"),
	}

	cleanUp, mp := SetUpWithMockProcessor(t)
	defer cleanUp()
	require.NotNil(t, mp, "MockProcessor must be set up correctly")

	for x := 0; x < 2; x++ {
		span, _ := tracing.StartSpan("test-operation")
		require.NotNil(t, span, "Span must be created correctly")
		for _, keyvalue := range expectedAttrs {
			span.SetAttributes(keyvalue)
		}
		span.LogFields([]tracing.Field{
			tracing.String("stringkey", "val"),
			tracing.String("stringkey2", "val2"),
		}...)
		span.Finish() // Finish span to populate the spanStorage in MockProcessor
	}

	expectedFalse := mp.SpanNameExist("noSpanByThisName")
	assert.False(t, expectedFalse, "should return false when span name is invalid")

	expectedTrue := mp.SpanNameExist("test-operation")
	assert.True(t, expectedTrue, "should return true when span name is correct")

	_, expectedErr := mp.GetAttributes("noSpanByThisName")
	assert.Error(t, expectedErr, "should be an error when requesting the attributes for a span that does not exist")

	expectedAttributes, err := mp.GetAttributes("test-operation")
	assert.NoError(t, err, "should be no error when fetching existing span")
	assert.NotNil(t, expectedAttributes, "return should contain valid data")
	for _, v := range expectedAttributes {
		assert.EqualValues(t, KeyValueToMap(v), KeyValueToMap(expectedAttrs), "expected values should match actually returned values")
	}

	_, expectedFalse = mp.FindAttribute("noSpanByThisName", "testTag")
	assert.False(t, expectedFalse, "should return false when incorrect span name is given")

	expectedAttribute, expectedTrue := mp.FindAttribute("test-operation", "key1")
	assert.True(t, expectedTrue)
	assert.Equal(t, "value1", expectedAttribute)

	expectedNil := mp.ForceFlush(context.Background())
	assert.Nil(t, expectedNil, "should be nil as ForceFlush currently can't produce other results")

	assert.Equal(t, 2, mp.GetSpanAmount("test-operation"))
	assert.NoError(t, mp.CheckSpansNotNil(t, "test-operation"))

	eventattr, ok := mp.FindEventAttribute("test-operation", "stringkey")
	assert.True(t, ok)
	assert.Equal(t, "stringkey", string(eventattr.Key))

	eventval, ok := mp.FindEventAttributeValue("test-operation", "stringkey")
	assert.True(t, ok)
	assert.Equal(t, "val", eventval)

	mp.Reset() // Flush all data from MockProcessor's spanStorage

	_, ok = mp.FindEventAttribute("test-operation", "stringkey")
	assert.False(t, ok)
	_, ok = mp.FindEventAttributeValue("test-operation", "stringkey")
	assert.False(t, ok)

	assert.Equal(t, 0, mp.GetSpanAmount("test-operation"))
	assert.Error(t, mp.CheckSpansNotNil(t, "test-operation"))
}

func TestMockExporter(t *testing.T) {
	cleanUp, me := SetUpWithMockExporter(t)
	defer cleanUp()
	assert.NotNil(t, me, "MockExporter must be set up correctly")

	span, _, err := utils.SpanFromOID(OperationID, "test-operation")
	require.NotNil(t, span, "Span must be created correctly")
	require.NoError(t, err, "Must be no error when creating valid span")
	span.SetTag("testTag", "testValue")
	span.Finish() // Finish span to populate the spanStorage in MockExporter

	_, expectedErr := me.GetAttributes("noSpanByThisName")
	assert.Error(t, expectedErr, "should be an error when fetching an incorrect span name")

	expectedAttributes, err := me.GetAttributes("test-operation")
	assert.NoError(t, err, "should be no error when fetching valid span name")
	assert.NotNil(t, expectedAttributes, "when fetching valid span name, should produce results")
}
