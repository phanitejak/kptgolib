package dbstats

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/phanitejak/gopkg/logging"
	"github.com/phanitejak/gopkg/tracing"
)

func TestNewMinioReplicationMetricJob_Run(t *testing.T) {
	log := tracing.NewLogger(logging.NewLogger())
	store := &mockRedundancySyncStore{}
	metricsJob := NewDBMetricJob(log, "dummy_service", "service_user", 10, store)
	assert.Equal(t, "@every 10s", metricsJob.CronSpec())
	metricsJob.Run()
}

type mockRedundancySyncStore struct{}

// Stats to update specific column alarmTime.
func (s *mockRedundancySyncStore) Stats() sql.DBStats {
	return sql.DBStats{}
}
