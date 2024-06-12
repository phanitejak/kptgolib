package loggingtest_test

import (
	"context"
	"os/exec"
	"testing"

	"github.com/phanitejak/gopkg/logging/v2"
	"github.com/phanitejak/gopkg/logging/v2/loggingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLogger(t *testing.T) {
	logger := loggingtest.NewTestLogger(t)

	logger.Debug(context.Background(), "message")
	logger.Debugln(context.Background(), "message")
	logger.Debugf(context.Background(), "%s", "message")
	logger.Info(context.Background(), "message")
	logger.Infoln(context.Background(), "message")
	logger.Infof(context.Background(), "%s", "message")
	logger.Error(context.Background(), "message")
	logger.Errorln(context.Background(), "message")
	logger.Errorf(context.Background(), "%s", "message")

	logger.With("key", "value").Info(context.Background(), "message")
	logger.WithFields(map[string]interface{}{"key": "value"}).Info(context.Background(), "message")
}

func TestLoggingFatal(t *testing.T) {
	cmd := exec.Command("go", "test", "./testdata")
	out, err := cmd.Output()
	e := &exec.ExitError{}
	require.ErrorAs(t, err, &e)
	assert.Equal(t, 1, e.ExitCode())
	assert.Contains(t, string(out), "foo_test.go:12: error")
	assert.Contains(t, string(out), "foo_test.go:13: fatal")
}

func TestLoggerImplementsIncDepth(t *testing.T) {
	type incremental interface {
		IncDepth(depth int) logging.Logger
	}

	log := loggingtest.NewTestLogger(t)
	l, ok := log.(incremental)
	require.True(t, ok, "logger doesn't implement IncDepth(depth int) Logger method")
	log = l.IncDepth(0)
	log.Debug(context.Background())
}
