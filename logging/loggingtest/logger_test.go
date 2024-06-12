package loggingtest_test

import (
	"os/exec"
	"testing"

	"github.com/phanitejak/kptgolib/logging/loggingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestLogger(t *testing.T) {
	log := loggingtest.NewTestLogger(t)
	log.Debug("testing")
	log.Debugf("%s", "testing")
	log.Debugln("testing")
	log.Info("testing")
	log.Infof("%s", "testing")
	log.Infoln("testing")
	log.Error("testing")
	log.Errorf("%s", "testing")
	log.Errorln("testing")
	log.Print("testing")
	log.Printf("%s", "testing")
	log.Println("testing")
}

func TestLoggingFatal(t *testing.T) {
	cmd := exec.Command("go", "test", "./testdata")
	out, err := cmd.Output()
	e := &exec.ExitError{}
	require.ErrorAs(t, err, &e)
	assert.Equal(t, 1, e.ExitCode())
	assert.Contains(t, string(out), "foo_test.go:12: error")
	assert.Contains(t, string(out), "foo_test.go:13: fatal")
	assert.Contains(t, string(out), "foo_test.go:18: error")
	assert.Contains(t, string(out), "foo_test.go:19: fatal")
}

func TestLoggerImplementsIncDepth(t *testing.T) {
	log := loggingtest.NewTestLogger(t)
	log.IncDepth(0)
}
