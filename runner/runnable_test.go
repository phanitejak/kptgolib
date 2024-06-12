package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"gopkg/runner"
)

type constError string

func (c constError) Error() string { return string(c) }

const (
	errA   constError = "error A"
	errB   constError = "error B"
	errC   constError = "error C"
	errD   constError = "error D"
	panicA constError = "panic A"
	panicB constError = "panic B"
)

func TestRun_ExitTriggeredFromRunnable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		runnables    []runner.Runnable
		wantedErrors []error
	}{
		{
			name:         "empty runners",
			runnables:    []runner.Runnable{},
			wantedErrors: nil,
		},
		{
			name:         "one runner with failure in Run()",
			runnables:    []runner.Runnable{newClosedRunner(errA, nil)},
			wantedErrors: []error{errA},
		},
		{
			name:         "one runner with failure in Close()",
			runnables:    []runner.Runnable{newClosedRunner(nil, errA)},
			wantedErrors: []error{errA},
		},
		{
			name:         "multiple runners (one runner with failure in Run())",
			runnables:    []runner.Runnable{newBlockingRunner(ctx, nil, nil), newClosedRunner(errA, nil)},
			wantedErrors: []error{errA},
		},
		{
			name:         "multiple runners (one runner with failure in Close())",
			runnables:    []runner.Runnable{newBlockingRunner(ctx, nil, nil), newClosedRunner(nil, errA)},
			wantedErrors: []error{errA},
		},
		{
			name:         "multiple runners (one runner with failures in Run() and Close())",
			runnables:    []runner.Runnable{newBlockingRunner(ctx, nil, nil), newClosedRunner(errA, errB)},
			wantedErrors: []error{errA, errB},
		},
		{
			name:         "multiple failing runners (one runner with failures in Run() and Close())",
			runnables:    []runner.Runnable{newClosedRunner(errA, errB), newClosedRunner(errC, errD)},
			wantedErrors: []error{errA, errB, errC, errD},
		},
		{
			name:         "multiple runners with one panicing",
			runnables:    []runner.Runnable{newBlockingRunner(ctx, nil, nil), newPanickingRunner(ctx, nil, nil, panicA)},
			wantedErrors: []error{panicA},
		},
		{
			name:         "multiple runners with one failing and two panicing",
			runnables:    []runner.Runnable{newBlockingRunner(ctx, errA, nil), newPanickingRunner(ctx, nil, nil, panicA), newPanickingRunner(ctx, nil, nil, panicB)},
			wantedErrors: []error{errA, panicA, panicB},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := runner.Run(context.Background(), tt.runnables...)

			// Verify the exact number of errors returned.
			mErr := &multierror.Error{}
			errors.As(err, &mErr)
			assert.Equal(t, len(tt.wantedErrors), mErr.Len())

			// Verify that all expected errors were returned.
			for _, expectedErr := range tt.wantedErrors {
				assert.Truef(t, errors.Is(err, expectedErr), "expected error '%s' wasn't found from returned error: %s", expectedErr, err)
			}

			// Verify that all runnables were called.
			for i, r := range tt.runnables {
				assert.Truef(t, r.(*mockRunner).RunCalled(), "Run was not called for runner with index %d", i)
			}

			// Verify that all runnables were closed.
			for i, r := range tt.runnables {
				assert.Truef(t, r.(*mockRunner).CloseCalled(), "Close was not called for runner with index %d", i)
			}
		})
	}
}

func TestRun_ExitTriggeredFromExternalCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runnables := []runner.Runnable{
		newBlockingRunner(context.Background(), nil, nil),
		newBlockingRunner(context.Background(), nil, nil),
		newBlockingRunner(context.Background(), nil, nil),
	}

	go func() {
		// Verify that all runnables are running.
		for _, r := range runnables {
			<-r.(*mockRunner).runCalled
		}
		cancel()
	}()

	err := runner.Run(ctx, runnables...)
	assert.NoError(t, err)

	// Verify that all runnables were closed.
	for i, r := range runnables {
		assert.Truef(t, r.(*mockRunner).CloseCalled(), "Close was not called for runner with index %d", i)
	}
}

func TestFnRunner(t *testing.T) {
	r := newClosedRunner(nil, nil)
	err := runner.Run(context.Background(), runner.NewFnRunner(r.Run, r.Close))
	assert.NoError(t, err)
	assert.True(t, r.RunCalled(), "Run was not called for runner")
	assert.True(t, r.CloseCalled(), "Close was not called for runner")
}

type mockRunner struct {
	runErr      error
	closeErr    error
	runCalled   chan struct{}
	closeCalled chan struct{}
	ctx         context.Context
	cancel      func()
	panicErr    error
}

func newClosedRunner(runErr, closeErr error) *mockRunner {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return newMockRunner(ctx, runErr, closeErr, nil)
}

func newPanickingRunner(ctx context.Context, runErr, closeErr, panicErr error) *mockRunner {
	return newMockRunner(ctx, runErr, closeErr, panicErr)
}

func newBlockingRunner(ctx context.Context, runErr, closeErr error) *mockRunner {
	return newMockRunner(ctx, runErr, closeErr, nil)
}

func newMockRunner(ctx context.Context, runErr, closeErr, panicErr error) *mockRunner {
	ctx, cancel := context.WithCancel(ctx)
	return &mockRunner{
		runErr:      runErr,
		closeErr:    closeErr,
		runCalled:   make(chan struct{}),
		closeCalled: make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		panicErr:    panicErr,
	}
}

func (r *mockRunner) Run() error {
	close(r.runCalled)
	if r.panicErr != nil {
		panic(r.panicErr)
	}
	<-r.ctx.Done()
	return r.runErr
}

func (r *mockRunner) Close() error {
	close(r.closeCalled)
	r.cancel()
	return r.closeErr
}

func (r *mockRunner) RunCalled() bool {
	select {
	case <-r.runCalled:
		return true
	default:
		return false
	}
}

func (r *mockRunner) CloseCalled() bool {
	select {
	case <-r.closeCalled:
		return true
	default:
		return false
	}
}
