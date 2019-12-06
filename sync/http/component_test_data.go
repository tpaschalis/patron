package http

import "github.com/beatlabs/patron/errors"

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
