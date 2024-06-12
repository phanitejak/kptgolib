package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

// This file contains the functionality used by our services to implement tracing.
// The goal is to reduce the amount of external imports needed in the services.
// Please add new type aliases and variables if needed in the service code.

const (
	defaultTracerName = "nom.default.tracer"
)

// Helper attributes for tagging basic span concepts.
var (
	HTTPUrl        = semconv.HTTPURLKey
	HTTPMethod     = semconv.HTTPMethodKey
	HTTPStatusCode = semconv.HTTPStatusCodeKey
)

// Type aliases for tracing internals to keep them unexposed in service level.
// nolint
type (
	HTTPHeadersCarrier = propagation.HeaderCarrier
	TextMapCarrier     = propagation.MapCarrier
	SpanContext        = trace.SpanContext
	SpanStartOption    = trace.SpanStartOption
)

type legacySpan struct {
	trace.Span
}

func toLegacySpan(span trace.Span) *legacySpan {
	return &legacySpan{span}
}

// Span ...
type Span interface {
	trace.Span
	SetTag(key string, value interface{})
	Finish()
	LogFields(fields ...Field)
	SetOperationName(name string)
}

// StartSpan ...
func StartSpan(operationName string, opts ...SpanStartOption) (Span, context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer(defaultTracerName).Start(context.Background(), operationName, opts...)
	return toLegacySpan(span), ctx
}

// StartSpanFromContext ...
func StartSpanFromContext(ctx context.Context, operationName string, opts ...SpanStartOption) (Span, context.Context) {
	spanCtx, span := otel.GetTracerProvider().Tracer(defaultTracerName).Start(ctx, operationName, opts...)
	return toLegacySpan(span), spanCtx
}

// SetTag sets the given key and value as OpenTelemetry attributes.
func (l *legacySpan) SetTag(key string, value interface{}) {
	l.Span.SetAttributes(KeyValueToAttribute(key, value))
}

// LogFields logs the given fields as OpenTelemetry events.
func (l *legacySpan) LogFields(fields ...Field) {
	l.Span.AddEvent("", trace.WithAttributes(legacyLogFieldsToAttributes(fields)...))
}

// SetOperationName sets a name for the span.
func (l *legacySpan) SetOperationName(name string) {
	l.Span.SetName(name)
}

// Finish ...
func (l *legacySpan) Finish() {
	l.Span.End()
}

// FollowsFrom ...
func FollowsFrom(ctx context.Context) trace.SpanStartOption {
	link := trace.LinkFromContext(ctx, attribute.Key("ot-span-reference-type").String("follows-from-ref"))
	return trace.WithLinks(link)
}

// SpanFromContext ...
func SpanFromContext(ctx context.Context) Span {
	return toLegacySpan(trace.SpanFromContext(ctx))
}

// GetTraceIDFromContext gets the traceID of given context. Context needs to be valid trace.SpanContext.
func GetTraceIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("nil context")
	}

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return "", fmt.Errorf("span context not valid, traceID: %s", spanCtx.TraceID().String())
	}
	return spanCtx.TraceID().String(), nil
}

// ExtractSpanData extracts info from span that can be used by v2 logger.
func ExtractSpanData(ctx context.Context, isError bool) (traceID, spanID, isSampled string, ok bool) {
	span := SpanFromContext(ctx)
	if span == nil {
		return
	}

	if isError {
		span.SetAttributes(attribute.Key("error").Bool(true))
	}

	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return
	}

	return spanCtx.TraceID().String(), spanCtx.SpanID().String(), fmt.Sprintf("%v", spanCtx.IsSampled()), true
}
