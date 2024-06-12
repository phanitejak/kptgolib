package tracing

import (
	"context"
	"unicode/utf8"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/IBM/sarama"
)

const (
	KafkaSpanKindTagName  = "span.kind"
	KafkaTopicTagName     = "message_bus.destination"
	KafkaPartitionTagName = "kafka.partition"
	KafkaKeyTagName       = "kafka.key"
	KafkaTimestampTagName = "kafka.timestamp"
)

// MessageWithContext adds context to message headers, so consumers will be aware of trace context.
func MessageWithContext(ctx context.Context, msg *sarama.ProducerMessage) *sarama.ProducerMessage {
	span := SpanFromContext(ctx)
	if span == nil {
		return msg
	}

	textMapCarriers := TextMapCarrier(make(map[string]string))

	otel.GetTextMapPropagator().Inject(ctx, textMapCarriers)

	for key, value := range textMapCarriers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{Key: []byte(key), Value: []byte(value)})
	}

	span.SetAttributes(
		attribute.Key(KafkaSpanKindTagName).String("producer"),
		attribute.Key(KafkaTopicTagName).String(msg.Topic),
		attribute.Key(KafkaPartitionTagName).Int(int(msg.Partition)),
	)
	if msg.Key == nil {
		return msg
	}

	if key, err := msg.Key.Encode(); err == nil && utf8.Valid(key) {
		span.SetAttributes(attribute.Key(KafkaKeyTagName).String(string(key)))
	}

	return msg
}

// StartSpanFromMessage creates a new span from kafka message
// If message contains tracing headers it will create span, following existing trace span.
func StartSpanFromMessage(msg *sarama.ConsumerMessage, operationName string) (Span, context.Context) {
	return StartSpanFromMessageWithContext(context.Background(), msg, operationName)
}

// StartSpanFromMessageWithContext Creates a new span from kafka message
// If message contains tracing headers it will create span, following existing trace span
// Returned context is child of input context.
func StartSpanFromMessageWithContext(ctx context.Context, msg *sarama.ConsumerMessage, operationName string) (Span, context.Context) {
	carrier := &TextMapCarrier{}
	for _, header := range msg.Headers {
		if string(header.Key) != "" {
			carrier.Set(string(header.Key), string(header.Value))
		}
	}

	span, spanCtx := StartSpanFromContext(otel.GetTextMapPropagator().Extract(ctx, carrier), operationName)

	span.SetAttributes(
		attribute.Key(KafkaSpanKindTagName).String("consumer"),
		attribute.Key(KafkaTopicTagName).String(msg.Topic),
		attribute.Key(KafkaPartitionTagName).Int(int(msg.Partition)),
		attribute.Key(KafkaTimestampTagName).String(msg.Timestamp.String()),
	)

	if utf8.Valid(msg.Key) {
		span.SetAttributes(attribute.Key(KafkaKeyTagName).String(string(msg.Key)))
	}

	return span, spanCtx
}
