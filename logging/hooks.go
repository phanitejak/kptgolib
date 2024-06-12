package logging

import (
	"github.com/phanitejak/kptgolib/metrics"
	"github.com/sirupsen/logrus"
)

// MetricsHook exposes Prometheus counters for each of logrus' log levels.
type MetricsHook struct {
	counterVec metrics.CounterVec
}

var (
	supportedLevels = []logrus.Level{logrus.DebugLevel, logrus.InfoLevel, logrus.ErrorLevel}
	hook            = &MetricsHook{
		counterVec: metrics.RegisterCounterVec("events_total", "logger", "Total number of log messages.", "level"),
	}
)

var auditHook = &MetricsHook{
	counterVec: metrics.RegisterCounterVec("audit_events_total", "logger", "Total number of audit log messages.", "level"),
}

//nolint:gochecknoinits
func init() {
	// Initialise counters for all supported levels:
	for _, level := range supportedLevels {
		hook.counterVec.GetCustomCounter(level.String())
	}
}

// Fire increments the appropriate Prometheus counter depending on the entry's log level.
func (h *MetricsHook) Fire(entry *logrus.Entry) error {
	h.counterVec.GetCustomCounter(entry.Level.String()).Inc()
	return nil
}

// Levels returns all supported log levels.
func (h *MetricsHook) Levels() []logrus.Level {
	return supportedLevels
}

// GetMetricsHook retrieves a logging hook for metrics, counting number of log entries
func GetMetricsHook() *MetricsHook {
	return hook
}
