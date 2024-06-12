// Package dbstats MariaDB session statistics publisher poller
package dbstats

import (
	"database/sql"
	"fmt"

	"github.com/phanitejak/gopkg/mariadb/gorm-db-stats/exporter"
	"github.com/phanitejak/gopkg/tracing"
)

// StoreStats ...
type StoreStats interface {
	Stats() sql.DBStats
}

// DBMetricJob ...
type DBMetricJob struct {
	pollingInterval int
	log             *tracing.Logger
	store           StoreStats
	exporter        *exporter.DBMetricExporter
}

// NewDBMetricJob Creates DB metric job.
// sessionName: Unique string to identify DB session. if empty "service" is used.
//
//	If multiple service user sessions are used then have to instantiate multiple metic jobs.
//	Most probable sessions are service or admin
func NewDBMetricJob(log *tracing.Logger,
	serviceName, sessionName string,
	pollingInterval int,
	store StoreStats) *DBMetricJob {
	return &DBMetricJob{
		log:             log,
		pollingInterval: pollingInterval,
		exporter:        exporter.New(log, serviceName, sessionName),
		store:           store,
	}
}

// CronSpec ...
func (m *DBMetricJob) CronSpec() string {
	return fmt.Sprintf("@every %ds", m.pollingInterval)
}

// Run ...
func (m *DBMetricJob) Run() {
	m.log.Debugf("MinioReplicationMetric called")
	if err := m.PublishMetrics(); err != nil {
		m.log.Errorf("MinioReplicationMetric failed with error : %w", err)
	}
}

// PublishMetrics ...
func (m *DBMetricJob) PublishMetrics() error {
	m.exporter.PublishMetric(m.store.Stats())
	return nil
}
