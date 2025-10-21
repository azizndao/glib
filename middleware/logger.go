package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/azizndao/grouter/router"
	"github.com/azizndao/grouter/util"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Gray   = "\033[90m"
)

// LoggerConfig holds configuration for the logger
type LoggerConfig struct {
	Format     LogFormat
	TimeFormat string
	Output     *os.File
	Skip       func(*http.Request) bool

	// UseStructuredLogging enables structured logging with slog
	// When enabled, logs are written using slog instead of colored console output
	UseStructuredLogging bool

	// Logger is the slog.Logger instance to use for structured logging
	// If nil, the default logger is used
	Logger *slog.Logger

	// LogLevel determines which requests to log
	// Info: all requests, Warn: 4xx and 5xx, Error: 5xx only
	LogLevel slog.Level
}

// LogFormat defines the format of log output
type LogFormat string

const (
	LogFormatDefault  LogFormat = "default"
	LogFormatCombined LogFormat = "combined"
	LogFormatShort    LogFormat = "short"
	LogFormatTiny     LogFormat = "tiny"
)

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Format:               LogFormatDefault,
		TimeFormat:           "15:04:05",
		Output:               os.Stdout,
		Skip:                 nil,
		UseStructuredLogging: false,
		Logger:               nil,
		LogLevel:             slog.LevelInfo,
	}
}

// LoadLoggerConfig loads LoggerConfig from environment variables
// Environment variables:
//   - ENABLE_LOGGER (bool): enable/disable logger middleware (default: true)
//   - IS_DEBUG (bool): determines logging mode (default: false)
//     When IS_DEBUG=false: uses structured JSON logging (production mode)
//     When IS_DEBUG=true: uses colorful console logging (development mode)
//   - LOGGER_FORMAT (string): log format for console logging - options: default, combined, short, tiny (default: default)
//     Note: Only applies when IS_DEBUG=true. Ignored in production mode.
//   - LOGGER_TIME_FORMAT (string): time format string in Go layout format (default: "15:04:05")
//     Example: "2006-01-02 15:04:05" for full date/time
//     Note: Only applies when IS_DEBUG=true. Ignored in production mode.
//
// Returns nil if ENABLE_LOGGER=false, otherwise returns config
func LoadLoggerConfig() *LoggerConfig {
	if !util.GetEnvBool("ENABLE_LOGGER", true) {
		return nil
	}

	cfg := DefaultLoggerConfig()

	// Use IS_DEBUG to determine if we should use structured logging
	// IS_DEBUG=false means production mode (structured/JSON logging)
	// IS_DEBUG=true means development mode (colorful console logging)
	isDebug := util.GetEnvBool("IS_DEBUG", false)
	cfg.UseStructuredLogging = !isDebug

	// Load format and time format (only used when IS_DEBUG=true)
	formatStr := util.GetEnvLogFormat("LOGGER_FORMAT", string(cfg.Format))
	cfg.Format = LogFormat(formatStr)

	cfg.TimeFormat = util.GetEnv("LOGGER_TIME_FORMAT", cfg.TimeFormat)

	return &cfg
}

// Logger creates a logging middleware with custom configuration
//
// Example usage:
//
//	// Use default colored console logging
//	router.Use(middleware.Logger())
//
//	// Custom format
//	router.Use(middleware.Logger(middleware.LoggerConfig{
//	    Format: middleware.LogFormatTiny,
//	}))
//
//	// Structured logging with slog
//	router.Use(middleware.Logger(middleware.LoggerConfig{
//	    UseStructuredLogging: true,
//	    Logger: slog.Default(),
//	    LogLevel: slog.LevelInfo,
//	}))
func Logger(config ...LoggerConfig) router.Middleware {
	cfg := DefaultLoggerConfig()
	if len(config) > 0 {
		// Merge provided config with defaults
		provided := config[0]

		if provided.Format != "" {
			cfg.Format = provided.Format
		}
		if provided.TimeFormat != "" {
			cfg.TimeFormat = provided.TimeFormat
		}
		if provided.Output != nil {
			cfg.Output = provided.Output
		}
		if provided.Skip != nil {
			cfg.Skip = provided.Skip
		}
		// UseStructuredLogging is a bool, so we need to check if it was explicitly set
		// We'll accept the provided value since false is a valid setting
		cfg.UseStructuredLogging = provided.UseStructuredLogging

		if provided.Logger != nil {
			cfg.Logger = provided.Logger
		}
		if provided.LogLevel != 0 {
			cfg.LogLevel = provided.LogLevel
		}
	}

	// If structured logging is enabled, set up slog logger
	if cfg.UseStructuredLogging && cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return func(next router.Handler) router.Handler {
		return func(c *router.Ctx) error {
			// Skip if skip function returns true
			if cfg.Skip != nil && cfg.Skip(c.Request) {
				return next(c)
			}

			start := time.Now()

			// Create a response writer wrapper to capture status code and size
			wrapped := &responseWriter{
				ResponseWriter: c.Response,
				statusCode:     200,
				size:           0,
			}

			// Replace the response writer in context
			originalWriter := c.Response
			c.Response = wrapped

			// Process request
			err := next(c)

			// Restore original writer
			c.Response = originalWriter

			// Calculate duration
			duration := time.Since(start)

			// Log the request based on configuration
			if cfg.UseStructuredLogging {
				logStructuredRequest(cfg, c, wrapped.statusCode, wrapped.size, duration)
			} else {
				logRequest(cfg, c.Request, wrapped.statusCode, wrapped.size, duration)
			}

			return err
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status and size
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	size          int
	headerWritten bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.headerWritten {
		return // Prevent multiple WriteHeader calls
	}
	rw.headerWritten = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// logRequest logs the HTTP request with colors based on status code
func logRequest(cfg LoggerConfig, r *http.Request, status, size int, duration time.Duration) {
	timestamp := time.Now().Format(cfg.TimeFormat)

	// Color based on status code
	statusColor := getStatusColor(status)
	methodColor := getMethodColor(r.Method)

	var logLine string

	switch cfg.Format {
	case LogFormatTiny:
		logLine = fmt.Sprintf("%s%s%s %s%s%s %s%d%s %s%s%s\n",
			Gray, timestamp, Reset,
			methodColor, r.Method, Reset,
			statusColor, status, Reset,
			Cyan, formatDuration(duration), Reset,
		)

	case LogFormatShort:
		logLine = fmt.Sprintf("%s[%s]%s %s%s%s %s%s%s %s%d%s %s%s%s\n",
			Gray, timestamp, Reset,
			methodColor, r.Method, Reset,
			White, r.URL.Path, Reset,
			statusColor, status, Reset,
			Cyan, formatDuration(duration), Reset,
		)

	case LogFormatCombined:
		userAgent := r.Header.Get("User-Agent")
		if len(userAgent) > 50 {
			userAgent = userAgent[:50] + "..."
		}

		logLine = fmt.Sprintf("%s[%s]%s %s%s%s %s%s%s %s%d%s %s%dB%s %s%s%s \"%s%s%s\"\n",
			Gray, timestamp, Reset,
			methodColor, r.Method, Reset,
			White, r.URL.RequestURI(), Reset,
			statusColor, status, Reset,
			Purple, size, Reset,
			Cyan, formatDuration(duration), Reset,
			Gray, userAgent, Reset,
		)

	default: // LogFormatDefault
		logLine = fmt.Sprintf("%s[%s]%s %s%-6s%s %s%-50s%s %s%3d%s %s%8s%s %s%6dB%s\n",
			Gray, timestamp, Reset,
			methodColor, r.Method, Reset,
			White, truncate(r.URL.RequestURI(), 50), Reset,
			statusColor, status, Reset,
			Cyan, formatDuration(duration), Reset,
			Purple, size, Reset,
		)
	}

	fmt.Fprint(cfg.Output, logLine)
}

// getStatusColor returns color based on HTTP status code
func getStatusColor(status int) string {
	switch {
	case status >= 200 && status < 300:
		return Green
	case status >= 300 && status < 400:
		return Cyan
	case status >= 400 && status < 500:
		return Yellow
	case status >= 500:
		return Red
	default:
		return White
	}
}

// getMethodColor returns color based on HTTP method
func getMethodColor(method string) string {
	switch method {
	case "GET":
		return Blue
	case "POST":
		return Green
	case "PUT":
		return Yellow
	case "PATCH":
		return Purple
	case "DELETE":
		return Red
	case "HEAD", "OPTIONS":
		return Cyan
	default:
		return White
	}
}

// formatDuration formats duration for display
func formatDuration(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	case d >= time.Microsecond:
		return fmt.Sprintf("%.0fμs", float64(d.Nanoseconds())/1e3)
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}

// truncate truncates a string to specified length
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// logStructuredRequest logs the request using structured logging (slog)
func logStructuredRequest(cfg LoggerConfig, c *router.Ctx, status, size int, duration time.Duration) {
	requestID := GetRequestID(c)

	// Determine log level based on status code
	logLevel := cfg.LogLevel
	if status >= 500 {
		logLevel = slog.LevelError
	} else if status >= 400 {
		logLevel = slog.LevelWarn
	}

	// Only log if level is appropriate
	if logLevel >= cfg.LogLevel {
		// Build log attributes
		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"size", size,
			"remote_addr", c.IP(),
		}

		if requestID != "" {
			attrs = append(attrs, "request_id", requestID)
		}

		if c.UserAgent() != "" {
			attrs = append(attrs, "user_agent", c.UserAgent())
		}

		cfg.Logger.Log(c.Context(), logLevel, "HTTP request", attrs...)
	}
}
