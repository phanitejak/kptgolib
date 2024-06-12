// Package exporter MariaDB statistics metrics publisher
package exporter

import (
	"database/sql"

	"github.com/phanitejak/kptgolib/gerror"
	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/metrics"
)

const (
	// PemDecodeError ...
	PemDecodeError gerror.ErrorCode = "PEM Decode Error"

	key = "session"
)

// DBMetricExporter ...
type DBMetricExporter struct {
	sessinName          string
	log                 logging.Logger
	maxOpenConntections *metrics.CustomGaugeVec
	openConnections     *metrics.CustomGaugeVec
	inUse               *metrics.CustomGaugeVec
	idle                *metrics.CustomGaugeVec
	waitCount           *metrics.CustomGaugeVec
	waitDuration        *metrics.CustomGaugeVec
	maxIdleClosed       *metrics.CustomGaugeVec
	maxIdleTimeClosed   *metrics.CustomGaugeVec
	maxLifetimeClosed   *metrics.CustomGaugeVec
}

// New creates an instance of Exporter and returns it
func New(log logging.Logger, serviceName, sessionName string) *DBMetricExporter {
	session := sessionName
	if sessionName == "" {
		session = "service"
	}
	session = "_" + session
	maxOpenConntections := metrics.RegisterGaugeVec("maxOpenConntections"+session, serviceName, "Maximum Open connections", key)
	openConnections := metrics.RegisterGaugeVec("openConnections"+session, serviceName, "Number of open connections", key)
	inUse := metrics.RegisterGaugeVec("inUse"+session, serviceName, "Number of in use connections", key)
	idle := metrics.RegisterGaugeVec("idle"+session, serviceName, "Number of idle connections", key)
	waitCount := metrics.RegisterGaugeVec("waitCount"+session, serviceName, "Total number of connections waited for", key)
	waitDuration := metrics.RegisterGaugeVec("waitDuration"+session, serviceName, "Total time blocked waiting for a new connection", key)
	maxIdleClosed := metrics.RegisterGaugeVec("maxIdleClosed"+session, serviceName, "Total number of connections closed due to SetMaxIdleConns", key)
	maxIdleTimeClosed := metrics.RegisterGaugeVec("maxIdleTimeClosed"+session, serviceName, "Total number of connections closed due to SetConnMaxIdleTime", key)
	maxLifetimeClosed := metrics.RegisterGaugeVec("maxLifetimeClosed"+session, serviceName, "Total number of connections closed due to SetConnMaxLifetime", key)

	return &DBMetricExporter{
		sessinName:          sessionName,
		log:                 log,
		maxOpenConntections: maxOpenConntections,
		openConnections:     openConnections,
		inUse:               inUse,
		idle:                idle,
		waitCount:           waitCount,
		waitDuration:        waitDuration,
		maxIdleClosed:       maxIdleClosed,
		maxIdleTimeClosed:   maxIdleTimeClosed,
		maxLifetimeClosed:   maxLifetimeClosed,
	}
}

// PublishMetric ...
func (cme *DBMetricExporter) PublishMetric(stats sql.DBStats) {
	cme.maxOpenConntections.GetCustomGauge(cme.sessinName).Set(float64(stats.MaxOpenConnections))
	cme.openConnections.GetCustomGauge(cme.sessinName).Set(float64(stats.OpenConnections))
	cme.inUse.GetCustomGauge(cme.sessinName).Set(float64(stats.InUse))
	cme.idle.GetCustomGauge(cme.sessinName).Set(float64(stats.Idle))
	cme.waitCount.GetCustomGauge(cme.sessinName).Set(float64(stats.WaitCount))
	cme.waitDuration.GetCustomGauge(cme.sessinName).Set(float64(stats.WaitDuration))
	cme.maxIdleClosed.GetCustomGauge(cme.sessinName).Set(float64(stats.MaxIdleClosed))
	cme.maxIdleTimeClosed.GetCustomGauge(cme.sessinName).Set(float64(stats.MaxIdleTimeClosed))
	cme.maxLifetimeClosed.GetCustomGauge(cme.sessinName).Set(float64(stats.MaxLifetimeClosed))
}

// ResetMetric ...
func (cme *DBMetricExporter) ResetMetric() {
	cme.maxOpenConntections.Reset()
	cme.openConnections.Reset()
	cme.inUse.Reset()
	cme.idle.Reset()
	cme.waitCount.Reset()
	cme.waitDuration.Reset()
	cme.maxIdleClosed.Reset()
	cme.maxIdleTimeClosed.Reset()
	cme.maxLifetimeClosed.Reset()
}

// Unregister ...
func (cme *DBMetricExporter) Unregister() {
	_ = cme.maxOpenConntections.Unregister()
	_ = cme.openConnections.Unregister()
	_ = cme.inUse.Unregister()
	_ = cme.idle.Unregister()
	_ = cme.waitCount.Unregister()
	_ = cme.waitDuration.Unregister()
	_ = cme.maxIdleClosed.Unregister()
	_ = cme.maxIdleTimeClosed.Unregister()
	_ = cme.maxLifetimeClosed.Unregister()
}
