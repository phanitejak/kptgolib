# Tracing in Go

Tracing library abstracts underlying distributed tracing solution from services.
Please do not import opentracing or opentelemetry packages in your service code but instead use this library.
That way changing the chosen implementation can be switched from single place.

## Tracer configuration

Configuration is done via environmental variables. Variables that need to be provided are:

| Variable name             | Description                                                                                                           |
| ------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| JAEGER_SERVICE_NAME       | name of a service, e.g. `my-service`                                                                                  |
| JAEGER_ENDPOINT           | URL of Jaeger Collector HTTP endpoint                                                                                 |
| JAEGER_SAMPLER_TYPE       | type of sampler to be used, preferably `probabilistic`                                                                |
| JAEGER_SAMPLER_PARAM      | sampler configuration, for probabilistic: `0.01` = 1% of traces will be sent to jaeger, `1` = 100% of traces are sent |
| JAEGER_REPORTER_LOG_SPANS | log reported spans                                                                                                    |
| USE_SIMPLE_SPAN_PROCESSOR | if set to true, will report finished spans immediatelly, usually for testing purposes                                 |

Other variables can be used for configuration, for more information see [README on GitHub](https://github.com/jaegertracing/jaeger-client-go).

## Instrumenting Go code

### Tracing dependencies

Add this monorepo to your go.mod file and you will be able to import needed tracing libraries

```go
import(
	"github.com/phanitejak/gopkg/tracing"
)
```

### Instantiating Logger

In order to being able to correlate logs using `traceId`, you should create a tracing logger.

```go
logger := tracing.NewLogger(logging.NewLogger())
```

### Instantiating Tracer

As early as possible instantiate tracer and ensure that it will be closed when closing the application, for example:

> **NOTICE:** Deferring functions are **not invoked** when `os.Exit()` or `log.Fatal()` is called

```go
closer, err := tracing.InitGlobalTracer(tracing.WithLogger(logger))
if err != nil {
  return err
}
defer func () {
	err = closer.Close()
	log.Error(err)
}()
```

### Creating new span from context

General good practice is to have a Span per function.

You shall create spans for functions, you want to be visible in a [gaant chart in Jaeger UI](https://www.jaegertracing.io/docs/1.12/#trace-detail-view) to see how much time is spent for each operation:

- On db access
- Some heavy computations
- Long-running operations
- Cross-process communication (http, kafka)

Tracing provides method for creating such span from existing context:

> **NOTICE:** Always use defer to finish created spans.
> This way even if your function panics - span will be finished.

```go
func DoSomething(parentCtx context.Context){
	span, ctx := tracing.StartSpanFromContext(parentCtx, "DoSomething")
	defer span.Finish()

	//...do something
}
```

Span created this way will be created as a child span associated with the `parentCtx`.
If there is no such span, new span will be created without parent.

### Passing context

Tracing uses Go's `context.Context` for passing tracing context through application.
There are couple of practices to be kept when passing Go's Context:

- Context should be the first parameter of a function
- Do not overwrite parent context, passed as function parameter with newly created child context
- Never pass `nil` Context. If in doubt, `context.TODO` should be used instead of `nil`.

### Adding tags to a span

To add additional information to a span, you can add custom tags.
These can be later used to filter traces in Jaeger (e.g. to find all traces with errors for MRBTS-12035 agent from last hour).

> **NOTICE:** Just as with logs, you **SHOULD NOT** add any sensitive information to tags

```go
span.SetTag("tag-key", tagValue)
```

### Logging

Tracing library also provides a custom logger in order to correlate logs between your microservices using `traceId`.

To write logs with tracing information, additional method `For(ctx context.Context)` is introduced:

```go
logger.For(request.Context()).Info("my very important log")
```

It is also possible to use this logger without log correlation, when there is no tracing context (like application initialization).
It should be used then as usual:

```go
logger.Info("my very important log")
```

### Instrumenting HTTP Server

Http server has to extract Span information from HTTP request headers and write it into request's `context.Context`.

Then you can pass this `context.Context` further through your application to propagate trace through cross-process boundaries.

## Working with http-handler (preferred way)

```go
func setupMiddlewares(handler http.Handler) http.Handler {
	return tracing.Wrap(handler)
}
```

#### Working with Mux Router

Wrap router's handle functions using the `Wrap(route *mux.Route)` function:

```go
controllerFunc := func(writer http.ResponseWriter, request *http.Request) {
    serverSpan, _ = tracing.StartSpanFromContext(request.Context(), "test server")
	//Handle request
    defer serverSpan.Finish()
}
router.Handle("/resource/{resourceId}", tracing.Wrap(http.HandlerFunc(controllerFunc)))
```

### Instrumenting HTTP Client

Example below does several things:

- creates new span
- creates HTTP request
- injects tracing http headers into request using `tracing.RequestWithContext(myRequest, ctx)`
- executes request

```go
func sendRequestToSomeUrl(parentCtx context.Context){
  span, ctx := tracing.StartSpanFromContext(parentCtx, "sendRequestToSomeUrl")
  defer span.Finish()

  myRequest, err := http.NewRequest("GET", "some-url.com/something", myReader)
  if err !=nil {
  	return err
  }

  myRequestWithContext := tracing.RequestWithContext(myRequest, ctx)

  myHttpClient.Do(myRequestWithContext)

	//...do something
}
```

### Instrumenting kafka consumer

To extract span context from kafka message use `tracing.StartSpanFromMessage(msg, "myHandlerMethodName")` function.

Remember to finish your span when message processing is done.

```go
span, ctx := tracing.StartSpanFromMessage(msg, "HandleEvent")
defer span.Finish()
```

### Instrumenting kafka producer

To inject span context into kafka message headers use `tracing.MessageWithContext(myProducerMessage, ctx)` function

```go
myMessage := &sarama.ProducerMessage{}

myMessageWithContext := tracing.MessageWithContext(myMessage, ctx)

_, _, err := myProducer.SendMessage(myMessageWithContext)
if err != nil {
	return err
}
```
