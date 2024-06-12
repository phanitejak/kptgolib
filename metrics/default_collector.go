package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type defaultCollector struct {
	timeZoneDesc    *prometheus.Desc
	numCpusDesc     *prometheus.Desc
	numCgoCallsDesc *prometheus.Desc
}

// Describe returns all descriptions of the collector.
func (c *defaultCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.timeZoneDesc
	ch <- c.numCpusDesc
	ch <- c.numCgoCallsDesc
}

// Collect returns the current state of all metrics of the collector.
func (c *defaultCollector) Collect(ch chan<- prometheus.Metric) {
	now := time.Now()
	zone, timeZoneOffset := now.Zone()
	ch <- prometheus.MustNewConstMetric(
		c.timeZoneDesc,
		prometheus.GaugeValue,
		float64(timeZoneOffset*1000), zone,
	)
	ch <- prometheus.MustNewConstMetric(
		c.numCpusDesc,
		prometheus.GaugeValue,
		float64(runtime.NumCPU()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.numCgoCallsDesc,
		prometheus.GaugeValue,
		float64(runtime.NumCgoCall()),
	)
}

func newDefaultCollector() *defaultCollector {
	return &defaultCollector{
		timeZoneDesc:    prometheus.NewDesc("timezone_offset_milliseconds", "Timezone offset in milliseconds. Zone name abbreviation is stored in zone_name label.", []string{"zone_name"}, nil),
		numCpusDesc:     prometheus.NewDesc("process_cpu_count", "Number of logical CPUs usable by the current process.", nil, nil),
		numCgoCallsDesc: prometheus.NewDesc("process_cgo_calls", "Number of cgo calls made by the current process.", nil, nil),
	}
}
