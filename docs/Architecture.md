# Architecture

## Service

The `Service` has the role of gluing all of the above together, which are:

- setting up logging
- setting up default HTTP component with the following endpoints configured:
  - profiling via pprof
  - liveness check
  - readiness check
- setting up termination by os signal
- setting up SIGHUP custom hook if provided by an option
- starting and stopping components
- handling component errors
- setting up metrics and tracing

The service has some default settings which can be changed via environment variables:

- Service HTTP port, for setting the default HTTP components port to `50000` with `PATRON_HTTP_DEFAULT_PORT`
- Service HTTP read and write timeout, for changing the default HTTP read and write timeout to any value, use `PATRON_HTTP_READ_TIMEOUT`, `PATRON_HTTP_WRITE_TIMEOUT` respectively. For acceptable values check [here](https://golang.org/pkg/time/#ParseDuration).
- Log level, for setting zerolog with `INFO` log level with `PATRON_LOG_LEVEL`
- Tracing, for setting up jaeger tracing with
  - agent host `0.0.0.0` with `PATRON_JAEGER_AGENT_HOST`
  - agent port `6831` with `PATRON_JAEGER_AGENT_PORT`
  - sampler type `probabilistic`with `PATRON_JAEGER_SAMPLER_TYPE`
  - sampler param `0.0` with `PATRON_JAEGER_SAMPLER_PARAM`, which means that traces are not initiated here.

### Component

A `Component` is an interface that exposes the following API:

```go
type Component interface {
  Run(ctx context.Context) error  
}
```

The above API gives the `Service` the ability to start and gracefully shutdown a `component` via context cancellation. Furthermore, the component describes itself by implementing the `Info` method and thus giving the service the ability to report the information of all components. The framework divides the components in 2 categories:

- synchronous, which are components that follow the request/response pattern and
- asynchronous, which consume messages from a source but don't respond anything back

The following component implementations are available:

- HTTP (sync)
- gRPC
- RabbitMQ consumer (async)
- Kafka consumer (async)
- AWS SQS (async)

Adding to the above list is as easy as implementing a `Component` and a `Processor` for that component.

## Processors

### Synchronous

The implementation of the processor is responsible to create a `Request` by providing everything that is needed (Headers, Fields, decoder, raw io.Reader) pass it to the implementation by invoking the `Process` method and handle the `Response` or the `error` returned by the processor.

The sync package contains only a function definition along with the models needed:

```go
type ProcessorFunc func(context.Context, *Request) (*Response, error)
```

The `Request` model contains the following properties (which are provided when calling the "constructor" `NewRequest`)

- Fields, which may contain any fields associated with the request
- Raw, the raw request data (if any) in the form of a `io.Reader`
- Headers, the request headers in the form of `map[string]string`
- decode, which is a function of type `encoding.Decode` that decodes the raw reader

An exported function exists for decoding the raw io.Reader in the form of

```go
Decode(v interface{}) error
```

The `Response` model contains the following properties (which are provided when calling the "constructor" `NewResponse`)

- Payload, which may hold a struct of type `interface{}`

### Middlewares per Route

Middlewares can also run per routes using the processor as Handler.
So using the `Route` helpers:

```go
// A route with ...MiddlewareFunc that will run for this route only + tracing
route := NewRoute("/index", "GET" ProcessorFunc, true, ...MiddlewareFunc)
// A route with ...MiddlewareFunc that will run for this route only + auth + tracing
routeWithAuth := NewAuthRoute("/index", "GET" ProcessorFunc, true, Authendicator, ...MiddlewareFunc)
```

### HTTP Caching

The caching layer for HTTP routes is specified per Route.

```go
// RouteCache is the builder needed to build a cache for the corresponding route
type RouteCache struct {
	// cache is the ttl cache implementation to be used
	cache cache.TTLCache
	// age specifies the minimum and maximum amount for max-age and min-fresh header values respectively
	// regarding the client cache-control requests in seconds
	age age
}

func NewRouteCache(ttlCache cache.TTLCache, age Age) *RouteCache
```

#### server cache
- The **cache key** is based on the route path and the url request parameters.
- The server caches only **GET requests**.
- The server implementation must specify an **Age** parameters upon construction.
- Age with **Min=0** and **Max=0** effectively disables caching
- The route should return always the most fresh object instance.
- An **ETag header** must be always in responses that are part of the cache, representing the hash of the response.
- Requests within the time-to-live threshold, will be served from the cache. 
Otherwise the request will be handled as usual by the route processor function. 
The resulting response will be cached for future requests.
- Requests where the client control header requirements cannot be met i.e. **very low max-age** or **very high min-fresh** parameters,
will be returned to the client with a `Warning` header present in the response. 

```
Note : When a cache is used, the handler execution might be skipped.
That implies that all generic handler functionalities MUST be delegated to a custom middleware.
i.e. counting number of server client requests etc ... 
```

### Usage

- provide the cache in the route builder
```go
NewRouteBuilder("/", handler).
	WithRouteCache(cache, http.Age{
		Min: 30 * time.Minute,
		Max: 1 * time.Hour,
	}).
    MethodGet()
```

- use the cache as a middleware
```go
NewRouteBuilder("/", handler).
    WithMiddlewares(NewCachingMiddleware(NewRouteCache(cc, Age{Max: 10 * time.Second}))).
    MethodGet()
```

#### client cache-control
The client can control the cache with the appropriate Headers
- `max-age=?` 

returns the cached instance only if the age of the instance is lower than the max-age parameter.
This parameter is bounded from below by the server option `minAge`.
This is to avoid chatty clients with no cache control policy (or very aggressive max-age policy) to effectively disable the cache
- `min-fresh=?` 
 
returns the cached instance if the time left for expiration is lower than the provided parameter.
This parameter is bounded from above by the server option `maxFresh`.
This is to avoid chatty clients with no cache control policy (or very aggressive min-fresh policy) to effectively disable the cache

- `no-cache` / `no-store`

returns a new response to the client by executing the route processing function.
NOTE : Except for cases where a `minAge` or `maxFresh` parameter has been specified in the server.
This is again a safety mechanism to avoid 'aggressive' clients put unexpected load on the server.
The server is responsible to cap the refresh time, BUT must respond with a `Warning` header in such a case.
- `only-if-cached`

expects any response that is found in the cache, otherwise returns an empty response

#### metrics

The http cache exposes several metrics, used to 
- assess the state of the cache
- help trim the optimal time-to-live policy
- identify client control interference

By default we are using prometheus as the the pre-defined metrics framework.

- `additions = misses + evictions`

Always , the cache addition operations (objects added to the cache), 
must be equal to the misses (requests that were not cached) plus the evictions (expired objects).
Otherwise we would expect to notice also an increased amount of errors or having the cache misbehaving in a different manner.

- `additions ~ misses`

If the additions and misses are comparable e.g. misses are almost as many as the additions, 
it would point to some cleanup of the cache itself. In that case the cache seems to not be able to support
the request patterns and control headers.

- `hits ~ additions`

The cache hit count represents how well the cache performs for the access patterns of client requests. 
If this number is rather low e.g. comparable to the additions, 
this would signify that probably a cache is not a good option for the access patterns at hand.

- `eviction age`

The age at which the objects are evicted from the cache is a very useful indicator. 
If the vast amount of evictions are close to the time to live setting, it would indicate a nicely working cache.
If we find that many evictions happen before the time to live threshold, clients would be making use cache-control headers.
 

#### cache design reference
- https://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
- https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.9

#### improvement considerations
- we can could the storing of the cached objects and their age counter. That way we would avoid loading the whole object in memory,
if the object is already expired. This approach might provide considerable performance (in terms of memory utilisation) 
improvement for big response objects. 
- we could extend the metrics to use the key of the object as a label as well for more fine-grained tuning.
But this has been left out for now, due to the potentially huge number of metric objects.
We can review according to usage or make this optional in the future.
- improve the serialization performance for the cache response objects

### Asynchronous

The implementation of the async processor follows exactly the same principle as the sync processor.
The main difference is that:

- The `Request` is the `Message` and contains only data as `[]byte`
- There is no `Response`, so the processor may return an error

```go
type ProcessorFunc func(context.Context, *Message) error
```

Everything else is exactly the same.
