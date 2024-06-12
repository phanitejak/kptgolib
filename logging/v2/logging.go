// Package logging provides compatibility with Neo logging guidelines
package logging

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
)

const (
	// ISO8601 is timestamp format used by Logger
	ISO8601        = "2006-01-02T15:04:05.000Z07:00"
	loggerFieldKey = "logger"
)

// store exit method in variable so it can be patched in test
var exit = os.Exit

// Logger is the interface for loggers.
type Logger interface {
	Debug(ctx context.Context, v ...interface{})
	Debugln(ctx context.Context, v ...interface{})
	Debugf(ctx context.Context, f string, v ...interface{})

	Info(ctx context.Context, v ...interface{})
	Infoln(ctx context.Context, v ...interface{})
	Infof(ctx context.Context, f string, v ...interface{})

	Error(ctx context.Context, v ...interface{})
	Errorln(ctx context.Context, v ...interface{})
	Errorf(ctx context.Context, f string, v ...interface{})

	With(key string, value interface{}) Logger
	WithFields(map[string]interface{}) Logger

	// log with Error* and exit
	Fatal(ctx context.Context, v ...interface{})
	Fatalln(ctx context.Context, v ...interface{})
	Fatalf(ctx context.Context, f string, v ...interface{})
}

// StdLogger is the interface which allows us to use Logger with Print methods.
// To use this Logger as StdLogger just type cast the Logger to StdLogger.
type StdLogger interface {
	Print(ctx context.Context, v ...interface{})
	Println(ctx context.Context, v ...interface{})
	Printf(ctx context.Context, f string, v ...interface{})
}

type logger struct {
	entry *logrus.Entry
	depth int
}

// With adds kv pair to log message
func (l logger) With(key string, value interface{}) Logger {
	return logger{entry: l.entry.WithField(key, value)}
}

// WithFields adds map as a kv pairs to log message
func (l logger) WithFields(fields map[string]interface{}) Logger {
	return logger{entry: l.entry.WithFields(fields)}
}

// Debug logs a message at level Debug on the standard logger.
func (l logger) Debug(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Debug(args...)
}

// Debugln logs a message at level Debug on the standard logger.
func (l logger) Debugln(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Debugln(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func (l logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	l.with(ctx, false).sourced().Debugf(format, args...)
}

// Info logs a message at level Info on the standard logger.
func (l logger) Info(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Info(args...)
}

// Infoln logs a message at level Info on the standard logger.
func (l logger) Infoln(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Infoln(args...)
}

// Infof logs a message at level Info on the standard logger.
func (l logger) Infof(ctx context.Context, format string, args ...interface{}) {
	l.with(ctx, false).sourced().Infof(format, args...)
}

// Error logs a message at level Error on the standard logger.
func (l logger) Error(ctx context.Context, args ...interface{}) {
	l.with(ctx, true).sourced().WithField("stack_trace", string(debug.Stack())).Error(args...)
}

// Errorln logs a message at level Error on the standard logger.
func (l logger) Errorln(ctx context.Context, args ...interface{}) {
	l.with(ctx, true).sourced().WithField("stack_trace", string(debug.Stack())).Errorln(args...)
}

// Errorf logs a message at level Error on the standard logger.
func (l logger) Errorf(ctx context.Context, format string, args ...interface{}) {
	l.with(ctx, true).sourced().WithField("stack_trace", string(debug.Stack())).Errorf(format, args...)
}

// Print logs a message at level Debug on the standard logger.
func (l logger) Print(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Debug(args...)
}

// Println logs a message at level Debug on the standard logger.
func (l logger) Println(ctx context.Context, args ...interface{}) {
	l.with(ctx, false).sourced().Debugln(args...)
}

// Printf logs a message at level Debug on the standard logger.
func (l logger) Printf(ctx context.Context, format string, args ...interface{}) {
	l.with(ctx, false).sourced().Debugf(format, args...)
}

// Fatal logs a message at level Error on the standard logger and exits.
func (l logger) Fatal(ctx context.Context, args ...interface{}) {
	l.depth++
	l.Error(ctx, args...)
	exit(1)
}

// Fatalln logs a message at level Error on the standard logger.
func (l logger) Fatalln(ctx context.Context, args ...interface{}) {
	l.depth++
	l.Errorln(ctx, args...)
	exit(1)
}

// Fatalf logs a message at level Error on the standard logger and exits.
func (l logger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	l.depth++
	l.Errorf(ctx, format, args...)
	exit(1)
}

// sourced adds a source field to the logger that contains
// the file name and line where the logging happened.
func (l logger) sourced() *logrus.Entry {
	_, file, line, ok := runtime.Caller(l.depth + 2)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	return l.entry.WithField(loggerFieldKey, fmt.Sprintf("%s:%d", file, line))
}

// IncDepth can be used by wrappers to increment stack depth.
func (l logger) IncDepth(depth int) Logger {
	l.depth += depth
	return l
}

// PrivacyDataFormatter formats the given sensitive string.
func PrivacyDataFormatter(sensitiveData string) string {
	return fmt.Sprintf("[_priv_]%s[/_priv_]", sensitiveData)
}

// NewLogger returns a new Logger logging to stderr.
//
// Logger configuration is done in a way that it complies
// with Neo logging standards, configuration can be changed with
// environment variables as follows:
//
//	Variable            | Values
//	-----------------------------------------------------------
//	LOGGING_LEVEL       | 'debug', 'info' (default), 'error'
//	LOGGING_FORMAT      | 'json' (default), 'txt'
//
// If invalid configuration is given NewLogger will return Logger
// with default configuration and handle error by logging it.
// Log events contains following fields by default:
//
//	timestamp
//	message
//	logger
//	level
//	stack_trace (only in 'error' level)
//
// # Log metrics
//
// Logger will automatically collect metrics (log event counters) for Prometheus.
// Metrics will be exposed only if you run metrics.ManagementServer in your application.
func NewLogger() Logger {
	level, format, err := parseConfig()
	l := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: format,
		Hooks:     make(logrus.LevelHooks),
		Level:     level,
	}
	l.Hooks.Add(logging.GetMetricsHook())
	neoLogger := logger{entry: logrus.NewEntry(l)}

	// Handle error by logging it and allow application to continue with default logger configuration
	if err != nil {
		neoLogger.Errorf(context.Background(), "Error parsing logger config: %s", err)
	}
	return neoLogger
}

func parseConfig() (logLevel logrus.Level, outputFormat logrus.Formatter, err error) {
	// Set default settings
	logLevel = logrus.InfoLevel
	outputFormat = &logrus.JSONFormatter{}

	level := os.Getenv("LOGGING_LEVEL")

	switch strings.ToLower(level) {
	case "debug":
		logLevel = logrus.DebugLevel
	case "info", "": // default
		logLevel = logrus.InfoLevel
	case "error":
		logLevel = logrus.ErrorLevel
	default:
		err = fmt.Errorf("invalid LOGGING_LEVEL '%s', please specify LOGGING_LEVEL as 'debug', 'info' or 'error'", level)
		return
	}
	format := os.Getenv("LOGGING_FORMAT")
	switch format {
	case "json", "": // default
		outputFormat = &logrus.JSONFormatter{
			TimestampFormat: ISO8601,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyLevel: "level",
			},
		}
	case "txt":
		outputFormat = &logrus.TextFormatter{}
	default:
		err = fmt.Errorf("invalid LOGGING_FORMAT '%s' Please specify LOGGING_FORMAT as 'json' or 'txt'", format)
		return
	}
	return logLevel, outputFormat, err
}

func (l logger) with(context context.Context, isError bool) logger {
	span := opentracing.SpanFromContext(context)
	if span == nil {
		return l
	}

	if isError {
		span.SetTag("error", true)
	}

	ctx, isOfTypeJaegerSpanContext := span.Context().(jaeger.SpanContext)
	if !isOfTypeJaegerSpanContext {
		return l
	}

	l.entry = l.entry.
		WithField("trace_id", ctx.TraceID().String()).
		WithField("span_id", ctx.SpanID().String()).
		WithField("parent_id", ctx.ParentID().String()).
		WithField("is_sampled", fmt.Sprintf("%v", ctx.IsSampled()))
	return l
}
