package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/azizndao/grouter"
)

// CompressConfig holds configuration for the Compress middleware
type CompressConfig struct {
	// Level is the compression level (0-9)
	// 0 = no compression, 1 = best speed, 9 = best compression
	// Default: gzip.DefaultCompression (-1)
	Level int

	// MinLength is the minimum response size to compress (in bytes)
	// Responses smaller than this will not be compressed
	// Default: 1024 (1KB)
	MinLength int

	// SkipFunc is a function that determines if compression should be skipped
	// Default: nil (compress all responses)
	SkipFunc func(*grouter.Ctx) bool
}

// compressWriter wraps http.ResponseWriter to provide gzip compression
type compressWriter struct {
	http.ResponseWriter
	writer         io.Writer
	gzipWriter     *gzip.Writer
	config         CompressConfig
	headerWritten  bool
	shouldCompress bool
}

func (cw *compressWriter) WriteHeader(code int) {
	cw.headerWritten = true

	// Check if we should compress based on content type
	contentType := cw.ResponseWriter.Header().Get("Content-Type")
	cw.shouldCompress = isCompressibleContentType(contentType)

	// If we should compress, set the header
	if cw.shouldCompress {
		cw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		cw.ResponseWriter.Header().Del("Content-Length")
	}

	cw.ResponseWriter.WriteHeader(code)
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if !cw.headerWritten {
		cw.WriteHeader(http.StatusOK)
	}

	// If compression is enabled, use gzip writer
	if cw.shouldCompress {
		return cw.gzipWriter.Write(b)
	}

	// Otherwise, write directly
	return cw.ResponseWriter.Write(b)
}

// DefaultCompressConfig returns default configuration for compression
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 1024, // 1KB
		SkipFunc:  nil,
	}
}

// Compress creates a middleware that compresses HTTP responses using gzip.
// It only compresses responses that:
// - Are larger than the configured minimum length
// - Have a compressible content type (text/*, application/json, etc.)
// - Client supports gzip encoding (Accept-Encoding header)
//
// Example usage:
//
//	// Use default compression
//	router.Use(middleware.Compress())
//
//	// Custom configuration
//	router.Use(middleware.Compress(middleware.CompressConfig{
//	    Level:     gzip.BestCompression,
//	    MinLength: 2048, // 2KB
//	    SkipFunc: func(c *grouter.Ctx) bool {
//	        // Skip compression for images
//	        return strings.HasPrefix(c.Path(), "/images")
//	    },
//	}))
func Compress(config ...CompressConfig) grouter.Middleware {
	cfg := DefaultCompressConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Check if client supports gzip
			if !strings.Contains(c.Get("Accept-Encoding"), "gzip") {
				return next(c)
			}

			// Check if we should skip compression
			if cfg.SkipFunc != nil && cfg.SkipFunc(c) {
				return next(c)
			}

			// Create gzip writer
			gzipWriter, err := gzip.NewWriterLevel(c.Response, cfg.Level)
			if err != nil {
				return next(c)
			}
			defer gzipWriter.Close()

			// Wrap response writer
			cw := &compressWriter{
				ResponseWriter: c.Response,
				writer:         gzipWriter,
				gzipWriter:     gzipWriter,
				config:         cfg,
			}

			// Replace response writer
			originalWriter := c.Response
			c.Response = cw

			// Execute handler
			err = next(c)

			// Restore original writer
			c.Response = originalWriter

			return err
		}
	}
}

// isCompressibleContentType checks if a content type should be compressed
func isCompressibleContentType(contentType string) bool {
	// Empty content type - compress by default
	if contentType == "" {
		return true
	}

	// List of compressible content types
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

	contentType = strings.ToLower(contentType)
	for _, prefix := range compressible {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}

	return false
}
