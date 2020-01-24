package trace

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/rpcmetrics"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

const (
	versionTag = "version"
	hostsTag   = "hosts"
)

var (
	cls     io.Closer
	version = "dev"
)

// Setup tracing by providing all necessary parameters.
func Setup(name, ver, agent, typ string, prm float64) error {
	if ver != "" {
		version = ver
	}
	cfg := config.Configuration{
		ServiceName: name,
		Sampler: &config.SamplerConfig{
			Type:  typ,
			Param: prm,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agent,
		},
	}
	time.Sleep(100 * time.Millisecond)
	metricsFactory := prometheus.New()
	tr, clsTemp, err := cfg.NewTracer(
		config.Logger(jaegerLoggerAdapter{}),
		config.Observer(rpcmetrics.NewObserver(metricsFactory.Namespace(name, nil), rpcmetrics.DefaultNameNormalizer)),
	)
	if err != nil {
		return fmt.Errorf("cannot initialize jaeger tracer: %w", err)
	}
	cls = clsTemp
	opentracing.SetGlobalTracer(tr)
	version = ver
	return nil
}

// Close the tracer.
func Close() error {
	log.Debug("closing tracer")
	return cls.Close()
}

// ConsumerSpan starts a new consumer span.
func ConsumerSpan(ctx context.Context, opName, cmp, corID string, hdr map[string]string,
	tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.TextMapCarrier(hdr))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract consumer span: %v", err)
	}
	sp := opentracing.StartSpan(opName, consumerOption{ctx: spCtx})
	ext.Component.Set(sp, cmp)
	sp.SetTag(correlation.ID, corID)
	sp.SetTag(versionTag, version)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	return sp, opentracing.ContextWithSpan(ctx, sp)
}

// SpanComplete finishes a span with or without a error indicator.
func SpanComplete(sp opentracing.Span, err error) {
	ext.Error.Set(sp, err != nil)
	sp.Finish()
}

// SpanSuccess finishes a span with a success indicator.
func SpanSuccess(sp opentracing.Span) {
	ext.Error.Set(sp, false)
	sp.Finish()
}

// SpanError finishes a span with a error indicator.
func SpanError(sp opentracing.Span) {
	ext.Error.Set(sp, true)
	sp.Finish()
}

// ChildSpan starts a new child span with specified tags.
func ChildSpan(ctx context.Context, opName, cmp string, tags ...opentracing.Tag) (opentracing.Span, context.Context) {
	sp, ctx := opentracing.StartSpanFromContext(ctx, opName)
	ext.Component.Set(sp, cmp)
	for _, t := range tags {
		sp.SetTag(t.Key, t.Value)
	}
	sp.SetTag(versionTag, version)
	return sp, ctx
}

type jaegerLoggerAdapter struct {
}

func (l jaegerLoggerAdapter) Error(msg string) {
	log.Error(msg)
}

func (l jaegerLoggerAdapter) Infof(msg string, args ...interface{}) {
	log.Infof(msg, args...)
}

type consumerOption struct {
	ctx opentracing.SpanContext
}

func (r consumerOption) Apply(o *opentracing.StartSpanOptions) {
	if r.ctx != nil {
		opentracing.ChildOf(r.ctx).Apply(o)
	}
	ext.SpanKindConsumer.Apply(o)
}

// ComponentOpName returns a operation name for a component.
func ComponentOpName(cmp, target string) string {
	return cmp + " " + target
}
