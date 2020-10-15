# HTTP

## Security

The necessary abstraction is available to implement authentication in the following components:

- HTTP

### HTTP

In order to use authentication, an authenticator has to be implemented following the interface:

```go
type Authenticator interface {
  Authenticate(req *http.Request) (bool, error)
}
```

This authenticator can then be used to set up routes with authentication.

The following authenticator is available:

- API key authenticator, see examples

## HTTP lifecycle endpoints

When creating a new HTTP component, Patron will automatically create a liveness and readiness route, which can be used to know the lifecycle of the application:

```
# liveness
GET /alive

# readiness
GET /ready
```

Both can return either a `200 OK` or a `503 Service Unavailable` status code (default: `200 OK`).

It is possible to customize their behaviour by injecting an `http.AliveCheck` and/or an `http.ReadyCheck` `OptionFunc` to the HTTP component constructor.

### HTTP Middlewares

A `MiddlewareFunc` preserves the default net/http middleware pattern.
You can create new middleware functions and pass them to Service to be chained on all routes in the default Http Component.

```go
type MiddlewareFunc func(next http.Handler) http.Handler

// Setup a simple middleware for CORS
newMiddleware := func(h http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Add("Access-Control-Allow-Origin", "*")
        // Next
        h.ServeHTTP(w, r)
    })
}
```