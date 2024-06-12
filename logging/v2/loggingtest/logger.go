// Package loggingtest provides wrapper for testing.T to used as logging.Logger
package loggingtest

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/phanitejak/gopkg/logging/v2"
)

// TestLogger is decorating testing.T with logging.Logger.
type TestLogger struct {
	t     *testing.T
	depth int
}

// NewTestLogger is wrapping testing.T and offering a logging.Logger.
func NewTestLogger(t *testing.T) logging.Logger {
	return TestLogger{
		t: t,
	}
}

// Debug is logging arguments using t.Log.
func (t TestLogger) Debug(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Debugln is logging arguments using t.Log.
func (t TestLogger) Debugln(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Debugf is logging arguments using t.Logf.
func (t TestLogger) Debugf(_ context.Context, msg string, args ...interface{}) {
	t.t.Logf(t.formatf(msg, args...))
}

// Info is logging arguments using t.Log.
func (t TestLogger) Info(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Infoln is logging arguments using t.Log.
func (t TestLogger) Infoln(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Infof is logging arguments using t.Logf.
func (t TestLogger) Infof(_ context.Context, msg string, args ...interface{}) {
	t.t.Logf(t.formatf(msg, args...))
}

// Error is logging arguments using t.Log instead of t.Error in case error level logging is expected.
func (t TestLogger) Error(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Errorln is logging arguments using t.Log instead of t.Error in case error level logging is expected.
func (t TestLogger) Errorln(_ context.Context, args ...interface{}) {
	t.t.Log(t.format(args...))
}

// Errorf is logging arguments using t.Logf instead of t.Errorf in case error level logging is expected.
func (t TestLogger) Errorf(_ context.Context, msg string, args ...interface{}) {
	t.t.Logf(t.formatf(msg, args...))
}

// With is not supported for test logger.
func (t TestLogger) With(_ string, _ interface{}) logging.Logger {
	return t
}

// WithFields is not supported for test logger.
func (t TestLogger) WithFields(_ map[string]interface{}) logging.Logger {
	return t
}

// Fatal is logging arguments using t.Fatal.
func (t TestLogger) Fatal(_ context.Context, args ...interface{}) {
	t.t.Fatal(t.format(args...))
}

// Fatalln is logging arguments using t.Fatal.
func (t TestLogger) Fatalln(_ context.Context, args ...interface{}) {
	t.t.Fatal(t.format(args...))
}

// Fatalf is logging arguments  using t.Fatalf.
func (t TestLogger) Fatalf(_ context.Context, msg string, args ...interface{}) {
	t.t.Fatalf(t.formatf(msg, args...))
}

// IncDepth can be used by wrappers to increment stack depth.
func (t TestLogger) IncDepth(depth int) logging.Logger {
	t.depth += depth
	return t
}

func (t TestLogger) format(args ...interface{}) string {
	i := t.source()
	return "\n" + i + ": " + fmt.Sprint(args...)
}

func (t TestLogger) formatf(msg string, args ...interface{}) string {
	i := t.source()
	return "\n" + i + ": " + fmt.Sprintf(msg, args...)
}

func (t TestLogger) source() string {
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		return "<???>:0"
	}
	slash := strings.LastIndex(file, "/")
	file = file[slash+1:]
	return fmt.Sprintf("%s:%d", file, line)
}
