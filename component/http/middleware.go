package http

import (
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"

	"github.com/beatlabs/patron/component/http/auth"
	"github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	tracinglog "github.com/opentracing/opentracing-go/log"
)

const (
	serverComponent = "http-server"
	fieldNameError  = "error"
	gzipHeader      = "gzip"
	deflateHeader   = "deflate"
	lzwHeader       = "compress"
)

type responseWriter struct {
	status              int
	statusHeaderWritten bool
	payload             []byte
	writer              http.ResponseWriter
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{status: -1, statusHeaderWritten: false, writer: w}
}

// Status returns the http response status.
func (w *responseWriter) Status() int {
	return w.status
}

// Header returns the Header.
func (w *responseWriter) Header() http.Header {
	return w.writer.Header()
}

// Write to the internal responseWriter and sets the status if not set already.
func (w *responseWriter) Write(d []byte) (int, error) {

	value, err := w.writer.Write(d)
	if err != nil {
		return value, err
	}

	w.payload = d

	if !w.statusHeaderWritten {
		w.status = http.StatusOK
		w.statusHeaderWritten = true
	}

	return value, err
}

// WriteHeader writes the internal Header and saves the status for retrieval.
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.writer.WriteHeader(code)
	w.statusHeaderWritten = true
}

// MiddlewareFunc type declaration of middleware func.
type MiddlewareFunc func(next http.Handler) http.Handler

// NewRecoveryMiddleware creates a MiddlewareFunc that ensures recovery and no panic.
func NewRecoveryMiddleware() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch x := r.(type) {
					case string:
						err = errors.New(x)
					case error:
						err = x
					default:
						err = errors.New("unknown panic")
					}
					_ = err
					log.Errorf("recovering from an error: %v: %s", err, string(debug.Stack()))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// NewAuthMiddleware creates a MiddlewareFunc that implements authentication using an Authenticator.
func NewAuthMiddleware(auth auth.Authenticator) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authenticated, err := auth.Authenticate(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			if !authenticated {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewLoggingTracingMiddleware creates a MiddlewareFunc that continues a tracing span and finishes it.
// It also logs the HTTP request on debug logging level
func NewLoggingTracingMiddleware(path string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			corID := getOrSetCorrelationID(r.Header)
			sp, r := span(path, corID, r)
			lw := newResponseWriter(w)
			next.ServeHTTP(lw, r)
			finishSpan(sp, lw.Status(), lw.payload)
			logRequestResponse(corID, lw, r)
		})
	}
}

type compressionResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// CompressionMiddewareBuilder holds the required parameters for building a compression middleware.
type CompressionMiddewareBuilder struct {
	ignoreRoutes []string
	deflateLevel int
	lzwOrder     lzw.Order
	lzwLitWidth  int
	errors       []error
}

// ignore checks if the given url ignored from compression or not.
func (c *CompressionMiddewareBuilder) ignore(url string) bool {
	for _, iURL := range c.ignoreRoutes {
		if strings.HasPrefix(url, iURL) {
			return true
		}
	}

	return false
}

// NewCompressionMiddleware initializes the builder for a compression middleware.
// As per Section 3.5 of the HTTP/1.1 RFC, we support GZIP, Deflate and LZW as compression methods
// https://tools.ietf.org/html/rfc2616#section-3.5
func NewCompressionMiddleware() *CompressionMiddewareBuilder {
	return &CompressionMiddewareBuilder{
		deflateLevel: 8,
		lzwOrder:     0,
		lzwLitWidth:  8,
	}
}

// SetDeflateLevel sets the level of compression for Deflate; based on https://golang.org/pkg/compress/flate/
// Levels range from 1 (BestSpeed) to 9 (BestCompression); higher levels typically run slower but compress more.
// Level 0 (NoCompression) does not attempt any compression; it only adds the necessary DEFLATE framing.
// Level -1 (DefaultCompression) uses the default compression level.
// Level -2 (HuffmanOnly) will use Huffman compression only, giving a very fast compression for all types of input, but sacrificing considerable compression efficiency.
func (c *CompressionMiddewareBuilder) SetDeflateLevel(level int) *CompressionMiddewareBuilder {
	if level < -2 || level > 9 {
		c.errors = append(c.errors, errors.New("provided deflate level value not in the [-2, 9] range"))
	} else {
		c.deflateLevel = level
	}
	return c
}

// SetLZWParams sets the Order and LitWidth parameters for LZW compression; based on LZW and https://golang.org/pkg/compress/lzw/
// Order 0 uses the Least Significant Bits first (as in GIF files),
// while Order 1 uses the Most Significant Bits first (as in TIFF and PDF files).
// The LitWidth defines the number of bits to use for literal codes, and must be in the [2, 8] range
// The input size must be less than 1<<litWidth.
func (c *CompressionMiddewareBuilder) SetLZWParams(order lzw.Order, litWidth int) *CompressionMiddewareBuilder {
	if order != 0 && order != 1 {
		c.errors = append(c.errors, errors.New("provided lzw order value not valid"))
	} else {
		c.lzwOrder = order
	}

	if litWidth < 2 || litWidth > 8 {
		c.errors = append(c.errors, errors.New("provided lzw litWidth value not in the [2, 8] range"))
	} else {
		c.lzwLitWidth = litWidth
	}
	return c
}

// Write provides write func to the writer.
func (w compressionResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// WithIgnoreRoutes specifies which routes should be excluded from compression
// Any trailing slashes are trimmed, so we match both /metrics/ and /metrics?seconds=30
func (c *CompressionMiddewareBuilder) WithIgnoreRoutes(r ...string) *CompressionMiddewareBuilder {
	res := make([]string, 0, len(r))
	for _, e := range r {
		for len(e) > 1 && e[len(e)-1] == '/' {
			e = e[0 : len(e)-1]
		}
		res = append(res, e)
	}
	c.ignoreRoutes = res

	return c
}

// Build initializes the MiddlewareFunc from the gathered properties.
func (c *CompressionMiddewareBuilder) Build() (MiddlewareFunc, error) {
	if len(c.errors) > 0 {
		return nil, patronErrors.Aggregate(c.errors...)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hdr := r.Header.Get(encoding.AcceptEncodingHeader)

			if !isCompressionHeader(hdr) || c.ignore(r.URL.String()) {
				next.ServeHTTP(w, r)
				log.Debugf("url %s skipped from compression middleware", r.URL.String())
				return
			}
			// explicitly specify encoding in header
			w.Header().Set(encoding.ContentEncodingHeader, hdr)

			// keep content type intact
			respHeader := r.Header.Get(encoding.ContentTypeHeader)
			if respHeader != "" {
				w.Header().Set(encoding.ContentTypeHeader, respHeader)
			}

			var cw io.WriteCloser
			var err error
			switch hdr {
			case gzipHeader:
				cw = gzip.NewWriter(w)
			case deflateHeader:
				cw, err = flate.NewWriter(w, c.deflateLevel)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
			case lzwHeader:
				cw = lzw.NewWriter(w, c.lzwOrder, c.lzwLitWidth)
			default:
				next.ServeHTTP(w, r)
				return
			}

			defer func(cw io.WriteCloser) {
				err := cw.Close()
				if err != nil {
					log.Errorf("error in deferred call to Close() method on %v compression middleware : %v", hdr, err.Error())
				}
			}(cw)

			crw := compressionResponseWriter{Writer: cw, ResponseWriter: w}
			next.ServeHTTP(crw, r)
			log.Debugf("url %s used with %s compression method", r.URL.String(), hdr)
		})
	}, nil
}

// NewCachingMiddleware creates a cache layer as a middleware
// when used as part of a middleware chain any middleware later in the chain,
// will not be executed, but the headers it appends will be part of the cache
func NewCachingMiddleware(rc *cache.RouteCache) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			err := cache.Handler(w, r, rc, next)
			if err != nil {
				log.Errorf("error encountered in the caching middleware: %v", err)
				return
			}
		})
	}
}

// MiddlewareChain chains middlewares to a handler func.
func MiddlewareChain(f http.Handler, mm ...MiddlewareFunc) http.Handler {
	for i := len(mm) - 1; i >= 0; i-- {
		f = mm[i](f)
	}
	return f
}

func isCompressionHeader(h string) bool {
	return strings.Contains(h, "gzip") || strings.Contains(h, "deflate") || strings.Contains(h, "compress")
}

func logRequestResponse(corID string, w *responseWriter, r *http.Request) {
	if !log.Enabled(log.DebugLevel) {
		return
	}

	remoteAddr := r.RemoteAddr
	if i := strings.LastIndex(remoteAddr, ":"); i != -1 {
		remoteAddr = remoteAddr[:i]
	}

	info := map[string]interface{}{
		"request": map[string]interface{}{
			"remote-address": remoteAddr,
			"method":         r.Method,
			"url":            r.URL,
			"proto":          r.Proto,
			"status":         w.Status(),
			"referer":        r.Referer(),
			"user-agent":     r.UserAgent(),
			correlation.ID:   corID,
		},
	}
	log.Sub(info).Debug()
}

func getOrSetCorrelationID(h http.Header) string {
	cor, ok := h[correlation.HeaderID]
	if !ok {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if len(cor) == 0 {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	if cor[0] == "" {
		corID := uuid.New().String()
		h.Set(correlation.HeaderID, corID)
		return corID
	}
	return cor[0]
}

func span(path, corID string, r *http.Request) (opentracing.Span, *http.Request) {
	ctx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	if err != nil && err != opentracing.ErrSpanContextNotFound {
		log.Errorf("failed to extract HTTP span: %v", err)
	}

	strippedPath, err := stripQueryString(path)
	if err != nil {
		log.Warnf("unable to strip query string %q: %v", path, err)
		strippedPath = path
	}

	sp := opentracing.StartSpan(opName(r.Method, strippedPath), ext.RPCServerOption(ctx))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.HTTPUrl.Set(sp, r.URL.String())
	ext.Component.Set(sp, serverComponent)
	sp.SetTag(trace.VersionTag, trace.Version)
	sp.SetTag(correlation.ID, corID)
	return sp, r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
}

// stripQueryString returns a path without the query string
func stripQueryString(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	if len(u.RawQuery) == 0 {
		return path, nil
	}

	return path[:len(path)-len(u.RawQuery)-1], nil
}

func finishSpan(sp opentracing.Span, code int, payload []byte) {
	ext.HTTPStatusCode.Set(sp, uint16(code))
	isError := code >= http.StatusInternalServerError
	if isError && len(payload) != 0 {
		sp.LogFields(tracinglog.String(fieldNameError, string(payload)))
	}
	ext.Error.Set(sp, isError)
	sp.Finish()
}

func opName(method, path string) string {
	return method + " " + path
}
