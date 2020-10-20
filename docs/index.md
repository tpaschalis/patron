# patron ![CI](https://github.com/beatlabs/patron/workflows/CI/badge.svg) [![codecov](https://codecov.io/gh/beatlabs/patron/branch/master/graph/badge.svg)](https://codecov.io/gh/beatlabs/patron) [![Go Report Card](https://goreportcard.com/badge/github.com/beatlabs/patron)](https://goreportcard.com/report/github.com/beatlabs/patron) [![GoDoc](https://godoc.org/github.com/beatlabs/patron?status.svg)](https://godoc.org/github.com/beatlabs/patron) ![GitHub release](https://img.shields.io/github/release/beatlabs/patron.svg)

Patron is a framework for creating microservices, originally created by Sotiris Mantzaris (https://github.com/mantzas). This fork is maintained by Beat Engineering (https://thebeat.co)

`Patron` is french for `template` or `pattern`, but it means also `boss` which we found out later (no pun intended).

The entry point of the framework is the `Service`. The `Service` uses `Components` to handle the processing of sync and async requests. The `Service` starts by default an `HTTP Component` which hosts the debug, alive, ready and metric endpoints. Any other endpoints will be added to the default `HTTP Component` as `Routes`. Alongside `Routes` one can specify middleware functions to be applied ordered to all routes as `MiddlewareFunc`. The service set's up by default logging with `zerolog`, tracing and metrics with `jaeger` and `prometheus`.

`Patron` provides abstractions for the following functionality of the framework:

- service, which orchestrates everything
- components and processors, which provide an abstraction of adding processing functionality to the service
  - asynchronous message processing (RabbitMQ, Kafka, AWS SQS)
  - synchronous processing (HTTP)
  - gRPC support
- metrics and tracing
- logging

`Patron` provides the same defaults for making the usage as simple as possible.
`Patron` needs Go 1.13 as a minimum.

## Table of Contents

- [Architecture](Architecture.md)
- [CLI](other/CLI.md)
- Components
  - [Async](components/async/Async.md)
    - [Kafka](components/async/Kafka.md)
    - [AMQP](components/async/AMQP.md)
    - [AWS SQS](components/async/AWSSQS.md)
  - [HTTP](components/HTTP.md)
  - [gRPC](components/gRPC.md)
- [Clients](clients/Clients.md)
- Packages
  - [Reliability](other/Reliability.md)
  - [Observability](observability/Observability.md)
  - [Logging](observability/Logging.md)
  - [Distributed Tracing](observability/DistributedTracing.md)  
  - [Caching](other/Caching.md)
  - [Encoding](other/Encoding.md)
  - [Errors](other/Errors.md)
- [Examples](Examples.md)
- [Code of Conduct](CodeOfConduct.md)
- [Contribution Guidelines](ContributionGuidelines.md)
- [Acknowledgments](ACKNOWLEDGMENTS.md)
