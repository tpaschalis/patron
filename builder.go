package patron

import (
	"os"
	"strconv"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/info"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/zerolog"
	"github.com/beatlabs/patron/sync/http"
	"github.com/beatlabs/patron/trace"
	jaeger "github.com/uber/jaeger-client-go"
)

// Setup set's up metrics and default logging.
func Setup(name, version string) error {
	lvl, ok := os.LookupEnv("PATRON_LOG_LEVEL")
	if !ok {
		lvl = string(log.InfoLevel)
	}

	info.UpsertConfig("log_level", lvl)
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}
	info.UpdateHost(hostname)

	f := map[string]interface{}{
		"srv":  name,
		"ver":  version,
		"host": hostname,
	}
	logSetupOnce.Do(func() {
		err = log.Setup(zerolog.Create(log.Level(lvl)), f)
	})

	return err
}

// Builder definition.
type Builder struct {
	name          string
	version       string
	routes        []http.Route
	middlewares   []http.MiddlewareFunc
	healthCheck   http.HealthCheckFunc
	components    []Component
	docFile       string
	sighupHandler func()
}

// New builder constructor.
func New(name string, version string) *Builder {
	return &Builder{name: name, version: version}
}

// WithRoutes adds routes to the service.
func (b *Builder) WithRoutes(rr []http.Route) *Builder {
	b.routes = rr
	return b
}

// WithMiddlewares adds middlewares to the service.
func (b *Builder) WithMiddlewares(mm ...http.MiddlewareFunc) *Builder {
	b.middlewares = mm
	return b
}

// WithHealthCheck adds a custom health check to the service.
func (b *Builder) WithHealthCheck(hcf http.HealthCheckFunc) *Builder {
	b.healthCheck = hcf
	return b
}

// WithComponents adds custom components to the service.
func (b *Builder) WithComponents(cc ...Component) *Builder {
	b.components = cc
	return b
}

// WithDocs adds docs support to the service.
func (b *Builder) WithDocs(file string) *Builder {
	b.docFile = file
	return b
}

// WithSIGHUP adds custom SIGHUP handling to the service.
func (b *Builder) WithSIGHUP(handler func()) *Builder {
	b.sighupHandler = handler
	return b
}

// Run the service.
func (b *Builder) Run() error {

	err := Setup(b.name, b.version)
	if err != nil {
		return err
	}

	err = setupDefaultTracing(b.name, b.version)
	if err != nil {
		return err
	}
	defer func() {
		err := trace.Close()
		if err != nil {
			log.Errorf("failed to close trace %v", err)
		}
	}()

	var options []optionFunc

	if len(b.routes) > 0 {
		options = append(options, routes(b.routes))
	}

	if len(b.middlewares) > 0 {
		options = append(options, middlewares(b.middlewares...))
	}

	if b.healthCheck != nil {
		options = append(options, healthCheck(b.healthCheck))
	}

	if len(b.components) > 0 {
		options = append(options, components(b.components...))
	}

	if b.docFile != "" {
		options = append(options, docs(b.docFile))
	}

	if b.sighupHandler != nil {
		options = append(options, sighub(b.sighupHandler))
	}

	s, err := new(b.name, b.version, options...)
	if err != nil {
		return err
	}
	return s.Run()
	//TODO: fix cli to support the above
}

func setupDefaultTracing(name, version string) error {
	var err error

	host, ok := os.LookupEnv("PATRON_JAEGER_AGENT_HOST")
	if !ok {
		host = "0.0.0.0"
	}
	port, ok := os.LookupEnv("PATRON_JAEGER_AGENT_PORT")
	if !ok {
		port = "6831"
	}
	agent := host + ":" + port
	info.UpsertConfig("jaeger-agent", agent)
	tp, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_TYPE")
	if !ok {
		tp = jaeger.SamplerTypeProbabilistic
	}
	info.UpsertConfig("jaeger-agent-sampler-type", tp)
	var prmVal = 0.0
	var prm = "0.0"

	if prm, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_PARAM"); ok {
		prmVal, err = strconv.ParseFloat(prm, 64)
		if err != nil {
			return errors.Wrap(err, "env var for jaeger sampler param is not valid")
		}
	}

	info.UpsertConfig("jaeger-agent-sampler-param", prm)
	log.Infof("setting up default tracing %s, %s with param %s", agent, tp, prm)
	return trace.Setup(name, version, agent, tp, prmVal)
}
