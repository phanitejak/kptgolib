package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Counter is an interface for metrics counters
type Counter interface {
	Add(i int64)
	Inc()
	Unregister() bool
	GetCollector() prometheus.Collector
}

// CustomCounter is type for business logic specific 1-dimension counter metrics
// (no custom labels).
type CustomCounter struct {
	counter prometheus.Counter
}

// GetCollector get the counter
func (cc *CustomCounter) GetCollector() prometheus.Collector {
	return cc.counter
}

// GetCollector get the counterVec
func (ccv *CustomCounterVec) GetCollector() prometheus.Collector {
	return ccv.counterVec
}

// Add adds the given value to the counter value.
func (cc *CustomCounter) Add(i int64) { cc.counter.Add(float64(i)) }

// Inc increments counter value by 1.
func (cc *CustomCounter) Inc() { cc.counter.Inc() }

// Unregister unregisters the counter
func (cc *CustomCounter) Unregister() bool {
	return prometheus.Unregister(cc.counter)
}

// CounterVec is an interface for metrics vec counters
type CounterVec interface {
	GetCustomCounter(labelValues ...string) Counter
	DeleteSerie(labelValues ...string) bool
	Reset()
	Unregister() bool
	GetCollector() prometheus.Collector
}

// CustomCounterVec is type for business logic specific 2-n dimension counter
// metrics (1-n custom labels).
type CustomCounterVec struct {
	counterVec *prometheus.CounterVec
	metricName string
}

// GetCustomCounter gets custom counter for given labels. Labels has to be given
// in the same order than registered.
func (ccv *CustomCounterVec) GetCustomCounter(labelValues ...string) Counter {
	finalLabelValues := append(labelValues, ccv.metricName)
	return &CustomCounter{ccv.counterVec.WithLabelValues(finalLabelValues...)}
}

// DeleteSerie deletes custom counter for given labels. Labels has to be given
// in the same order than registered.
func (ccv *CustomCounterVec) DeleteSerie(labelValues ...string) bool {
	finalLabelValues := append(labelValues, ccv.metricName)
	return ccv.counterVec.DeleteLabelValues(finalLabelValues...)
}

// Reset deletes all metrics in this counter vector.
func (ccv *CustomCounterVec) Reset() {
	ccv.counterVec.Reset()
}

// Unregister unregisters the counterVec.
func (ccv *CustomCounterVec) Unregister() bool {
	return prometheus.Unregister(ccv.counterVec)
}

// RegisterCounter registers given counter metric by using given subsystem name
// and metric description. NEO metrics namespace is added to metric name as
// prefix.
func RegisterCounter(metricName string, subsystem string, desc string) Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	})
	prometheus.MustRegister(counter)
	return &CustomCounter{counter}
}

// RegisterCounterVec registers given counter vector metric by using given
// keys, subsystem name and metric description. NEO metrics namespace is
// added to metric name as prefix.
func RegisterCounterVec(metricName string, subsystem string, desc string, keys ...string) CounterVec {
	finalKeys := append(keys, plainMetricNameKey)
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	}, finalKeys)
	prometheus.MustRegister(counterVec)
	return &CustomCounterVec{counterVec, metricName}
}
