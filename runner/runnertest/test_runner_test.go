package runnertest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg/runner"
	"gopkg/runner/runnertest"
)

func TestRunAndCleanup(t *testing.T) {
	mod := &TestModule{done: make(chan struct{})}
	a := &TestApp{
		mods: []runner.Module{
			runner.CloserAsModule(mod),
		},
	}

	// Close done channel so mod.Close can return.
	close(mod.done)
	// RunAndCleanup will assert our exit code for us.
	runnertest.RunAndCleanup(t, a)
}

func TestRun(t *testing.T) {
	mod := &TestModule{done: make(chan struct{})}
	a := &TestApp{
		mods: []runner.Module{
			runner.CloserAsModule(mod),
		},
	}

	// Close done channel so mod.Close can return.
	close(mod.done)

	tr := runnertest.Run(t, a)
	tr.Stop()
	code := tr.ExitCode()
	assert.Equal(t, 0, code)
}

type TestApp struct {
	mods []runner.Module
}

func (a *TestApp) Name() string             { return "test-app" }
func (a *TestApp) Modules() []runner.Module { return a.mods }

type TestModule struct {
	done chan struct{}
}

func (m *TestModule) Close() error {
	<-m.done
	return nil
}
