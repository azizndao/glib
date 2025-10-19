package middleware

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/azizndao/grouter"
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
		Format:     LogFormatDefault,
		TimeFormat: "15:04:05",
		Output:     os.Stdout,
		Skip:       nil,
	}
}

// Logger creates a logging middleware with custom configuration
func Logger(config ...LoggerConfig) grouter.Middleware {
	cfg := DefaultLoggerConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
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

			// Log the request
			logRequest(cfg, c.Request, wrapped.statusCode, wrapped.size, duration)

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

// Convenience functions for different log formats

// LoggerTiny returns a tiny logger middleware
func LoggerTiny() grouter.Middleware {
	config := DefaultLoggerConfig()
	config.Format = LogFormatTiny
	return Logger(config)
}

// LoggerShort returns a short logger middleware
func LoggerShort() grouter.Middleware {
	config := DefaultLoggerConfig()
	config.Format = LogFormatShort
	return Logger(config)
}

// LoggerCombined returns a combined logger middleware
func LoggerCombined() grouter.Middleware {
	config := DefaultLoggerConfig()
	config.Format = LogFormatCombined
	return Logger(config)
}

// LoggerDefault returns a default logger middleware
func LoggerDefault() grouter.Middleware {
	return Logger(DefaultLoggerConfig())
}
