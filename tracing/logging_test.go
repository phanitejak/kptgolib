package tracing_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/phanitejak/gopkg/logging"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/phanitejak/gopkg/tracing/tracingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingForBackgroundContextShouldWork(t *testing.T) {
	logger := tracing.NewLogger(logging.NewLogger())

	logger.For(context.Background()).Info("Test")
}

func TestLoggingForEmptyContextShouldWork(t *testing.T) {
	logger := tracing.NewLogger(logging.NewLogger())

	logger.For(context.TODO()).Info("Test")
}

func TestLoggingWithoutContextShouldWork(t *testing.T) {
	logger := tracing.NewLogger(logging.NewLogger())

	logger.Info("Test")
}

func TestLoggingForContext(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	_, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")

	logger := tracing.NewLogger(logging.NewLogger())

	logger.For(ctx).Info("Test")
	logger.For(ctx).Infof("Test")
	logger.For(ctx).Infoln("Test")
	logger.For(ctx).Debug("Test")
	logger.For(ctx).Debugf("Test")
	logger.For(ctx).Debugln("Test")
	logger.For(ctx).Error("Test")
	logger.For(ctx).Errorf("Test")
	logger.For(ctx).Errorln("Test")
}

func TestShouldLogIsSampledAsString(t *testing.T) {
	cleanUp := tracingtest.SetUp(t)
	defer cleanUp()

	_, ctx := tracing.StartSpanFromContext(context.Background(), "testSpan")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	stderr := os.Stderr
	defer func() {
		os.Stderr = stderr
	}()

	os.Stderr = w
	logger := tracing.NewLogger(logging.NewLogger())

	logger.For(ctx).Info("Test")

	err = w.Close()
	require.NoError(t, err)

	decoder := json.NewDecoder(r)

	logEntry := struct {
		IsSampled string `json:"is_sampled"`
		TraceID   string `json:"trace_id"`
		SpanID    string `json:"span_id"`
		Level     string `json:"level"`
		Logger    string `json:"logger"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}{}

	err = decoder.Decode(&logEntry)
	require.NoError(t, err)
}

func TestLogFatal(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := tracing.NewLogger(logging.NewLogger())

		logger.For(ctx).Fatal("Crashing application")
		return
	}
	runTest("TestLogFatal", t)
}

func TestLogFatalf(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := tracing.NewLogger(logging.NewLogger())

		logger.For(ctx).Fatalf("Crashing application")
		return
	}
	runTest("TestLogFatalf", t)
}

func TestLogFatalln(t *testing.T) {
	if os.Getenv("CRASH_APPLICATION") == "1" {
		_, ctx := tracing.StartSpanFromContext(context.Background(), "crashingSpan")

		logger := tracing.NewLogger(logging.NewLogger())

		logger.For(ctx).Fatalln("Crashing application")
		return
	}
	runTest("TestLogFatalln", t)
}

func runTest(testName string, t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run="+testName) //nolint:gosec
	cmd.Env = append(os.Environ(), "CRASH_APPLICATION=1")
	err := cmd.Run()
	var e *exec.ExitError
	ok := errors.As(err, &e)
	require.True(t, ok, "error should be of type ExitError")
	assert.False(t, e.Success(), "error should not have status success")
	assert.Equal(t, "exit status 1", e.String())
}

func TestLoggingFromExecutable(t *testing.T) {
	const file = "wrap.go"
	cmd := exec.Command("go", "run", "testdata/"+file) //nolint: gosec
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)

	var msg struct {
		Level      string    `json:"level"`
		Logger     string    `json:"logger"`
		Message    string    `json:"message"`
		StackTrace string    `json:"stack_trace"`
		Timestamp  time.Time `json:"timestamp"`
	}

	lines := bytes.Split(out, []byte("\n"))
	for i, lvl := range []string{"debug", "info", "error", "debug", "info", "error", "debug", "info", "error"} {
		line := 25 + i
		require.NoError(t, json.Unmarshal(lines[i], &msg))
		assert.Equalf(t, lvl, msg.Level, "%s:%d has unexpected output:\n%s", file, line, lines[i])
		assert.Equalf(t, fmt.Sprintf("%s:%d", file, line), msg.Logger, "%s:%d has unexpected output:\n%s", file, line, lines[i])
		assert.Equalf(t, lvl+" msg", msg.Message, "%s:%d has unexpected output:\n%s", file, line, lines[i])
	}
}
