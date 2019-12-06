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

// +++ Actual
// @@ -1,2 +1,11 @@
// -([]error) <nil>
// +([]error) (len=8) {
// + (*errors.fundamental)(Nil AliveCheckFunc was provided),
// + (*errors.fundamental)(Nil ReadyCheckFunc provided),
// + (*errors.fundamental)(Invalid HTTP Port provided),
// + (*errors.fundamental)(Negative or zero read timeout provided),
// + (*errors.fundamental)(Negative or zero write timeout provided),
// + (*errors.fundamental)(Empty Routes slice provided),
// + (*errors.fundamental)(Empty list of middlewares provided),
// + (*errors.fundamental)(Invalid cert or key provided)
// +}
