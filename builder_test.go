package patron

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	patronhttp "github.com/beatlabs/patron/sync/http"
	"github.com/stretchr/testify/assert"
)

func TestNew_MissingName(t *testing.T) {
	err := New("", "").Run()
	assert.EqualError(t, err, "name is required\n")
}

func TestNew_TraceError(t *testing.T) {
	require.NoError(t, os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "XXX"))
	err := New("name", "").Run()
	assert.EqualError(t, err, "env var for jaeger sampler param is not valid: strconv.ParseFloat: parsing \"XXX\": invalid syntax\n")
	require.NoError(t, os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "0.0"))
}

func TestNew_MissingRoutes(t *testing.T) {
	err := New("name", "").WithRoutes().Run()
	assert.EqualError(t, err, "routes are empty\n")
}

func TestNew_MissingMiddlewares(t *testing.T) {
	err := New("name", "").WithMiddlewares().Run()
	assert.EqualError(t, err, "middlewares are empty\n")
}

func TestNew_MissingHealthcheck(t *testing.T) {
	err := New("name", "").WithHealthCheck(nil).Run()
	assert.EqualError(t, err, "health check function is nil\n")
}

func TestNew_MissingComponents(t *testing.T) {
	err := New("name", "").WithComponents().Run()
	assert.EqualError(t, err, "components are empty\n")
}

func TestNew_MissingDocsFile(t *testing.T) {
	err := New("name", "").WithDocs("").Run()
	assert.EqualError(t, err, "failed to import doc file\n")
}

func TestNew_MissingSIGHUP(t *testing.T) {
	err := New("name", "").WithSIGHUP(nil).Run()
	assert.EqualError(t, err, "sighub handler is nil\n")
}

func TestRun_HttpError(t *testing.T) {
	require.NoError(t, os.Setenv("PATRON_HTTP_DEFAULT_PORT", "XXX"))
	err := New("name", "").Run()
	assert.EqualError(t, err, "env var for HTTP default port is not valid: strconv.ParseInt: parsing \"XXX\": invalid syntax")
	require.NoError(t, os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50000"))
}

func TestRun_Error(t *testing.T) {
	h := func(_ http.ResponseWriter, _ *http.Request) {
	}
	m := func(_ http.Handler) http.Handler { return nil }
	err := New("name", "").
		WithRoutes(patronhttp.NewRouteRaw("/", "GET", h, true)).
		WithMiddlewares(m).
		WithComponents(&testComponent{errorRunning: true}).
		Run()
	assert.EqualError(t, err, "failed to run component\n")
}
