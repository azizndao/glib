package middleware

import (
	"compress/gzip"

	"github.com/azizndao/glib/util"
)

// CompressConfig holds configuration for the Compress middleware
type CompressConfig struct {
	// Level is the compression level (0-9)
	// -1 = default compression
	// 0 = no compression
	// 1 = best speed
	// 9 = best compression
	// Default: gzip.DefaultCompression (-1)
	Level int
}

// DefaultCompressConfig returns default compression configuration
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		Level: gzip.DefaultCompression,
	}
}

// LoadCompressConfig loads CompressConfig from environment variables
// Environment variable: ENABLE_COMPRESS (bool)
// Returns nil if ENABLE_COMPRESS=false, otherwise returns default config
func LoadCompressConfig() *CompressConfig {
	if !util.GetEnvBool("ENABLE_COMPRESS", true) {
		return nil
	}

	cfg := DefaultCompressConfig()
	return &cfg
}
