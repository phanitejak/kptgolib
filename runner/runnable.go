// Package runner implements a simple runner interface for running multiple go routines which return errors
package runner

import (
	"context"
	"fmt"
	"runtime"

	"github.com/hashicorp/go-multierror"
)

// Runnable interface defines something runnable.
type Runnable interface {
	Run() error
	Close() error
}

// FnRunner is a structure with run and close functions.
type FnRunner struct {
	run   func() error
	close func() error
}

// NewFnRunner is a function for constructing a new FnRunner.
func NewFnRunner(runFn, closeFn func() error) *FnRunner {
	return &FnRunner{
		run:   runFn,
		close: closeFn,
	}
}

// Run executes the f.run() function.
func (f *FnRunner) Run() error {
	return f.run()
}

// Close executes the f.close() function.
func (f *FnRunner) Close() error {
	return f.close()
}

// Run executes all the runnables given to it, ctx can be canceled to control the execution flow from outside.
func Run(ctx context.Context, runnables ...Runnable) error {
	if len(runnables) == 0 {
		return nil
	}

	wg := &multierror.Group{}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, runnable := range runnables {
		runnable := runnable
		started := make(chan struct{})
		wg.Go(func() (err error) {
			defer cancel()
			defer func() { err = recoverPanicOrReturnErr(recover(), err) }()

			close(started)
			return runnable.Run()
		})
		<-started
	}

	<-ctx.Done()

	for i := len(runnables) - 1; i >= 0; i-- {
		runnable := runnables[i]
		started := make(chan struct{})
		wg.Go(func() (err error) {
			defer func() { err = recoverPanicOrReturnErr(recover(), err) }()
			defer close(started)
			return runnable.Close()
		})
		<-started
	}

	return wg.Wait().ErrorOrNil()
}

func recoverPanicOrReturnErr(recover interface{}, err error) error {
	if r := recover; r != nil {
		// Same as stdlib http server code. Manually allocate stack trace buffer size
		// to prevent excessively large logs
		const size = 64 << 10
		stacktrace := make([]byte, size)
		stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]

		if e, ok := r.(error); ok {
			return fmt.Errorf("observed a panic %w\n%s", e, stacktrace)
		}
		if s, ok := r.(string); ok {
			return fmt.Errorf("observed a panic: %s\n%s", s, stacktrace)
		}
		return fmt.Errorf("observed a panic: %#v (%v)\n%s", r, r, stacktrace)
	}

	return err
}
