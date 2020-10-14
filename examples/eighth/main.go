package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
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

// In the following example, we define a route that serves some random data.
// We call this route with and without Accept-Encoding headers so we that we test the compression methods
// $ curl -s localhost:50000/foo | wc -c
// 1398106
// $ curl -s localhost:50000/foo -H "Accept-Encoding: nonexistent" | wc -c
// 1398106
// $ curl -s localhost:50000/foo -H "Accept-Encoding: gzip" | wc -c
// 1053068
// $ curl -s localhost:50000/foo -H "Accept-Encoding: deflate" | wc -c
// 1053045
// $ curl -s localhost:50000/foo -H "Accept-Encoding: compress" | wc -c
// 1458451
//
// For ignored routes, we don't see any compression applied, even if we specify a correct header
// $ curl -s localhost:50000/bar -H "Accept-Encoding: gzip" | wc -c
// 1398106
// $ curl -s localhost:50000/bar  | wc -c
// 1398106
func main() {
	name := "eighth"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	handle(err)

	compressionMiddleware, err := patronhttp.NewCompressionMiddleware().WithIgnoreRoutes("/bar", "/alive", "/ready", "/metrics").Build()
	handle(err)

	// You could either add the compression middleware per-route, like here ...
	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewRouteBuilder("/foo", eighth).MethodGet()). //.WithMiddlewares(compressionMiddleware))
		Append(patronhttp.NewRouteBuilder("/bar", eighth).MethodGet())

	// or pass middlewares to the HTTP component globally, like we do below
	ctx := context.Background()
	err = patron.New(name, version).
		WithRoutesBuilder(routesBuilder).
		WithMiddlewares(middlewareCors, compressionMiddleware).
		Run(ctx)
	handle(err)
}

// creates some random data to send back
func eighth(_ context.Context, _ *patronhttp.Request) (*patronhttp.Response, error) {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, 1<<20)
	_, err := rand.Read(data)
	if err != nil {
		return nil, err
	}

	return patronhttp.NewResponse(data), nil
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
