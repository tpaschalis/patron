package http

import (
	"context"
	"net/http"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const (
	// HTTPComponent definition.
	HTTPComponent = "http"
	// HTTPClientComponent definition.
	HTTPClientComponent = "http-client"

	versionTag = "version"
	hostsTag   = "hosts"
)

var (
	version = "dev"
)

// Client interface of a HTTP client.
type Client interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// TracedClient defines a HTTP client with tracing integrated.
type TracedClient struct {
	cl *http.Client
	cb *circuitbreaker.CircuitBreaker
}

// New creates a new HTTP client.
func New(oo ...OptionFunc) (*TracedClient, error) {
	tc := &TracedClient{
		cl: &http.Client{
			Timeout:   60 * time.Second,
			Transport: &nethttp.Transport{},
		},
		cb: nil,
	}

	for _, o := range oo {
		err := o(tc)
		if err != nil {
			return nil, err
		}
	}

	return tc, nil
}

// Do executes a HTTP request with integrated tracing and tracing propagation downstream.
func (tc *TracedClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req, ht := nethttp.TraceRequest(opentracing.GlobalTracer(), req,
		nethttp.OperationName(OpName(req.Method, req.URL.String())),
		nethttp.ComponentName(HTTPClientComponent))
	defer ht.Finish()

	req.Header.Set(correlation.HeaderID, correlation.IDFromContext(ctx))

	rsp, err := tc.do(req)
	if err != nil {
		ext.Error.Set(ht.Span(), true)
	} else {
		ext.HTTPStatusCode.Set(ht.Span(), uint16(rsp.StatusCode))
	}

	ext.HTTPMethod.Set(ht.Span(), req.Method)
	ext.HTTPUrl.Set(ht.Span(), req.URL.String())
	return rsp, err
}

func (tc *TracedClient) do(req *http.Request) (*http.Response, error) {
	if tc.cb == nil {
		return tc.cl.Do(req)
	}

	r, err := tc.cb.Execute(func() (interface{}, error) {
		return tc.cl.Do(req)
	})
	if err != nil {
		return nil, err
	}

	return r.(*http.Response), nil
}

// Span starts a new HTTP span.
func Span(path, corID string, r *http.Request) (opentracing.Span, *http.Request) {
	ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract HTTP span: %v", err)
	}
	sp := opentracing.StartSpan(OpName(r.Method, path), ext.RPCServerOption(ctx))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.HTTPUrl.Set(sp, r.URL.String())
	ext.Component.Set(sp, "http")
	sp.SetTag(versionTag, version)
	sp.SetTag(correlation.ID, corID)
	return sp, r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
}

// FinishHTTPSpan finishes a HTTP span by providing a HTTP status code.
func FinishHTTPSpan(sp opentracing.Span, code int) {
	ext.HTTPStatusCode.Set(sp, uint16(code))
	ext.Error.Set(sp, code >= http.StatusInternalServerError)
	sp.Finish()
}

// OpName return a string representation of the HTTP request operation.
func OpName(method, path string) string {
	return method + " " + path
}
