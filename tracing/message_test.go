package tracing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/IBM/sarama"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/phanitejak/gopkg/tracing/tracingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartSpanFromMessageCheckProcessor(t *testing.T) {
	cleanUp, mockPros := tracingtest.SetUpWithMockProcessor(t)
	defer cleanUp()

	msg := defaultTestMessage
	expectedAttributes := []attribute.KeyValue{
		attribute.Key(tracing.KafkaSpanKindTagName).String("consumer"),
		attribute.Key(tracing.KafkaTopicTagName).String(msg.Topic),
		attribute.Key(tracing.KafkaPartitionTagName).Int(int(msg.Partition)),
		attribute.Key(tracing.KafkaKeyTagName).String(string(msg.Key)),
		attribute.Key(tracing.KafkaTimestampTagName).String(msg.Timestamp.String()),
	}

	operationName := "NewMessage"
	span, _ := tracing.StartSpanFromMessage(msg, operationName)
	spanContext := span.SpanContext()
	require.True(t, spanContext.IsValid())

	sampled := 0
	if spanContext.IsSampled() {
		sampled = 1
	}

	assert.Equal(t, traceID, spanContext.TraceID().String())
	assert.NotEqual(t, spanID, spanContext.SpanID().String(), "new span should have a new spanID")
	assert.Equal(t, 1, sampled)

	span.End()
	attr, err := mockPros.GetAttributes(operationName)
	assert.NoError(t, err)
	assert.Len(t, attr, 1)
	for _, v := range attr {
		assert.EqualValues(t, tracingtest.KeyValueToMap(expectedAttributes), tracingtest.KeyValueToMap(v))
	}
}

func TestStartSpanFromMessageCheckExporter(t *testing.T) {
	cleanUp, mockExporter := tracingtest.SetUpWithMockExporter(t)
	defer cleanUp()

	msg := defaultTestMessage
	expectedAttributes := []attribute.KeyValue{
		attribute.Key(tracing.KafkaSpanKindTagName).String("consumer"),
		attribute.Key(tracing.KafkaTopicTagName).String(msg.Topic),
		attribute.Key(tracing.KafkaPartitionTagName).Int(int(msg.Partition)),
		attribute.Key(tracing.KafkaKeyTagName).String(string(msg.Key)),
		attribute.Key(tracing.KafkaTimestampTagName).String(msg.Timestamp.String()),
	}

	operationName := "NewMessage"
	span, _ := tracing.StartSpanFromMessage(msg, operationName)
	spanContext := span.SpanContext()
	require.True(t, spanContext.IsValid())

	sampled := 0
	if spanContext.IsSampled() {
		sampled = 1
	}

	assert.Equal(t, traceID, spanContext.TraceID().String())
	assert.Equal(t, 1, sampled)

	span.End()
	attr, err := mockExporter.GetAttributes(operationName)
	assert.NoError(t, err)
	assert.EqualValues(t, tracingtest.KeyValueToMap(expectedAttributes), tracingtest.KeyValueToMap(attr))
}

func TestMessageWithContext(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	span, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")

	spanContext := span.SpanContext()
	require.True(t, spanContext.IsValid())

	key := "my-key"
	msg := &sarama.ProducerMessage{
		Topic:     "my-topic",
		Key:       sarama.StringEncoder(key),
		Partition: 3,
	}

	messageWithContext := tracing.MessageWithContext(ctx, msg)

	assertHeaderKeyExists(t, "traceparent", messageWithContext)
	value := assertHeaderKeyExists(t, "uber-trace-id", messageWithContext)
	assert.Equal(t, uberTraceFromSpan(span), string(value))
	assert.Equal(t, msg.Key, messageWithContext.Key)
	assert.Equal(t, msg.Topic, messageWithContext.Topic)
	assert.Equal(t, msg.Partition, messageWithContext.Partition)

	span.End()
}

func TestMessageWithoutContext(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	messageWithContext := tracing.MessageWithContext(context.Background(), &sarama.ProducerMessage{})
	assert.Equal(t, 0, len(messageWithContext.Headers))
}

var (
	traceID            = "12341234123412341234123412341234"
	spanID             = "4321432143214321"
	defaultTestMessage = &sarama.ConsumerMessage{
		Topic:     "my-topic",
		Key:       []byte("my-key"),
		Timestamp: time.Now(),
		Partition: 5,
		Headers: []*sarama.RecordHeader{
			// Tracing header
			{Key: []byte("uber-trace-id"), Value: []byte(fmt.Sprintf("%s:%s:0:1", traceID, spanID))},
			// Normal strings
			{Key: []byte("SomeHeaderKey"), Value: []byte("having_some_value")},
			{Key: []byte("and-some-other-header-key"), Value: []byte("withSomeOtherValue")},
			// Empty strings
			{Key: []byte("empty-value"), Value: []byte("")},
			{Key: []byte(""), Value: []byte("empty key")},
			{Key: nil, Value: []byte("nil key")},
			{Key: []byte("nil-value"), Value: nil},
			// Invalid utf8 strings
			{Key: []byte{0x99, 0x99, 0x99}, Value: []byte{0x99, 0x99, 0x99}},
			{Key: []byte("Bin-data"), Value: []byte{0x99, 0x99, 0x99}},
			{Key: []byte{0x99, 0x99, 0x99}, Value: []byte("binary-key")},
		},
	}
)

func assertHeaderKeyExists(t *testing.T, key string, msg *sarama.ProducerMessage) []byte {
	for _, header := range msg.Headers {
		if string(header.Key) == key {
			return header.Value
		}
	}
	assert.Failf(t, "Key %s not found in kafka-message-headers", key)
	return nil
}
