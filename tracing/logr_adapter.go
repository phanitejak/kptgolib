package tracing

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/phanitejak/kptgolib/logging"
	loggingv2 "github.com/phanitejak/kptgolib/logging/v2"
)

// logrLoggerAdapter implements logr.LogSink interface for logging.Logger implementation.
// Used to hook NOM logger into OpenTelemetry internal logging.
type logrLoggerAdapter struct {
	logger logging.Logger
}

// Init initializes the logger
func (l logrLoggerAdapter) Init(_ logr.RuntimeInfo) {
}

func (l logrLoggerAdapter) Enabled(level int) bool {
	return level < 1 // Log only error messages
}

func (l logrLoggerAdapter) WithName(_ string) logr.LogSink {
	return l
}

// Error logs error
func (l logrLoggerAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrLoggerAdapter).logger.Error(context.Background(), fmt.Sprintf("%s: %s", msg, err))
}

// Info logs info
func (l logrLoggerAdapter) Info(_ int, msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrLoggerAdapter).logger.Info(msg)
}

func (l logrLoggerAdapter) WithValues(keysAndValues ...interface{}) logr.LogSink {
	key := ""
	var value interface{}
	for i, kv := range keysAndValues {
		if i%2 == 0 {
			_, ok := kv.(string)
			if !ok {
				return l
			}
			key = kv.(string)
		} else {
			value = kv
			v, ok := value.(interface{ MarshalLog() interface{} })
			if ok {
				value = v.MarshalLog()
			}
			l.logger = l.logger.With(key, value)
		}
	}
	return l
}

// logrV2LoggerAdapter implements logr.LogSink interface for loggingv2.Logger implementation.
// Used to hook NOM logger into OpenTelemetry internal logging.
type logrV2LoggerAdapter struct {
	logger loggingv2.Logger
}

// Init initializes the logger
func (l logrV2LoggerAdapter) Init(_ logr.RuntimeInfo) {
}

func (l logrV2LoggerAdapter) Enabled(level int) bool {
	return level < 1 // Log only error messages
}

func (l logrV2LoggerAdapter) WithName(_ string) logr.LogSink {
	return l
}

// Error logs error
func (l logrV2LoggerAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrV2LoggerAdapter).logger.Error(context.Background(), fmt.Sprintf("%s: %s", msg, err))
}

// Info logs info
func (l logrV2LoggerAdapter) Info(_ int, msg string, keysAndValues ...interface{}) {
	l.WithValues(keysAndValues...).(logrV2LoggerAdapter).logger.Info(context.Background(), msg)
}

func (l logrV2LoggerAdapter) WithValues(keysAndValues ...interface{}) logr.LogSink {
	key := ""
	var value interface{}
	for i, kv := range keysAndValues {
		if i%2 == 0 {
			_, ok := kv.(string)
			if !ok {
				return l
			}
			key = kv.(string)
		} else {
			value = kv
			v, ok := value.(interface{ MarshalLog() interface{} })
			if ok {
				value = v.MarshalLog()
			}
			l.logger = l.logger.With(key, value)
		}
	}
	return l
}
