// Package runner takes care of running App using it's life cycle methods.
package runner

import (
	"context"
	"os"
	"os/signal"

	"github.com/phanitejak/kptgolib/logging"
	"github.com/phanitejak/kptgolib/tracing"
)

// exitFn allows overwritting os.Exit for testing purposes.
var exitFn = os.Exit

// Module abstracts runnable units into life cycle methods like http servers for example.
// Run() is expected to block until app is finished.
type Module interface {
	Init(*tracing.Logger) error
	Run() error
	Close() error
}

// App abstracts application to it's  identification and modules.
type App interface {
	Name() string
	Modules() []Module
}

// AppRunner can be used to run App.
type AppRunner struct {
	log   *tracing.Logger
	ctx   context.Context
	ready chan struct{}
}

// RunApp is convenience function to create new Runner with tracing logger, hook into os.Interrupt and start running an App.
// If any of App's life cycle methods returns an error it will be logged and os.Exit(1) will be issued.
func RunApp(a App) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log := tracing.NewLogger(logging.NewLogger())
	exitCode := NewRunner(ctx, log).Run(a)
	exitFn(exitCode)
}

// NewRunner creates runner with given logger and channel for signaling when to stop.
func NewRunner(ctx context.Context, log *tracing.Logger) *AppRunner {
	return &AppRunner{
		log:   log,
		ctx:   ctx,
		ready: make(chan struct{}),
	}
}

// Ready will return once everything is initialized successfully.
func (r *AppRunner) Ready() {
	<-r.ready
}

// Run will take care of running app.
func (r *AppRunner) Run(a App) (exitCode int) {
	mods := a.Modules()
	runnables := make([]Runnable, 0, len(mods))

	r.log.Infof("initializing %s", a.Name())
	for _, mod := range mods {
		runnables = append(runnables, mod)
		if err := mod.Init(r.log); err != nil {
			r.log.Errorf("failed to initialize modules for %s: %s", a.Name(), err)
			close(r.ready)
			return 1
		}
	}
	r.log.Infof("%s initialized successfully", a.Name())

	ctx, cancel := context.WithCancel(r.ctx)
	runnables = append(runnables, NewFnRunner(
		func() error {
			<-ctx.Done()
			r.log.Infof("Context cancelled for %s", a.Name())
			return nil
		},
		func() error {
			cancel()
			return nil
		},
	))

	r.log.Infof("running %s", a.Name())
	close(r.ready)

	if err := Run(ctx, runnables...); err != nil {
		r.log.Errorf("%s exited with error: %s", a.Name(), err)
		exitCode = 1
	}

	return exitCode
}

// Initializer wraps modules Init method.
type Initializer interface {
	Init(*tracing.Logger) error
}

type initable struct {
	i    Initializer
	done chan struct{}
}

// InitializerAsModule wraps i as module with Run and Close methods.
func InitializerAsModule(i Initializer) Module {
	return &initable{
		i:    i,
		done: make(chan struct{}),
	}
}

// Init calls Init of underlying Initializer.
func (i *initable) Init(l *tracing.Logger) error {
	return i.i.Init(l)
}

// Run will block until close is called.
func (i *initable) Run() error {
	<-i.done
	return nil
}

// Close will cause run to exit.
func (i *initable) Close() error {
	close(i.done)
	return nil
}

// Closer wraps modules Close method.
type Closer interface {
	Close() error
}

type closable struct {
	c    Closer
	done chan struct{}
}

// CloserAsModule wraps c as module with Init and Run methods.
func CloserAsModule(c Closer) Module {
	return &closable{
		c:    c,
		done: make(chan struct{}),
	}
}

// Init calls Init of underlying Closer.
func (c *closable) Init(*tracing.Logger) error {
	return nil
}

// Run will block until close is called.
func (c *closable) Run() error {
	<-c.done
	return nil
}

// Close will cause run to exit.
func (c *closable) Close() error {
	err := c.c.Close()
	close(c.done)
	return err
}
