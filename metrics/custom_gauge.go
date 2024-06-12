package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// CustomGauge is type for business logic specific 1-dimension gauge metrics.
type CustomGauge struct {
	gauge prometheus.Gauge
}

// GetCollector get the gauge
func (cg *CustomGauge) GetCollector() prometheus.Collector {
	return cg.gauge
}

// GetCollector get the gaugeVec
func (cgv *CustomGaugeVec) GetCollector() prometheus.Collector {
	return cgv.gaugeVec
}

// Set sets the gauge to the given value.
func (cg *CustomGauge) Set(f float64) { cg.gauge.Set(f) }

// Add adds the given value to the gauge value.
func (cg *CustomGauge) Add(f float64) { cg.gauge.Add(f) }

// Sub subtracts the given value from the gauge value.
func (cg *CustomGauge) Sub(f float64) { cg.gauge.Sub(f) }

// Unregister unregisters the gauge
func (cg *CustomGauge) Unregister() bool {
	return prometheus.Unregister(cg.gauge)
}

// CustomGaugeVec is type for business logic specific 2-n dimension gauge
// metrics (1-n custom labels).
type CustomGaugeVec struct {
	gaugeVec   *prometheus.GaugeVec
	metricName string
}

// GetCustomGauge gets custom gauge for given labels. Labels has to be given
// in the same order than registered.
func (cgv *CustomGaugeVec) GetCustomGauge(labelValues ...string) *CustomGauge {
	finalLabelValues := append(labelValues, cgv.metricName)
	return &CustomGauge{cgv.gaugeVec.WithLabelValues(finalLabelValues...)}
}

// DeleteSerie deletes custom gauge for given labels. Labels has to be given
// in the same order than registered.
func (cgv *CustomGaugeVec) DeleteSerie(labelValues ...string) bool {
	finalLabelValues := append(labelValues, cgv.metricName)
	return cgv.gaugeVec.DeleteLabelValues(finalLabelValues...)
}

// Reset deletes all metrics in this gauge vector.
func (cgv *CustomGaugeVec) Reset() {
	cgv.gaugeVec.Reset()
}

// Unregister unregisters the gaugeVec.
func (cgv *CustomGaugeVec) Unregister() bool {
	return prometheus.Unregister(cgv.gaugeVec)
}

// RegisterGauge registers given gauge metric by using given subsystem name
// and metric description. NEO metrics namespace is added to metric name as
// prefix.
func RegisterGauge(metricName string, subsystem string,
	desc string) *CustomGauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	})
	prometheus.MustRegister(gauge)
	return &CustomGauge{gauge}
}

// RegisterGaugeVec registers given gauge vector metric by using given keys,
// subsystem name and metric description. NEO metrics namespace is added to
// metric name as prefix.
func RegisterGaugeVec(metricName string, subsystem string, desc string,
	keys ...string) *CustomGaugeVec {
	finalKeys := append(keys, plainMetricNameKey)
	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Subsystem: subsystem,
		Name:      metricName,
		Help:      desc,
	}, finalKeys)
	prometheus.MustRegister(gaugeVec)
	return &CustomGaugeVec{gaugeVec, metricName}
}
