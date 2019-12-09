package http

import (
	"time"

	"github.com/beatlabs/patron/errors"
)

// Variables used for testing the HTTP Component builder

var httpBuilderNoErrors = []error{}
var httpBuilderAllErrors = []error{
	errors.New("Nil AliveCheckFunc was provided"),
	errors.New("Nil ReadyCheckFunc provided"),
	errors.New("Invalid HTTP Port provided"),
	errors.New("Negative or zero read timeout provided"),
	errors.New("Negative or zero write timeout provided"),
	errors.New("Empty Routes slice provided"),
	errors.New("Empty list of middlewares provided"),
	errors.New("Invalid cert or key provided"),
}

type builderTestcase struct {
	acf      AliveCheckFunc
	rcf      ReadyCheckFunc
	p        int
	rt       time.Duration
	wt       time.Duration
	rr       []Route
	mm       []MiddlewareFunc
	c        string
	k        string
	wantErrs []error
}

var builderTestcases = []builderTestcase{
	{
		acf: DefaultAliveCheck,
		rcf: DefaultReadyCheck,
		p:   httpPort,
		rt:  httpReadTimeout,
		wt:  httpIdleTimeout,
		rr: []Route{
			aliveCheckRoute(DefaultAliveCheck),
			readyCheckRoute(DefaultReadyCheck),
			metricRoute(),
		},
		mm: []MiddlewareFunc{
			NewRecoveryMiddleware(),
			panicMiddleware("error"),
		},
		c:        "cert.file",
		k:        "key.file",
		wantErrs: httpBuilderNoErrors,
	},
	{
		acf:      nil,
		rcf:      nil,
		p:        -1,
		rt:       -10 * time.Second,
		wt:       -20 * time.Second,
		rr:       []Route{},
		mm:       []MiddlewareFunc{},
		c:        "",
		k:        "",
		wantErrs: httpBuilderAllErrors,
	},
}
