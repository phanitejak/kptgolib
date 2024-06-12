package tracing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-logr/logr"

	"github.com/phanitejak/kptgolib/logging"
	loggingv2 "github.com/phanitejak/kptgolib/logging/v2"
	"github.com/phanitejak/kptgolib/tracing/configuration"

	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func TestWithLogger(t *testing.T) {
	// Try to set a duplicate tracerProvider
	c := &conf{logger: logging.NewLogger()}
	setLogger(c)
	c.logger.Info("Setting a duplicate tracerProvider produces an error")
	otel.SetTracerProvider(otel.GetTracerProvider())

	// Try to init a globalTracer with logger
	logger := logging.NewLogger()
	closer, err := InitGlobalTracer(WithLogger(logger))
	require.NoError(t, err)
	assert.NoError(t, closer.Close())
}

func TestWithLoggerV2(t *testing.T) {
	// Try to init a globalTracer with loggerV2
	v2logger := loggingv2.NewLogger()
	closer, err := InitGlobalTracer(WithV2Logger(v2logger))
	require.NoError(t, err)
	assert.NoError(t, closer.Close())
}

func TestLogrAdapterV1(t *testing.T) {
	var loggerv1 logr.LogSink
	var loggerv2 logr.LogSink
	loggerv1 = &logrLoggerAdapter{logger: logging.NewLogger()}
	loggerv2 = &logrV2LoggerAdapter{logger: loggingv2.NewLogger()}

	logsv1 := logr.New(loggerv1)
	logsv2 := logr.New(loggerv2)

	kvWithOne := []interface{}{"test-key-1", "test-val-1"}
	kvWithTwo := []interface{}{"test-key-1", "test-val-1", "test-key-2", "test-val-2"}

	logsv1.Info("Test message", kvWithOne...)
	logsv1.Info("Test message", kvWithTwo...)

	logsv2.Info("Test message", kvWithOne...)
	logsv2.Info("Test message", kvWithTwo...)

	logsv1.Info("Test message", nil)
	logsv2.Info("Test message", nil)

	logsv1.WithName("name")
	logsv2.WithName("name")

	logsv1.Error(errors.New("sError"), "string error")
	logsv2.Error(errors.New("sError"), "string error")
}

func TestTraceIDRatio(t *testing.T) {
	type test struct {
		ratio         string
		hasSamplerArg bool
	}

	testsNoSamplerArgs := []test{
		{ratio: "0", hasSamplerArg: false},
		{ratio: "0.5", hasSamplerArg: false},
		{ratio: "1", hasSamplerArg: false},
	}
	for _, tc := range testsNoSamplerArgs {
		_, err := parseTraceIDRatio(tc.ratio, tc.hasSamplerArg)
		require.NoError(t, err)
	}

	testsWithOkValues := []test{
		{ratio: "0", hasSamplerArg: true},
		{ratio: "0.255123213", hasSamplerArg: true},
		{ratio: "0.5", hasSamplerArg: true},
		{ratio: "1", hasSamplerArg: true},
	}
	for _, tc := range testsWithOkValues {
		_, err := parseTraceIDRatio(tc.ratio, tc.hasSamplerArg)
		require.NoError(t, err)
	}

	testsWithUnsuitableValues := []test{
		{ratio: "-1", hasSamplerArg: true},
		{ratio: "2", hasSamplerArg: true},
		{ratio: "test", hasSamplerArg: true},
	}
	for _, tc := range testsWithUnsuitableValues {
		_, err := parseTraceIDRatio(tc.ratio, tc.hasSamplerArg)
		require.Error(t, err)
	}
}

func TestTracingConfigurations(t *testing.T) {
	tc := configuration.TracingConfiguration{}

	// Batcher Exporter with random jaeger endpoint
	tc.JaegerEndpoint = "test-endpoint"

	_, err1 := createWithBatcherExporterOpt(&tc)
	require.NoError(t, err1)

	// Batcher Exporter with random jaeger endpoint and SimpleProcessor
	tc.UseSimpleSpanProcessor = true

	_, err2 := createWithBatcherExporterOpt(&tc)
	require.NoError(t, err2)

	// Sampler that never samples
	tc.JaegerSamplerType = "const"
	tc.JaegerSamplerParam = "0"
	_, err3 := createWithSamplerOpt(&tc)
	require.NoError(t, err3)

	// Nil sampler
	tc.JaegerSamplerType = "blahblah"
	sampler, err4 := createWithSamplerOpt(&tc)
	require.Nil(t, sampler)
	require.NoError(t, err4)
}

func TestOtelPropagatorParsing(t *testing.T) {
	tc := configuration.TracingConfiguration{}

	// correctly set propagator values
	tc.OtelPropagators = []string{"tracecontext", "baggage", "jaeger"}
	defaults, err1 := parseOtelPropagators(&tc)
	require.NoError(t, err1)
	require.Equal(t, defaults, []propagation.TextMapPropagator{propagation.TraceContext{}, propagation.Baggage{}, jaeger.Jaeger{}})

	// no propagators set
	tc.OtelPropagators = []string{}
	_, err2 := parseOtelPropagators(&tc)
	require.ErrorContains(t, err2, "no otel propagators found")

	// unsupported propagator in the set
	tc.OtelPropagators = []string{"tracecontext", "fake_propagator"}
	_, err3 := parseOtelPropagators(&tc)
	require.ErrorContains(t, err3, "propagator not supported: fake_propagator")
}

func TestTracingOptsBuilderErrors(t *testing.T) {
	// invalid sampling rate
	t.Setenv("JAEGER_SAMPLER_PARAM", "20")
	_, _, err1 := buildTracerProviderOptsAndPropagators()
	require.ErrorContains(t, err1, "failed creating tracing sampler")
	t.Setenv("JAEGER_SAMPLER_PARAM", "0")

	// no propagators
	t.Setenv("OTEL_PROPAGATORS", "")
	_, _, err2 := buildTracerProviderOptsAndPropagators()
	require.ErrorContains(t, err2, "failed parsing propagators")

	// Failing conversion from env variables
	t.Setenv("JAEGER_REPORTER_LOG_SPANS", "notbool")
	_, _, err3 := buildTracerProviderOptsAndPropagators()
	require.ErrorContains(t, err3, "failed to parse TracingConfiguration")
}
