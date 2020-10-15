# gRPC

On the server side, the gRPC component injects a `UnaryInterceptor` which handles tracing and log propagation.
On the client side, we inject a `UnaryInterceptor` which handles tracing and log propagation.
