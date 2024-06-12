package exporter

import (
	"database/sql"
	"testing"

	"github.com/phanitejak/kptgolib/logging"
)

var log = logging.NewLogger()

func TestMinioReplicationMetericExporter_PublishMetric(t *testing.T) {
	metricHelper := New(log, "dummy_service", "service_user")
	tests := []struct {
		name    string
		cme     *DBMetricExporter
		wantErr bool
	}{
		{"Publish Metrics", metricHelper, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.cme.PublishMetric(sql.DBStats{})
			tt.cme.Unregister()
		})
	}
}

func TestMinioReplicationMetericExporter_ResetMetric(t *testing.T) {
	cme := New(log, "dummy_service", "service_user")
	cme.ResetMetric()
	cme.Unregister()
}
