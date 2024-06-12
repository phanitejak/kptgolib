// Package configuration is about logging and environment
package configuration

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// TracingConfiguration is env configuration
type TracingConfiguration struct {
	ServiceName            string   `envconfig:"JAEGER_SERVICE_NAME"`
	JaegerEndpoint         string   `envconfig:"JAEGER_ENDPOINT" default:""`
	JaegerSamplerType      string   `envconfig:"JAEGER_SAMPLER_TYPE" default:"probabilistic"`
	JaegerSamplerParam     string   `envconfig:"JAEGER_SAMPLER_PARAM" default:"0"`
	JaegerReporterLogSpans bool     `envconfig:"JAEGER_REPORTER_LOG_SPANS"`
	UseSimpleSpanProcessor bool     `envconfig:"USE_SIMPLE_SPAN_PROCESSOR" default:"false"`
	OtelPropagators        []string `envconfig:"OTEL_PROPAGATORS" default:"tracecontext,baggage,jaeger"`
}

// FromEnv ...
func FromEnv() (*TracingConfiguration, error) {
	config := &TracingConfiguration{}
	err := envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TracingConfiguration: %w", err)
	}
	return config, nil
}
