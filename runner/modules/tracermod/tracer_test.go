package tracermod_test

import (
	"testing"

	"github.com/phanitejak/gopkg/logging/loggingtest"
	"github.com/phanitejak/gopkg/runner/modules/tracermod"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
