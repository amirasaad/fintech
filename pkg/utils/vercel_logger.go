package utils

import (
	"log/slog"
	"os"
	"time"
)

// VercelLogger provides logging utilities optimized for Vercel deployment
type VercelLogger struct {
	logger *slog.Logger
}

// NewVercelLogger creates a new Vercel-optimized logger
func NewVercelLogger() *VercelLogger {
	// Use stderr for Vercel visibility
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     getLogLevel(),
		AddSource: true,
	})

	return &VercelLogger{
		logger: slog.New(handler),
	}
}

// getLogLevel determines the appropriate log level for the environment
func getLogLevel() slog.Level {
	env := os.Getenv("VERCEL_ENV")
	if env == "production" {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}

// Info logs an info message with Vercel-optimized formatting
func (v *VercelLogger) Info(msg string, args ...any) {
	v.logger.Info(msg, args...)
}

// Error logs an error message with Vercel-optimized formatting
func (v *VercelLogger) Error(msg string, args ...any) {
	v.logger.Error(msg, args...)
}

// Warn logs a warning message with Vercel-optimized formatting
func (v *VercelLogger) Warn(msg string, args ...any) {
	v.logger.Warn(msg, args...)
}

// Debug logs a debug message with Vercel-optimized formatting
func (v *VercelLogger) Debug(msg string, args ...any) {
	v.logger.Debug(msg, args...)
}

// LogService logs a service operation with Vercel context
func (v *VercelLogger) LogService(service, operation string, args ...any) {
	allArgs := append([]any{
		"service", service,
		"operation", operation,
		"timestamp", time.Now(),
		"environment", os.Getenv("VERCEL_ENV"),
	}, args...)
	v.logger.Info("Service Operation", allArgs...)
}

// LogError logs an error with Vercel context
func (v *VercelLogger) LogError(service, operation string, err error, args ...any) {
	allArgs := append([]any{
		"service", service,
		"operation", operation,
		"error", err,
		"timestamp", time.Now(),
		"environment", os.Getenv("VERCEL_ENV"),
	}, args...)
	v.logger.Error("Service Error", allArgs...)
}

// LogHTTP logs an HTTP request with Vercel context
func (v *VercelLogger) LogHTTP(method, path, ip, userAgent string, status int, duration time.Duration) {
	v.logger.Info("HTTP Request",
		"method", method,
		"path", path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"ip", ip,
		"user_agent", userAgent,
		"timestamp", time.Now(),
		"environment", os.Getenv("VERCEL_ENV"),
	)
}
