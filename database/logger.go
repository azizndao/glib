package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/logger"
)

// GormLogger adapts slog.Logger to GORM's logger interface.
type GormLogger struct {
	logger        *slog.Logger
	slowThreshold time.Duration
	logLevel      logger.LogLevel
}

// NewGormLogger creates a new GORM logger using slog.
func NewGormLogger(log *slog.Logger, slowThreshold time.Duration) *GormLogger {
	return &GormLogger{
		logger:        log,
		slowThreshold: slowThreshold,
		logLevel:      logger.Info,
	}
}

// LogMode sets the log level.
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.logLevel = level
	return &newLogger
}

// Info logs info level messages.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Info {
		l.logger.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn logs warning level messages.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Warn {
		l.logger.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error logs error level messages.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.logLevel >= logger.Error {
		l.logger.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL queries with execution time.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.logLevel >= logger.Error:
		// Log errors
		l.logger.ErrorContext(ctx, "SQL Error",
			"error", err,
			"elapsed", elapsed,
			"rows", rows,
			"sql", sql,
		)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.logLevel >= logger.Warn:
		// Log slow queries
		l.logger.WarnContext(ctx, "Slow SQL",
			"elapsed", elapsed,
			"threshold", l.slowThreshold,
			"rows", rows,
			"sql", sql,
		)
	case l.logLevel == logger.Info:
		// Log all queries in info mode
		l.logger.InfoContext(ctx, "SQL",
			"elapsed", elapsed,
			"rows", rows,
			"sql", sql,
		)
	}
}
