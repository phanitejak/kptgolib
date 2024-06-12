package tracermod_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg/logging/loggingtest"
	"gopkg/runner/modules/tracermod"
	"gopkg/tracing"
)

func TestNewGlobalTracer(t *testing.T) {
	tracer := tracermod.NewGlobalTracer()

	err := tracer.Init(tracing.NewLogger(loggingtest.NewTestLogger(t)))
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := tracer.Run()
		assert.NoError(t, err)
	}()

	err = tracer.Close()
	require.NoError(t, err)
	<-done
}
