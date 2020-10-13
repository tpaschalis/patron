package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A middleware generator that tags resp for assertions
func tagMiddleware(tag string) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(tag))
			// next
			h.ServeHTTP(w, r)
		})
	}
}

// Panic middleware to test recovery middleware
func panicMiddleware(v interface{}) MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(v)
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"middleware 1,2,3 and finish", args{next: handler, mws: []MiddlewareFunc{t1, t2, t3}}, 202, "t1\nt2\nt3\n"},
		{"middleware 1,2 and finish", args{next: handler, mws: []MiddlewareFunc{t1, t2}}, 202, "t1\nt2\n"},
		{"no middleware and finish", args{next: handler, mws: []MiddlewareFunc{}}, 202, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

func TestMiddlewares(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name         string
		args         args
		expectedCode int
		expectedBody string
	}{
		{"auth middleware success", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{success: true})}}, 202, ""},
		{"auth middleware false", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{success: false})}}, 401, "Unauthorized\n"},
		{"auth middleware error", args{next: handler, mws: []MiddlewareFunc{NewAuthMiddleware(&MockAuthenticator{err: errors.New("auth error")})}}, 500, "Internal Server Error\n"},
		{"tracing middleware", args{next: handler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, 202, ""},
		{"recovery middleware from panic 1", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware("error")}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 2", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(errors.New("error"))}}, 500, "Internal Server Error\n"},
		{"recovery middleware from panic 3", args{next: handler, mws: []MiddlewareFunc{NewRecoveryMiddleware(), panicMiddleware(-1)}}, 500, "Internal Server Error\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
		})
	}
}

// TestSpanLogError tests whether an HTTP handler with a tracing middleware adds a log event in case of we return an error.
func TestSpanLogError(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)

	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("foo"))
		require.NoError(t, err)
	})

	r, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}
	tests := []struct {
		name                 string
		args                 args
		expectedCode         int
		expectedBody         string
		expectedSpanLogError string
	}{
		{"tracing middleware - error", args{next: errorHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, http.StatusInternalServerError, "foo", "foo"},
		{"tracing middleware - success", args{next: successHandler, mws: []MiddlewareFunc{NewLoggingTracingMiddleware("/index")}}, http.StatusOK, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mtr.Reset()
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())

			if tt.expectedSpanLogError != "" {
				require.Equal(t, 1, len(mtr.FinishedSpans()))
				spanLogError := getSpanLogError(t, mtr.FinishedSpans()[0])
				assert.Equal(t, tt.expectedSpanLogError, spanLogError)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	rc := httptest.NewRecorder()
	rw := newResponseWriter(rc)

	_, err := rw.Write([]byte("test"))
	assert.NoError(t, err)
	rw.WriteHeader(202)

	assert.Equal(t, 202, rw.status, "status expected 202 but got %d", rw.status)
	assert.Len(t, rw.Header(), 1, "Header count expected to be 1")
	assert.True(t, rw.statusHeaderWritten, "expected to be true")
	assert.Equal(t, "test", rc.Body.String(), "body expected to be test but was %s", rc.Body.String())
}

func TestStripQueryString(t *testing.T) {
	type args struct {
		path string
	}
	tests := map[string]struct {
		args         args
		expectedPath string
		expectedErr  error
	}{
		"query string 1": {
			args: args{
				path: "foo?bar=value1&baz=value2",
			},
			expectedPath: "foo",
		},
		"query string 2": {
			args: args{
				path: "/foo?bar=value1&baz=value2",
			},
			expectedPath: "/foo",
		},
		"query string 3": {
			args: args{
				path: "http://foo/bar?baz=value1",
			},
			expectedPath: "http://foo/bar",
		},
		"no query string": {
			args: args{
				path: "http://foo/bar",
			},
			expectedPath: "http://foo/bar",
		},
		"empty": {
			args: args{
				path: "",
			},
			expectedPath: "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s, err := stripQueryString(tt.args.path)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPath, s)
			}
		})
	}
}

func getSpanLogError(t *testing.T, span *mocktracer.MockSpan) string {
	logs := span.Logs()
	if len(logs) == 0 {
		assert.FailNow(t, "empty logs")
		return ""
	}

	for _, log := range logs {
		for _, field := range log.Fields {
			if field.Key == fieldNameError {
				return field.ValueString
			}
		}
	}

	assert.FailNowf(t, "missing logs", "missing field %s", fieldNameError)
	return ""
}

func TestNewCompressionMiddleware(t *testing.T) {
	tests := map[string]struct {
		cm *CmBuilder
	}{
		"gzip":     {cm: NewCompressionMiddleware().WithGZIP()},
		"deflate":  {cm: NewCompressionMiddleware().WithDeflate(8)},
		"compress": {cm: NewCompressionMiddleware().WithLZW(0, 8)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
			req, err := http.NewRequest("GET", "/test", nil)
			assert.NoError(t, err)

			req.Header.Set("Accept-Encoding", name)
			gzipMiddleware, err := tc.cm.Build()
			assert.NoError(t, err)
			assert.NotNil(t, gzipMiddleware)

			rc := httptest.NewRecorder()
			gzipMiddleware(handler).ServeHTTP(rc, req)
			actual := rc.Header().Get("Content-Encoding")
			assert.NotNil(t, actual)
			assert.Equal(t, name, actual)
		})
	}
}

func TestNewCompressionMiddleware_Ignore(t *testing.T) {
	var ceh, cth string // accept-encoding, content type

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })

	tests := map[string]struct {
		cm *CmBuilder
	}{
		"gzip":     {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithGZIP()},
		"deflate":  {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithDeflate(8)},
		"compress": {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithLZW(0, 8)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			middleware, err := tc.cm.Build()
			assert.NoError(t, err)
			assert.NotNil(t, middleware)

			// check if the route actually ignored
			req1, err := http.NewRequest("GET", "/metrics", nil)
			assert.NoError(t, err)
			req1.Header.Set("Accept-Encoding", name)
			assert.NoError(t, err)

			rc1 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc1, req1)

			ceh = rc1.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, ceh, "")

			cth = rc1.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, cth, "")

			// check if other routes remains untouched
			req2, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			req2.Header.Set("Accept-Encoding", name)
			req2.Header.Set("Content-Type", "application/json")

			rc2 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc2, req2)

			ceh = rc2.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, name, ceh)

			cth = rc2.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, "application/json", cth)
		})
	}
}

func TestNewCompressionMiddleware_Headers(t *testing.T) {
	var ceh, cth string // accept-encoding, content type

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })

	tests := map[string]struct {
		cm *CmBuilder
	}{
		"gzip":     {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithGZIP()},
		"deflate":  {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithDeflate(8)},
		"compress": {cm: NewCompressionMiddleware().WithIgnoreRoutes("/metrics").WithLZW(0, 8)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			middleware, err := tc.cm.Build()
			assert.NoError(t, err)
			assert.NotNil(t, middleware)

			// check if the route actually ignored
			req1, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			req1.Header.Set("Accept-Encoding", name)
			req1.Header.Set("Content-Type", "text/plain")

			rc1 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc1, req1)

			ceh = rc1.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, name, ceh)

			cth = rc1.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, "text/plain", cth)

			// check if other routes remains untouched
			req2, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			req2.Header.Set("Accept-Encoding", name)
			req2.Header.Set("Content-Type", "application/json")

			rc2 := httptest.NewRecorder()
			middleware(handler).ServeHTTP(rc2, req2)

			ceh = rc2.Header().Get("Content-Encoding")
			assert.NotNil(t, ceh)
			assert.Equal(t, name, ceh)

			cth = rc2.Header().Get("Content-Type")
			assert.NotNil(t, cth)
			assert.Equal(t, "application/json", cth)
		})
	}
}

func TestCompressionMiddlewareBuilder(t *testing.T) {
	tests := map[string]struct {
		cm      *CmBuilder
		wantErr bool
		errMsg  string
	}{
		"no arguments provided":                       {cm: NewCompressionMiddleware(), wantErr: false},
		"gzip middleware":                             {cm: NewCompressionMiddleware().WithGZIP(), wantErr: false},
		"deflate middleware":                          {cm: NewCompressionMiddleware().WithDeflate(0), wantErr: false},
		"compress/lzw middleware":                     {cm: NewCompressionMiddleware().WithLZW(0, 8), wantErr: false},
		"deflate with wrong levels - too low":         {cm: NewCompressionMiddleware().WithDeflate(-3), wantErr: true, errMsg: "provided deflate level value not in the [-2, 9] range\n"},
		"deflate with wrong levels - too high":        {cm: NewCompressionMiddleware().WithDeflate(10), wantErr: true, errMsg: "provided deflate level value not in the [-2, 9] range\n"},
		"compress/lzw with wrong order - too low":     {cm: NewCompressionMiddleware().WithLZW(-1, 8), wantErr: true, errMsg: "provided lzw order value not valid\n"},
		"compress/lzw with wrong order - too high":    {cm: NewCompressionMiddleware().WithLZW(2, 8), wantErr: true, errMsg: "provided lzw order value not valid\n"},
		"compress/lzw with wrong litWidth - too low":  {cm: NewCompressionMiddleware().WithLZW(1, 1), wantErr: true, errMsg: "provided lzw litWidth value not in the [2, 8] range\n"},
		"compress/lzw with wrong litWidth - too high": {cm: NewCompressionMiddleware().WithLZW(1, 9), wantErr: true, errMsg: "provided lzw litWidth value not in the [2, 8] range\n"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mw, err := tc.cm.Build()

			if tc.wantErr {
				assert.Nil(t, mw)
				assert.EqualError(t, err, tc.errMsg)
			} else {
				assert.NotNil(t, mw)
				assert.Nil(t, err)
			}
		})
	}
}

func TestCompressionMiddlewareIgnoredRoutes(t *testing.T) {
	tests := map[string]struct {
		input []string
		want  []string
	}{
		"no routes":                             {input: []string{}, want: []string{}},
		"no trailing slashes":                   {input: []string{"/alive", "/ready"}, want: []string{"/alive", "/ready"}},
		"trailing slashes are removed":          {input: []string{"/alive/", "/ready/"}, want: []string{"/alive", "/ready"}},
		"with and without trailing slashes mix": {input: []string{"/alive////", "/ready/", "/metrics"}, want: []string{"/alive", "/ready", "/metrics"}},
		"exclude everything":                    {input: []string{"/"}, want: []string{"/"}},
		"multiple slashes only":                 {input: []string{"///"}, want: []string{"/"}},
		"nested paths":                          {input: []string{"/user/foo//", "/client/bar/", "/file"}, want: []string{"/user/foo", "/client/bar", "/file"}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmb := NewCompressionMiddleware().WithIgnoreRoutes(tc.input...)
			assert.Equal(t, tc.want, cmb.ignoreRoutes)
		})
	}
}
