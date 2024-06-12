// Package tracing provides tracing initialization and abstractions over tracing internals.
package tracing

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/go-logr/logr"
	jaegerpropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	exporter "go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"gopkg/logging"
	loggingv2 "gopkg/logging/v2"
	"gopkg/tracing/configuration"
)

type conf struct {
	logger   logging.Logger
	loggerV2 loggingv2.Logger
	opts     []tracerProviderOpt
}

const (
	legacyProbabilisticSampler = "probabilistic"
	legacyConstantSampler      = "const"
)

// tracerProviderOpt is an interface that makes controller-gen-obj-all not fail.
// TODO: Fix interfaces below into type casted interfaces when controller-gen is upgraded to 0.9.0 or newer.
type tracerProviderOpt interface {
	tracesdk.TracerProviderOption
}

// SpanProcessor ...
type SpanProcessor interface {
	tracesdk.SpanProcessor
}

// InitGlobalTracer creates and initializes a new tracer with given options.
func InitGlobalTracer(options ...func(*conf) error) (closer io.Closer, err error) {
	c := &conf{logger: logging.NewLogger()}
	for _, option := range options {
		err = option(c)
		if err != nil {
			return
		}
	}

	return setTracer(c)
}

func setTracer(c *conf) (closer io.Closer, err error) {
	setLogger(c)

	opts, propagators, err := buildTracerProviderOptsAndPropagators()
	if err != nil {
		return nil, err
	}
	opts = append(opts, getTracerProviderOptsFromConf(c)...)
	tp := tracesdk.NewTracerProvider(
		opts...,
	)

	otel.SetTracerProvider(tp)

	// The createted composite TextMapPropagator will inject and extract cross-cutting concerns in the order the TextMapPropagators were provided.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))
	return contextCloser(tp.Shutdown), nil
}

type contextCloser func(ctx context.Context) error

func (c contextCloser) Close() error {
	return c(context.Background())
}

func buildTracerProviderOptsAndPropagators() (opts []tracesdk.TracerProviderOption, propagators []propagation.TextMapPropagator, err error) {
	cfg, err := getTracingConfig()
	if err != nil {
		return
	}
	withExporter, err := createWithBatcherExporterOpt(cfg)
	if err != nil {
		return opts, propagators, fmt.Errorf("failed creating tracing exporter: %w", err)
	}
	opts = append(opts, withExporter)

	withResource, err := createWithResourceOpt(cfg)
	if err != nil {
		return opts, propagators, fmt.Errorf("failed creating tracing reosurce: %w", err)
	}
	opts = append(opts, withResource)

	withSampler, err := createWithSamplerOpt(cfg)
	if err != nil {
		return opts, propagators, fmt.Errorf("failed creating tracing sampler: %w", err)
	}
	if withSampler != nil {
		opts = append(opts, withSampler)
	}

	propagators, err = parseOtelPropagators(cfg)
	if err != nil {
		return opts, propagators, fmt.Errorf("failed parsing propagators: %w", err)
	}

	return opts, propagators, nil
}

func createWithResourceOpt(cfg *configuration.TracingConfiguration) (tracesdk.TracerProviderOption, error) {
	r, err := resource.New(
		context.Background(),
		resource.WithHost(),
		resource.WithContainerID(),
		resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
	)
	if err != nil {
		return nil, err
	}
	return tracesdk.WithResource(r), nil
}

func createWithSamplerOpt(cfg *configuration.TracingConfiguration) (tracesdk.TracerProviderOption, error) {
	switch cfg.JaegerSamplerType {
	case legacyConstantSampler:
		enabled, err := parseConstantSamplerArg(cfg.JaegerSamplerParam)
		if err != nil {
			return nil, err
		}
		if enabled {
			return tracesdk.WithSampler(tracesdk.AlwaysSample()), nil
		}
		return tracesdk.WithSampler(tracesdk.NeverSample()), nil
	case legacyProbabilisticSampler:
		ratio, err := parseTraceIDRatio(cfg.JaegerSamplerParam, cfg.JaegerSamplerParam != "")
		if err != nil {
			return nil, err
		}
		return tracesdk.WithSampler(tracesdk.ParentBased(ratio)), nil
	default:
		return nil, nil // Use default OpenTelemetry sampler creation.
	}
}

func createWithBatcherExporterOpt(cfg *configuration.TracingConfiguration) (tracesdk.TracerProviderOption, error) {
	if cfg.JaegerEndpoint == "" {
		return tracesdk.WithBatcher(newNoopExporter()), nil
	}
	exp, err := exporter.New(exporter.WithCollectorEndpoint(exporter.WithEndpoint(cfg.JaegerEndpoint)))
	if err != nil {
		return nil, err
	}
	if cfg.UseSimpleSpanProcessor {
		return tracesdk.WithSpanProcessor(tracesdk.NewSimpleSpanProcessor(exp)), nil
	}
	return tracesdk.WithBatcher(exp), nil
}

// parseOtelPropagators parses the propagators for tracing from env variables
// currently supported propagators are: tracecontext, baggage and jaeger
func parseOtelPropagators(cfg *configuration.TracingConfiguration) (ops []propagation.TextMapPropagator, err error) {
	if len(cfg.OtelPropagators) == 0 {
		return ops, fmt.Errorf("no otel propagators found")
	}

	for _, op := range cfg.OtelPropagators {
		switch op {
		case "tracecontext":
			ops = append(ops, propagation.TraceContext{})
		case "baggage":
			ops = append(ops, propagation.Baggage{})
		case "jaeger":
			ops = append(ops, jaegerpropagator.Jaeger{})
		default:
			return ops, fmt.Errorf("propagator not supported: %s", op)
		}
	}
	return ops, nil
}

// WithLogger is wrapping conf with logger. Can be used as and opt for InitGlobalTracer.
func WithLogger(logger logging.Logger) func(*conf) error {
	return func(c *conf) (err error) {
		c.logger = logger
		return
	}
}

// WithV2Logger is wrapping configuration with logger v2. Can be used as and opt for InitGlobalTracer.
func WithV2Logger(logger loggingv2.Logger) func(*conf) error {
	return func(c *conf) (err error) {
		c.loggerV2 = logger
		return
	}
}

// WithProcessor is wrapping configuration with SpanProcessor. Used for initializing processor mocks. Can be used as and opt for InitGlobalTracer.
func WithProcessor(pros SpanProcessor) func(*conf) error {
	return func(c *conf) (err error) {
		c.opts = append(c.opts, tracesdk.WithSpanProcessor(pros))
		return
	}
}

func getTracingConfig() (*configuration.TracingConfiguration, error) {
	cfg, err := configuration.FromEnv()
	if err != nil {
		return nil, err
	}

	// This is a workaround for a bug https://github.com/jaegertracing/jaeger-client-go/issues/350
	// Now application won't fail if service name is not specified.
	if cfg.ServiceName == "" {
		cfg.ServiceName = "unused-service-name"
	}

	return cfg, nil
}

func setLogger(c *conf) {
	var logger logr.LogSink
	if c.logger != nil {
		logger = &logrLoggerAdapter{logger: c.logger}
	}
	if c.loggerV2 != nil {
		logger = &logrV2LoggerAdapter{logger: c.loggerV2}
	}
	logs := logr.New(logger)
	otel.SetLogger(logs)
	otel.SetErrorHandler(&TracerErrorHandler{logs})
}

var _ tracesdk.SpanExporter = (*noopExporter)(nil)

// newNoopExporter returns a new no-op exporter.
func newNoopExporter() *noopExporter {
	return new(noopExporter)
}

// noopExporter is an exporter that drops all received Spans and
// performs no action.
type noopExporter struct{}

// ExportSpans handles export of SpanSnapshots by dropping them.
func (nsb *noopExporter) ExportSpans(context.Context, []tracesdk.ReadOnlySpan) error { return nil }

// Shutdown stops the exporter by doing nothing.
func (nsb *noopExporter) Shutdown(context.Context) error { return nil }

func parseTraceIDRatio(arg string, hasSamplerArg bool) (tracesdk.Sampler, error) {
	if !hasSamplerArg {
		return tracesdk.TraceIDRatioBased(1.0), nil
	}
	v, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return tracesdk.TraceIDRatioBased(1.0), fmt.Errorf("failed to parse samping rate")
	}
	if v < 0.0 {
		return tracesdk.TraceIDRatioBased(1.0), fmt.Errorf("sampling rate cannot be smaller than 0")
	}
	if v > 1.0 {
		return tracesdk.TraceIDRatioBased(1.0), fmt.Errorf("sampling rate cannot be greater than 1")
	}

	return tracesdk.TraceIDRatioBased(v), nil
}

func parseConstantSamplerArg(arg string) (enabled bool, err error) {
	samplerParam, err := strconv.Atoi(arg)
	if err != nil {
		return false, err
	}
	return samplerParam == 1, nil
}

// getTracerProviderOptsFromConf is a workaround to inject tracerProviderOpt into the TracerProvider.
// TODO: To be removed when generator is upgraded.
func getTracerProviderOptsFromConf(c *conf) (opts []tracesdk.TracerProviderOption) {
	for _, opt := range c.opts {
		o, _ := opt.(tracesdk.TracerProviderOption)
		opts = append(opts, o)
	}
	return opts
}
