package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Summary is an interface for summary metrics
type Summary interface {
	GetCollector() prometheus.Collector
	Observe(f float64)
	ObserveDuration(startTime time.Time)
	Unregister() bool
}

// CustomSummary is type for business logic specific 1-dimension summary metrics.
type CustomSummary struct {
	observer  prometheus.Observer
	collector prometheus.Collector
}

// GetCollector get the summary
func (cs *CustomSummary) GetCollector() prometheus.Collector {
	return cs.collector
}

// GetCollector get the summaryVec
func (csv *CustomSummaryVec) GetCollector() prometheus.Collector {
	return csv.summaryVec
}

// Observe observers the given value.
func (cs *CustomSummary) Observe(f float64) { cs.observer.Observe(f) }

// ObserveDuration observers the elapsed time since given time in milliseconds.
func (cs *CustomSummary) ObserveDuration(startTime time.Time) {
	cs.observer.Observe(float64(time.Since(startTime)) / float64(time.Millisecond))
}

// Unregister unregisters the summary
func (cs *CustomSummary) Unregister() bool {
	return prometheus.Unregister(cs.collector)
}

// CustomSummaryVec is type for business logic specific 2-n dimension summary
// metrics (1-n custom labels).
type CustomSummaryVec struct {
	summaryVec *prometheus.SummaryVec
	metricName string
}

// GetCustomSummary gets custom summary for given labels. Labels has to be given
// in the same order than registered.
func (csv *CustomSummaryVec) GetCustomSummary(labelValues ...string) Summary {
	finalLabelValues := append(labelValues, csv.metricName)
	return &CustomSummary{csv.summaryVec.WithLabelValues(finalLabelValues...), csv.summaryVec}
}

// DeleteSerie deletes custom summary for given labels. Labels has to be given
// in the same order than registered.
func (csv *CustomSummaryVec) DeleteSerie(labelValues ...string) bool {
	finalLabelValues := append(labelValues, csv.metricName)
	return csv.summaryVec.DeleteLabelValues(finalLabelValues...)
}

// Reset deletes all metrics in this summary vector.
func (csv *CustomSummaryVec) Reset() {
	csv.summaryVec.Reset()
}

// Unregister unregisters the summaryVec.
func (csv *CustomSummaryVec) Unregister() bool {
	return prometheus.Unregister(csv.summaryVec)
}

// RegisterSummary registers given summary metric by using given subsystem name
// and metric description. NEO metrics namespace is added to metric name as
// prefix.
func RegisterSummary(metricName string, subsystem string, desc string) Summary {
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	})

	return registerSummaryMetric(summary)
}

// RegisterSummaryWithObjectives registers given summary metric by using given subsystem name
// , metric description and the quantile rank. NEO metrics namespace is added to metric name as
// prefix. It gives option to configure quantities.
func RegisterSummaryWithObjectives(metricName string, subsystem string, desc string, objectives map[float64]float64) Summary {
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  metricNamespace,
		Subsystem:  subsystem,
		Name:       metricName,
		Help:       desc,
		Objectives: objectives,
	})
	return registerSummaryMetric(summary)
}

func registerSummaryMetric(summary prometheus.Summary) Summary {
	prometheus.MustRegister(summary)
	return &CustomSummary{summary, summary}
}

// RegisterSummaryVec registers given summary vector metric by using given keys,
// subsystem name and metric description. NEO metrics namespace is added to
// metric name as prefix.
func RegisterSummaryVec(metricName string, subsystem string, desc string, keys ...string) *CustomSummaryVec {
	finalKeys := append(keys, plainMetricNameKey)
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	}, finalKeys)
	prometheus.MustRegister(summaryVec)
	return &CustomSummaryVec{summaryVec, metricName}
}
