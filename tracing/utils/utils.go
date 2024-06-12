// Package utils provides utility methods for the operations
package utils

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"gopkg/tracing"
)

// SpanFromOID is for creating the tracing Span with specified oID as trace ID, operationName as operationName.
// Note: using this method in a different context may break the tracing logic.
func SpanFromOID(oID string, operationName string) (span tracing.Span, spanContext context.Context, err error) {
	if len(oID) != 32 {
		return nil, spanContext, fmt.Errorf("the length of oID is not equal to 32, oID is %s", oID)
	}
	spanID := oID[16:32]
	carrier := &tracing.TextMapCarrier{}

	carrier.Set("uber-trace-id", fmt.Sprintf("%s:%s:0:1", oID, spanID))
	span, spanContext = tracing.StartSpanFromContext(otel.GetTextMapPropagator().Extract(context.Background(), carrier), operationName)
	return span, spanContext, nil
}

// GetOperationIDFromContext retrieves the operationID from the given context
func GetOperationIDFromContext(ctx context.Context) (string, error) {
	operationID, err := tracing.GetTraceIDFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get operationID from context: %w", err)
	}
	return operationID, nil
}
