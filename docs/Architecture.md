# Architecture

Patron has two basic concepts:

- Component, which define a long-running task like a server e.g. HTTP, gRPC, Kafka consumer, etc.
- Service, which is responsible to run provided components and monitor them for errors

## Component

A `Component` is an interface that exposes the following API:

```go
type Component interface {
  Run(ctx context.Context) error  
}
```

The above API gives the `Service` the ability to start and gracefully shutdown a `component` via context cancellation.  
The framework distinguishes between two types of components:

- synchronous, which are components that follow the request/response pattern and
- asynchronous, which consume messages from a source but don't respond anything back

The following component implementations are available:

- HTTP (sync)
- gRPC
- RabbitMQ consumer (async)
- Kafka consumer (async)
- AWS SQS (async)

## Service

The `Service` has the role of gluing all the above together:

- setting up logging, metrics and tracing
- setting up default HTTP component with the following endpoints configured:
  - profiling via pprof
  - liveness check
  - readiness check
- setting up termination by os signal
- setting up SIGHUP custom hook if provided by an option
- starting and stopping components
- handling component errors

The service has some default settings which can be changed via environment variables:

- Service HTTP port, for setting the default HTTP components port to `50000` with `PATRON_HTTP_DEFAULT_PORT`
- Service HTTP read and write timeout, use `PATRON_HTTP_READ_TIMEOUT`, `PATRON_HTTP_WRITE_TIMEOUT` respectively. For acceptable values check [here](https://golang.org/pkg/time/#ParseDuration).
- Log level, for setting the logger with `INFO` log level with `PATRON_LOG_LEVEL`
- Tracing, for setting up jaeger tracing with
  - agent host `0.0.0.0` with `PATRON_JAEGER_AGENT_HOST`
  - agent port `6831` with `PATRON_JAEGER_AGENT_PORT`
  - sampler type `probabilistic`with `PATRON_JAEGER_SAMPLER_TYPE`
  - sampler param `0.0` with `PATRON_JAEGER_SAMPLER_PARAM`, which means that no traces are sent.
  
