package tracing

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"go.opentelemetry.io/otel/trace"

	"github.com/phanitejak/gopkg/logging"
)

// Logger is wrapping tracing span and logger together
type Logger struct {
	logging.Logger
	span Span
}

// For is logging with tracing info. Use this predominantly.
func (l *Logger) For(context context.Context) *Logger {
	span := SpanFromContext(context)
	if span == nil {
		return l
	}

	ctx := trace.SpanContextFromContext(context)
	if !ctx.IsValid() {
		return l
	}
	// NNEO-12959: Lost parent_id in refactoring, doesn't seem to be easily available anymore
	ctxLogger := &Logger{
		Logger: l.Logger.
			With("trace_id", ctx.TraceID().String()).
			With("span_id", ctx.SpanID().String()).
			With("is_sampled", fmt.Sprintf("%v", ctx.IsSampled())),
		span: span,
	}

	incLog, ok := ctxLogger.Logger.(depthInc)
	if ok {
		ctxLogger.Logger = incLog.IncDepth(1)
	}

	return ctxLogger
}

// Debug logs a message at level Debug.
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Debugln logs a message at level Debug.
func (l *Logger) Debugln(args ...interface{}) {
	l.Logger.Debugln(args...)
}

// Debugf logs a message at level Debug.
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// Info logs a message at level Info.
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Infoln logs a message at level Info.
func (l *Logger) Infoln(args ...interface{}) {
	l.Logger.Infoln(args...)
}

// Infof logs a message at level Info.
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// Error logs the error
func (l *Logger) Error(args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Error(args...)
}

// Errorln logs the error
func (l *Logger) Errorln(args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Errorln(args...)
}

// Errorf logs the error
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Errorf(format, args...)
}

// Fatal logs error and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Fatal(args...)
}

// Fatalln logs error and exits
func (l *Logger) Fatalln(args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Fatalln(args...)
}

// Fatalf logs error and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.addErrorTagIfSpanExists()
	l.Logger.Fatalf(format, args...)
}

func (l *Logger) addErrorTagIfSpanExists() {
	if l.span == nil {
		return
	}
	l.span.SetTag("error", true)
}

type depthInc interface {
	IncDepth(depth int) logging.Logger
}

// NewLogger accepts logger as a parameter and returns tracing logger.
func NewLogger(logger logging.Logger) *Logger {
	l, ok := logger.(depthInc)
	if ok {
		logger = l.IncDepth(1)
	}
	return &Logger{Logger: logger}
}

// TracerErrorHandler ...
type TracerErrorHandler struct {
	log logr.Logger
}

// Handle ...
func (e *TracerErrorHandler) Handle(err error) {
	e.log.Error(err, "error during tracing handling")
}
