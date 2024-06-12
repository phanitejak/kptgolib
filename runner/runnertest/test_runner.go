// Package runnertest provides utilities to run App easily in tests.
package runnertest

import (
	"context"
	"testing"
	"time"

	"github.com/phanitejak/gopkg/logging/loggingtest"
	"github.com/phanitejak/gopkg/runner"
	"github.com/phanitejak/gopkg/tracing"
	"github.com/stretchr/testify/assert"
)

// RunAndCleanup starts the App and expects it to exit with exit code 0.
// All modules returned by App are expected to close within a 30 seconds.
func RunAndCleanup(t *testing.T, a runner.App) {
	tr := Run(t, a)
	t.Cleanup(func() {
		tr.stop()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		assert.Equal(t, 0, tr.ExitCodeCtx(ctx), "App exited with non zero exit code.")
	})
}

// Run can be used to start App when user needs to have control over exit code handling.
// All other cases RunAndCleanup should be used instead.
func Run(t *testing.T, a runner.App) *TestRunner {
	exitCode := make(chan int)
	ctx, stop := context.WithCancel(context.Background())
	log := tracing.NewLogger(loggingtest.NewTestLogger(t))

	r := runner.NewRunner(ctx, log)
	go func() { exitCode <- r.Run(a) }()
	r.Ready()

	return &TestRunner{
		exitCode: exitCode,
		ctx:      ctx,
		stop:     stop,
		r:        r,
		t:        t,
	}
}

// TestRunner is test utility to easily run Apps in integration tests.
type TestRunner struct {
	exitCode chan int
	ctx      context.Context
	stop     func()
	r        *runner.AppRunner
	t        testing.TB
}

// Stop signals runner to close all modules.
// After Stop() is called user should read the exit code and assert it.
func (tr *TestRunner) Stop() {
	tr.stop()
}

// ExitCode blocks until all modules from App have been closed
// and then returns exit code returned by runner.
func (tr *TestRunner) ExitCode() int {
	return <-tr.exitCode
}

// ExitCodeCtx blocks until all modules from App have been closed
// and then returns exit code returned by runner or given context gets cancelled.
func (tr *TestRunner) ExitCodeCtx(ctx context.Context) int {
	select {
	case code := <-tr.exitCode:
		return code
	case <-ctx.Done():
		tr.t.Error("Context cancelled before receiving App's exit code.")
		return 1
	}
}
