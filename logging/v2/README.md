# Go Logging V2

Version 2 of the logging library, which enforces trace context usage.

# What's new?

- Golang's context now is a mandatory parameter for each logging invocation
- Logging metrics are now aggregated under `logger_v2` subsystem - changes are needed in Grafana Dashboards

# Usage

### Initialize logger

```go
package example

import (
	"github.com/phanitejak/kptgolib/logging-v2"
	"github.com/phanitejak/kptgolib/tracing"
)

func main() {
	// Create your tracing logger
	log := logging.NewLogger()

	// Enable tracing in your application
	// and instrument your HTTP servers/clients and Kafka producers/consumers.
	// Same way as usual - to have context propagation setup in place
	_, err := tracing.InitGlobalTracer()
	if err != nil {
		// Error handle somehow
	}

	// Inject your logger into your application
	// Init and run your application

}

```

### Log with traceable context

```go
// Either create a span from existing request context or create a fresh one
package example

import (
	"context"

	"github.com/phanitejak/kptgolib/logging-v2"
	"github.com/opentracing/opentracing-go"
)

var log logging.Logger

func myOperation(parentCtx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(parentCtx, "myOperation")
	defer span.Finish()

	log.Info(ctx, "Message")
	// Example Output:
	// {"is_sampled":"false","level":"info","logger":"logging_test.go:70","message":"Message","parent_id":"0","span_id":"54b168451e541edd","timestamp":"2020-12-11T12:02:00.370+02:00","trace_id":"54b168451e541edd"}

}

```
