package main

import (
	"context"
	"fmt"
	"github.com/beatlabs/patron"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
}

var middlewareCors = func(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
		w.Header().Add("Access-Control-Allow-Credentials", "Allow")
		h.ServeHTTP(w, r)
	})
}

// In the following example, we set up four routes that serve some random data.
// One route uses GZIP for the compression middleware, one uses Deflate, and one uses no compression
// Calling the routes below one can see
//
// -- No Compression
// $ curl -s localhost:50000/qux | wc -c
// 1398106
//
// -- GZIP compression, with and without headers
// $ curl -s -H "Accept-Encoding: gzip" localhost:50000/foo | wc -c
// 1053077
// $ curl -s localhost:50000/foo | wc -c
// 1398106
//
// -- Deflate compression, with and without headers
// $ curl -s -H "Accept-Encoding: deflate" localhost:50000/bar | wc -c
// 1053024
// $ curl -s localhost:50000/bar | wc -c
// 1398106
//
// -- LZW compression, with and without headers
// $ curl -s -H "Accept-Encoding: compress" localhost:50000/baz | wc -c
// 1457478
// $ curl -s localhost:50000/baz | wc -c
// 1398106
//
func main() {
	name := "eighth"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	handle(err)

	gzipMiddleware, err := patronhttp.NewCompressionMiddleware().WithGZIP().WithIgnoreRoutes("/alive", "/ready", "/metrics").Build()
	handle(err)
	deflateMiddleware, err := patronhttp.NewCompressionMiddleware().WithDeflate(8).WithIgnoreRoutes("/alive", "/ready", "/metrics").Build()
	handle(err)
	lzwMiddleware, err := patronhttp.NewCompressionMiddleware().WithLZW(1, 8).WithIgnoreRoutes("/alive", "/ready", "/metrics").Build()
	handle(err)

	// You can either add compression middlewares per-route, like here ...
	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewRouteBuilder("/foo", eighth).MethodGet().WithMiddlewares(gzipMiddleware)).
		Append(patronhttp.NewRouteBuilder("/bar", eighth).MethodGet().WithMiddlewares(deflateMiddleware)).
		Append(patronhttp.NewRouteBuilder("/baz", eighth).MethodGet().WithMiddlewares(lzwMiddleware)).
		Append(patronhttp.NewRouteBuilder("/qux", eighth).MethodGet())

	// or pass middlewares to the HTTP component globally, like we do with CORS below
	ctx := context.Background()
	err = patron.New(name, version).
		WithRoutesBuilder(routesBuilder).
		WithMiddlewares(middlewareCors).
		Run(ctx)
	handle(err)
}

// creates some random data to send back
func eighth(_ context.Context, _ *patronhttp.Request) (*patronhttp.Response, error) {
	rand.Seed(time.Now().UnixNano())
	oneMiB := make([]byte, 1<<20)
	_, err := rand.Read(oneMiB)
	if err != nil {
		return nil, err
	}

	return patronhttp.NewResponse(oneMiB), nil
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
