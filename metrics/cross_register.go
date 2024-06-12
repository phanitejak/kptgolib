package metrics

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	gometrics "github.com/rcrowley/go-metrics"
)

var (
	configs = make(map[string]*PrometheusConfig)
	mutex   sync.Mutex
)

// KafkaProducerPrefix is a common prefix for Kafka producer metrics.
const KafkaProducerPrefix = "kafka_producer"

// KafkaConsumerPrefix is a common prefix for Kafka consumer metrics.
const KafkaConsumerPrefix = "kafka_consumer"

// PrometheusConfig provides a container with config parameters for the
// Prometheus Exporter

type PrometheusConfig struct {
	namespace     string
	Registry      gometrics.Registry // Registry to be exported
	subsystem     string
	promRegistry  prometheus.Registerer // Prometheus registry
	FlushInterval time.Duration         // interval to update prom metrics
	gauges        map[string]prometheus.Gauge
	ticker        *time.Ticker
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewPrometheusProvider(r gometrics.Registry, namespace string, subsystem string, promRegistry prometheus.Registerer, flushInterval time.Duration) *PrometheusConfig {
	return &PrometheusConfig{
		namespace:     namespace,
		subsystem:     subsystem,
		Registry:      r,
		promRegistry:  promRegistry,
		FlushInterval: flushInterval,
		gauges:        make(map[string]prometheus.Gauge),
		ticker:        time.NewTicker(flushInterval),
	}
}

func (c *PrometheusConfig) flattenKey(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)
	return key
}

func (c *PrometheusConfig) gaugeFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	g, ok := c.gauges[key]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		})
		c.promRegistry.MustRegister(g)
		c.gauges[key] = g
	}
	g.Set(val)
}

func (c *PrometheusConfig) UpdatePrometheusMetrics() {
	for range c.ticker.C {
		c.UpdatePrometheusMetricsOnce()
	}
}

func (c *PrometheusConfig) UnregisterPrometheusMetrics() {
	c.ticker.Stop()
	for _, gauge := range c.gauges {
		c.promRegistry.Unregister(gauge)
	}
}

func (c *PrometheusConfig) UpdatePrometheusMetricsOnce() {
	mutex.Lock()
	defer mutex.Unlock()
	c.Registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case gometrics.Counter:
			c.gaugeFromNameAndValue(name, float64(metric.Count()))
		case gometrics.Gauge:
			c.gaugeFromNameAndValue(name, float64(metric.Value()))
		case gometrics.GaugeFloat64:
			c.gaugeFromNameAndValue(name, metric.Value())
		case gometrics.Histogram:
			samples := metric.Snapshot().Sample().Values()
			if len(samples) > 0 {
				lastSample := samples[len(samples)-1]
				c.gaugeFromNameAndValue(name, float64(lastSample))
			}
		case gometrics.Meter:
			lastSample := metric.Snapshot().Rate1()
			c.gaugeFromNameAndValue(name, lastSample)
		case gometrics.Timer:
			lastSample := metric.Snapshot().Rate1()
			c.gaugeFromNameAndValue(name, lastSample)
		}
	})
}

// MustCrossRegisterMetrics registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library as-is. E.g. Sarama Kafka (https://github.com/IBM/sarama) metrics
// are stored in the go-metrics registry. Use this function only in case you
// are cross-registering only one metrics registry. Otherwise use function
// that defines unique metrics prefix to metric names collision.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
func MustCrossRegisterMetrics(goMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix("", goMetricsRegistry)
}

// MustCrossRegisterKafkaConsumerMetrics registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
func MustCrossRegisterKafkaConsumerMetrics(kafkaConsumerGoMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix(KafkaConsumerPrefix, kafkaConsumerGoMetricsRegistry)
}

// MustCrossRegisterKafkaProducerMetrics registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
func MustCrossRegisterKafkaProducerMetrics(kafkaProducerGoMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix(KafkaProducerPrefix, kafkaProducerGoMetricsRegistry)
}

// MustCrossRegisterKafkaConsumerMetricsPrefix registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterKafkaConsumerMetricsPrefix. In case you want to handle error, use function CrossRegisterKafkaConsumerMetricsPrefix.
func MustCrossRegisterKafkaConsumerMetricsPrefix(kafkaConsumerGoMetricsRegistry gometrics.Registry, prefixPostfix string) {
	MustCrossRegisterMetricsWithPrefix(KafkaConsumerPrefix+"_"+prefixPostfix, kafkaConsumerGoMetricsRegistry)
}

// MustCrossRegisterKafkaProducerMetricsPrefix registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
func MustCrossRegisterKafkaProducerMetricsPrefix(kafkaProducerGoMetricsRegistry gometrics.Registry, prefixPostfix string) {
	MustCrossRegisterMetricsWithPrefix(KafkaProducerPrefix+"_"+prefixPostfix, kafkaProducerGoMetricsRegistry)
}

// CrossRegisterMetrics registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library as-is. E.g. Sarama Kafka (https://github.com/IBM/sarama) metrics
// are stored in the go-metrics registry. Use this function only in case you
// are cross-registering only one metrics registry. Otherwise use function
// that defines unique metrics prefix to metric names collision.
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterMetrics(goMetricsRegistry gometrics.Registry) error {
	return CrossRegisterMetricsWithPrefix("", goMetricsRegistry)
}

// CrossRegisterKafkaConsumerMetrics registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer.
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterKafkaConsumerMetrics(kafkaConsumerGoMetricsRegistry gometrics.Registry) error {
	return CrossRegisterMetricsWithPrefix(KafkaConsumerPrefix, kafkaConsumerGoMetricsRegistry)
}

// CrossRegisterKafkaProducerMetrics registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer.
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterKafkaProducerMetrics(kafkaProducerGoMetricsRegistry gometrics.Registry) error {
	return CrossRegisterMetricsWithPrefix(KafkaProducerPrefix, kafkaProducerGoMetricsRegistry)
}

// CrossRegisterKafkaConsumerMetricsPrefix registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterKafkaConsumerMetricsPrefix(kafkaConsumerGoMetricsRegistry gometrics.Registry, prefixPostfix string) error {
	return CrossRegisterMetricsWithPrefix(KafkaConsumerPrefix+"_"+prefixPostfix, kafkaConsumerGoMetricsRegistry)
}

// CrossRegisterKafkaProducerMetricsPrefix registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterKafkaProducerMetricsPrefix(kafkaProducerGoMetricsRegistry gometrics.Registry, prefixPostfix string) error {
	return CrossRegisterMetricsWithPrefix(KafkaProducerPrefix+"_"+prefixPostfix, kafkaProducerGoMetricsRegistry)
}

// CrossRegisterGoMetrics registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library as-is. E.g. Sarama Kafka (https://github.com/IBM/sarama) metrics
// are stored in the go-metrics registry. Use this function only in case you
// are cross-registering only one metrics registry. Otherwise use function
// that defines unique metrics prefix to metric names collision.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterMetrics. In case you want to handle error, use function CrossRegisterMetrics.
func CrossRegisterGoMetrics(goMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix("", goMetricsRegistry)
}

// CrossRegisterKafkaConsumerGoMetrics registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// CrossRegisterKafkaConsumerMetrics. In case you want to handle error, use function CrossRegisterKafkaConsumerMetrics.
func CrossRegisterKafkaConsumerGoMetrics(kafkaConsumerGoMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix(KafkaConsumerPrefix, kafkaConsumerGoMetricsRegistry)
}

// CrossRegisterKafkaProducerGoMetrics registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterKafkaProducerMetrics. In case you want to handle error, use function CrossRegisterKafkaProducerMetrics.
func CrossRegisterKafkaProducerGoMetrics(kafkaProducerGoMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix(KafkaProducerPrefix, kafkaProducerGoMetricsRegistry)
}

// CrossRegisterKafkaConsumerGoMetricsPrefix registers given Kafka consumer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_consumer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterKafkaConsumerMetricsPrefix. In case you want to handle error, use function CrossRegisterKafkaConsumerMetricsPrefix.
func CrossRegisterKafkaConsumerGoMetricsPrefix(kafkaConsumerGoMetricsRegistry gometrics.Registry, prefixPostfix string) {
	MustCrossRegisterMetricsWithPrefix(KafkaConsumerPrefix+"_"+prefixPostfix, kafkaConsumerGoMetricsRegistry)
}

// CrossRegisterKafkaProducerGoMetricsPrefix registers given Kafka producer go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library using prefix kafka_producer_<prefixPostfix>.
// In case cross registered metrics uniqueness cannot be guaranteed, panic will happen.
//
// Deprecated: In case you want panic happen when  cross registered metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterKafkaProducerMetricsPrefix. In case you want to handle error, use function CrossRegisterKafkaProducerMetricsPrefix.
func CrossRegisterKafkaProducerGoMetricsPrefix(kafkaProducerGoMetricsRegistry gometrics.Registry, prefixPostfix string) {
	MustCrossRegisterMetricsWithPrefix(KafkaProducerPrefix+"_"+prefixPostfix, kafkaProducerGoMetricsRegistry)
}

// CrossRegisterGoMetricsWithPrefix registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library by adding given prefix for metric name. E.g. Sarama Kafka
// (https://github.com/IBM/sarama) metrics are stored in the go-metrics
// registry.
// Prefix must be unique and not match between already existing prefixes.
// In case cross registered metrics uniqueness cannot be guaranteed, panic is happen.
//
// Deprecated: In case you want panic happen when metrics uniqueness cannot be guaranteed, use
// MustCrossRegisterMetricsWithPrefix. In case you want to handle error, use function
// CrossRegisterMetricsWithPrefix.
func CrossRegisterGoMetricsWithPrefix(prefix string, goMetricsRegistry gometrics.Registry) {
	MustCrossRegisterMetricsWithPrefix(prefix, goMetricsRegistry)
}

// CrossRegisterGoMetricsWithPrefix registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library by adding given prefix for metric name. E.g. Sarama Kafka
// (https://github.com/IBM/sarama) metrics are stored in the go-metrics
// registry
// Prefix must be unique and not match between already existing prefixes
// In case cross registered metrics uniqueness cannot be guaranteed, an error is returned.
func CrossRegisterMetricsWithPrefix(prefix string, goMetricsRegistry gometrics.Registry) error {
	mutex.Lock()
	defer mutex.Unlock()
	if isAlreadyDefined(prefix) {
		return fmt.Errorf("prefix '%s' is matching to already existing prefix or already existing prefix is matching it! Use different prefix", prefix)
	}
	pClient := NewPrometheusProvider(goMetricsRegistry, prefix,
		"", prometheus.DefaultRegisterer, 1*time.Second)
	defer appendConfig(prefix, pClient)
	go pClient.UpdatePrometheusMetrics()
	return nil
}

// MustCrossRegisterGoMetricsWithPrefix registers given go-metrics
// (https://github.com/rcrowley/go-metrics) registry metrics to NEO metrics
// library by adding given prefix for metric name. E.g. Sarama Kafka
// (https://github.com/IBM/sarama) metrics are stored in the go-metrics
// registry.
// Prefix must be unique and not match between already existing prefixes.
// In case cross registered metrics uniqueness cannot be guaranteed, panic is happen.
func MustCrossRegisterMetricsWithPrefix(prefix string, goMetricsRegistry gometrics.Registry) {
	mutex.Lock()
	defer mutex.Unlock()
	if isAlreadyDefined(prefix) {
		panic(fmt.Sprintf("Prefix '%s' is matching to already existing prefix or already existing prefix is matching it! Use different prefix!", prefix))
	}
	pClient := NewPrometheusProvider(goMetricsRegistry, prefix,
		"", prometheus.DefaultRegisterer, 1*time.Second)
	defer appendConfig(prefix, pClient)
	go pClient.UpdatePrometheusMetrics()
}

// UnregisterMetrics unregisters all cross-registered metrics from NEO metrics
// registry (prometheus).
func UnregisterMetrics() {
	mutex.Lock()
	defer mutex.Unlock()
	for prefix := range configs {
		if config, ok := configs[prefix]; ok {
			config.UnregisterPrometheusMetrics()
			delete(configs, prefix)
		} else {
			panic(fmt.Sprintf("Prefix '%s' is not registered!", prefix))
		}
	}
}

// UnregisterKafkaConsumerMetrics unregisters given Kafka consumer metrics
// registered by CrossRegisterKafkaConsumerGoMetrics.
func UnregisterKafkaConsumerMetrics() {
	UnregisterMetricsWithPrefix(KafkaConsumerPrefix)
}

// UnregisterKafkaProducerMetrics unregisters given Kafka consumer metrics
// registered by CrossRegisterKafkaProducerGoMetrics.
func UnregisterKafkaProducerMetrics() {
	UnregisterMetricsWithPrefix(KafkaProducerPrefix)
}

// UnregisterKafkaConsumerMetrics unregisters given Kafka consumer metrics
// registered by CrossRegisterKafkaConsumerGoMetricsPrefix.
func UnregisterKafkaConsumerMetricsPrefix(prefixPostfix string) {
	UnregisterMetricsWithPrefix(KafkaConsumerPrefix + "_" + prefixPostfix)
}

// UnregisterKafkaProducerMetrics unregisters given Kafka consumer metrics
// registered by CrossRegisterKafkaProducerGoMetricsPrefix.
func UnregisterKafkaProducerMetricsPrefix(prefixPostfix string) {
	UnregisterMetricsWithPrefix(KafkaProducerPrefix + "_" + prefixPostfix)
}

// UnregisterMetricsWithPrefix unregisters all cross-registered metrics using
// given prefix from NEO metrics registry (prometheus).
func UnregisterMetricsWithPrefix(prefix string) {
	mutex.Lock()
	defer mutex.Unlock()
	if config, ok := configs[prefix]; ok {
		config.UnregisterPrometheusMetrics()
		delete(configs, prefix)
	} else {
		panic(fmt.Sprintf("Prefix '%s' is not registered!", prefix))
	}
}

func isAlreadyDefined(prefix string) bool {
	if prefix == "" {
		return false
	}
	for alreadyDefinedPrefix := range configs {
		if strings.HasPrefix(prefix, alreadyDefinedPrefix) || strings.HasPrefix(alreadyDefinedPrefix, prefix) {
			return true
		}
	}
	return false
}

//nolint:gosec
func appendConfig(key string, config *PrometheusConfig) {
	if key != "" {
		configs[key] = config
	} else {
		configs[strconv.Itoa(rand.Int())] = config
	}
}
