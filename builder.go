package patron

import "github.com/beatlabs/patron/sync/http"

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
