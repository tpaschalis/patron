# Observability

## Metrics and Tracing

Tracing and metrics are provided by Jaeger's implementation of the OpenTracing project.
Every component has been integrated with the above library and produces traces and metrics.
Metrics are provided with the default HTTP component at the `/metrics` route for Prometheus to scrape.
Tracing will be sent to a jaeger agent which can be setup through environment variables mentioned in the config section. Sane defaults are applied for making the use easy.
We have included some clients inside the trace package which are instrumented and allow propagation of tracing to
downstream systems. The tracing information is added to each implementations header. These clients are:

- HTTP
- AMQP
- Kafka
- SQL