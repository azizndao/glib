package middleware

import (
	"bufio"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/azizndao/grouter"
)

// Encoder is an interface that wraps the compression writer
type Encoder interface {
	io.Writer
	Close() error
	Flush() error
}

// EncoderFunc creates a new compression writer for the given writer and compression level
type EncoderFunc func(w io.Writer, level int) Encoder

// ioResetterWriter is an interface for encoders that can be reset and reused (poolable)
type ioResetterWriter interface {
	io.Writer
	Reset(w io.Writer)
}

// Compressor manages compression encoders and configuration
type Compressor struct {
	level              int
	poolSize           int
	encoders           map[string]EncoderFunc
	pools              map[string]*sync.Pool
	pooledEncoders     map[string]*sync.Pool
	encodingPrecedence []string
}

// NewCompressor creates a new compressor with the specified compression level
// Level should be between 0-9 where:
//   - 0 = no compression
//   - 1 = best speed
//   - 9 = best compression
//   - -1 = default compression
func NewCompressor(level int) *Compressor {
	c := &Compressor{
		level:              level,
		poolSize:           0,
		encoders:           make(map[string]EncoderFunc),
		pools:              make(map[string]*sync.Pool),
		pooledEncoders:     make(map[string]*sync.Pool),
		encodingPrecedence: make([]string, 0),
	}

	// Register default encoders (gzip has priority over deflate)
	c.SetEncoder("gzip", func(w io.Writer, level int) Encoder {
		gw, _ := gzip.NewWriterLevel(w, level)
		return gw
	})

	c.SetEncoder("deflate", func(w io.Writer, level int) Encoder {
		fw, _ := flate.NewWriter(w, level)
		return &flateEncoder{fw}
	})

	return c
}

// SetEncoder registers a custom encoder for the given encoding
// Common encodings: "gzip", "deflate", "br" (brotli)
//
// Example with brotli:
//
//	import "github.com/andybalholm/brotli"
//
//	compressor.SetEncoder("br", func(w io.Writer, level int) middleware.Encoder {
//	    return brotli.NewWriterLevel(w, level)
//	})
func (c *Compressor) SetEncoder(encoding string, fn EncoderFunc) {
	encoding = strings.ToLower(encoding)
	if encoding == "" {
		panic("the encoding cannot be empty")
	}
	if fn == nil {
		panic("attempted to set a nil encoder function")
	}

	// Clear existing entries for this encoding
	delete(c.pooledEncoders, encoding)
	delete(c.encoders, encoding)
	delete(c.pools, encoding)

	// Check if encoder supports Reset (can be pooled)
	encoder := fn(io.Discard, c.level)
	if _, ok := encoder.(ioResetterWriter); ok {
		// Create pool for resettable encoders
		pool := &sync.Pool{
			New: func() interface{} {
				return fn(io.Discard, c.level)
			},
		}
		c.pooledEncoders[encoding] = pool
	} else {
		// Non-poolable encoder
		c.encoders[encoding] = fn
	}

	// Update precedence list (newer encoders get priority)
	for i, v := range c.encodingPrecedence {
		if v == encoding {
			c.encodingPrecedence = append(c.encodingPrecedence[:i], c.encodingPrecedence[i+1:]...)
			break
		}
	}
	c.encodingPrecedence = append([]string{encoding}, c.encodingPrecedence...)
}

// Handler returns the compression middleware
func (c *Compressor) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder, encoding, cleanup := c.selectEncoder(r.Header, w)
		if encoder == nil {
			// No acceptable encoding found
			next.ServeHTTP(w, r)
			return
		}
		defer cleanup()

		// Set Vary header to inform caches that response varies by Accept-Encoding
		w.Header().Add("Vary", "Accept-Encoding")

		// Wrap the response writer
		cw := &compressResponseWriter{
			ResponseWriter: w,
			encoder:        encoder,
			encoding:       encoding,
		}
		defer cw.Close()

		next.ServeHTTP(cw, r)
	})
}

// Middleware returns a grouter-compatible middleware
func (c *Compressor) Middleware() grouter.Middleware {
	return func(next grouter.Handler) grouter.Handler {
		return func(ctx *grouter.Ctx) error {
			encoder, encoding, cleanup := c.selectEncoder(ctx.Request.Header, ctx.Response)
			if encoder == nil {
				// No acceptable encoding found
				return next(ctx)
			}
			defer cleanup()

			// Set Vary header
			ctx.Set("Vary", "Accept-Encoding")

			// Wrap the response writer
			cw := &compressResponseWriter{
				ResponseWriter: ctx.Response,
				encoder:        encoder,
				encoding:       encoding,
			}
			defer cw.Close()

			// Replace response writer
			originalWriter := ctx.Response
			ctx.Response = cw

			// Execute handler
			err := next(ctx)

			// Restore original writer
			ctx.Response = originalWriter

			return err
		}
	}
}

// selectEncoder picks the best encoder based on Accept-Encoding header
func (c *Compressor) selectEncoder(h http.Header, w io.Writer) (Encoder, string, func()) {
	acceptEncoding := h.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return nil, "", func() {}
	}

	// Parse Accept-Encoding header
	encodings := parseAcceptEncoding(acceptEncoding)

	// Find best match using precedence order
	for _, name := range c.encodingPrecedence {
		if contains(encodings, name) {
			// Check for pooled encoder first
			if pool, ok := c.pooledEncoders[name]; ok {
				encoder := pool.Get()
				if resetter, ok := encoder.(ioResetterWriter); ok {
					resetter.Reset(w)
					cleanup := func() {
						pool.Put(encoder)
					}
					if enc, ok := encoder.(Encoder); ok {
						return enc, name, cleanup
					}
					return resetter.(Encoder), name, cleanup
				}
			}

			// Fallback to non-pooled encoder
			if fn, ok := c.encoders[name]; ok {
				return fn(w, c.level), name, func() {}
			}
		}
	}

	// No encoder found to match the accepted encoding
	return nil, "", func() {}
}

// compressResponseWriter wraps http.ResponseWriter to provide compression
type compressResponseWriter struct {
	http.ResponseWriter
	encoder        Encoder
	encoding       string
	headerWritten  bool
	shouldCompress bool
}

func (cw *compressResponseWriter) WriteHeader(code int) {
	if cw.headerWritten {
		return
	}
	cw.headerWritten = true

	// Check if we should compress based on content type
	contentType := cw.ResponseWriter.Header().Get("Content-Type")
	cw.shouldCompress = isCompressible(contentType)

	if cw.shouldCompress {
		// Don't compress if already encoded
		if cw.ResponseWriter.Header().Get("Content-Encoding") != "" {
			cw.shouldCompress = false
		}
	}

	if cw.shouldCompress {
		cw.ResponseWriter.Header().Set("Content-Encoding", cw.encoding)
		cw.ResponseWriter.Header().Del("Content-Length")
	}

	cw.ResponseWriter.WriteHeader(code)
}

func (cw *compressResponseWriter) Write(p []byte) (int, error) {
	if !cw.headerWritten {
		cw.WriteHeader(http.StatusOK)
	}

	if cw.shouldCompress {
		return cw.encoder.Write(p)
	}

	return cw.ResponseWriter.Write(p)
}

func (cw *compressResponseWriter) Flush() {
	if cw.shouldCompress {
		cw.encoder.Flush()
	}

	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (cw *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := cw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("http.Hijacker not implemented")
}

func (cw *compressResponseWriter) Unwrap() http.ResponseWriter {
	return cw.ResponseWriter
}

func (cw *compressResponseWriter) Close() error {
	if cw.shouldCompress && cw.encoder != nil {
		return cw.encoder.Close()
	}
	return nil
}

// flateEncoder wraps flate.Writer to implement Encoder interface
type flateEncoder struct {
	*flate.Writer
}

func (fe *flateEncoder) Flush() error {
	return fe.Writer.Flush()
}

// parseAcceptEncoding parses the Accept-Encoding header
func parseAcceptEncoding(s string) []string {
	var encodings []string
	for _, enc := range strings.Split(s, ",") {
		enc = strings.TrimSpace(enc)
		// Remove quality value if present
		if idx := strings.Index(enc, ";"); idx != -1 {
			enc = enc[:idx]
		}
		enc = strings.TrimSpace(enc)
		if enc != "" && enc != "*" {
			encodings = append(encodings, strings.ToLower(enc))
		}
	}
	return encodings
}

// contains checks if a slice contains a string
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// isCompressible checks if the content type should be compressed
func isCompressible(contentType string) bool {
	if contentType == "" {
		return true
	}

	// Don't compress if already compressed
	ct := strings.ToLower(contentType)

	// Already compressed formats
	if strings.Contains(ct, "gzip") ||
		strings.Contains(ct, "zip") ||
		strings.Contains(ct, "compress") ||
		strings.HasPrefix(ct, "image/") && !strings.HasPrefix(ct, "image/svg") ||
		strings.HasPrefix(ct, "video/") ||
		strings.HasPrefix(ct, "audio/") {
		return false
	}

	// Compressible types
	compressible := []string{
		"text/",
		"application/json",
		"application/javascript",
		"application/xml",
		"application/x-javascript",
		"application/xhtml+xml",
		"application/rss+xml",
		"application/atom+xml",
		"application/ld+json",
		"image/svg+xml",
	}

	for _, prefix := range compressible {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}

	return false
}

// CompressConfig holds configuration for the Compress middleware
type CompressConfig struct {
	// Level is the compression level (0-9)
	// -1 = default compression
	// 0 = no compression
	// 1 = best speed
	// 9 = best compression
	// Default: gzip.DefaultCompression (-1)
	Level int

	// Encodings is a list of custom encoders to register
	// The map key is the encoding name (e.g., "br" for brotli)
	// Default: gzip and deflate are always registered
	Encodings map[string]EncoderFunc
}

// DefaultCompressConfig returns default compression configuration
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		Level:     gzip.DefaultCompression,
		Encodings: nil,
	}
}

// Compress creates a compression middleware with gzip/deflate support
//
// Example usage:
//
//	// Use default compression
//	router.Use(middleware.Compress())
//
//	// Custom compression level
//	router.Use(middleware.Compress(middleware.CompressConfig{
//	    Level: gzip.BestSpeed,
//	}))
//
//	// Add brotli support
//	import "github.com/andybalholm/brotli"
//	router.Use(middleware.Compress(middleware.CompressConfig{
//	    Level: gzip.BestCompression,
//	    Encodings: map[string]middleware.EncoderFunc{
//	        "br": func(w io.Writer, level int) middleware.Encoder {
//	            return brotli.NewWriterLevel(w, level)
//	        },
//	    },
//	}))
func Compress(config ...CompressConfig) grouter.Middleware {
	cfg := DefaultCompressConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	// Create compressor with configured level
	c := NewCompressor(cfg.Level)

	// Register custom encoders if provided
	for encoding, encoderFunc := range cfg.Encodings {
		c.SetEncoder(encoding, encoderFunc)
	}

	return c.Middleware()
}
