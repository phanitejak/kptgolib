// Package tracermod provides tracer module.
package tracermod

import (
	"io"

	"gopkg/tracing"
)

// GlobalTracer allows running tracer as module.
type GlobalTracer struct {
	done   chan struct{}
	closer io.Closer
}

// NewGlobalTracer returns instance of GlobalTracer.
func NewGlobalTracer() *GlobalTracer {
	return &GlobalTracer{}
}

// Init will initialize global tracer.
func (t *GlobalTracer) Init(l *tracing.Logger) (err error) {
	t.done = make(chan struct{})
	// TODO: Expose tracing option type so that it can be used here.
	t.closer, err = tracing.InitGlobalTracer(tracing.WithLogger(l))
	return err
}

// Run will block until close is called.
func (t *GlobalTracer) Run() error {
	<-t.done
	return nil
}

// Close will close global tracer, flush the traces and make Run() to return.
func (t *GlobalTracer) Close() error {
	close(t.done)
	return t.closer.Close()
}
