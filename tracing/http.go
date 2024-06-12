// Package tracing is about instrumenting open tracing
package tracing

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// RequestWithContext adds context to request headers so server will be aware of trace context.
func RequestWithContext(r *http.Request, ctx context.Context) *http.Request {
	span := SpanFromContext(ctx)
	if span == nil {
		return r
	}

	span.SetAttributes(
		HTTPMethod.String(r.Method),
		HTTPUrl.String(r.URL.String()),
		attribute.Key("span.kind").String("client"),
	)

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	return r
}

// Wrap route to send traces.
func Wrap(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(http.HandlerFunc(handler.ServeHTTP), "", otelhttp.WithSpanNameFormatter(nameFormatter))
}

func nameFormatter(_ string, r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}
