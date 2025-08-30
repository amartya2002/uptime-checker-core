// Package uptime exposes configuration options for the Checker via a
// functional options API.
package uptime

import (
    "fmt"
    "time"

    "go.uber.org/zap"
)

// ===== Options Pattern =====
type Option func(*Checker)

func WithWorkers(n int) Option {
    return func(c *Checker) { c.numWorkers = n }
}

func WithTimeout(d time.Duration) Option {
    return func(c *Checker) { c.httpClient.Timeout = d }
}

func WithLogLevel(level LogLevel) Option {
    return func(c *Checker) { c.logLevel = level }
}

func WithResultBuffer(size int) Option {
    return func(c *Checker) { c.results = make(chan Result, size) }
}

// enable/disable internal logs
func WithInternalLogs(enabled bool) Option {
    return func(c *Checker) { c.enableInternalLogs = enabled }
}

// WithZapLogger sets up a zap logger. If filePath is empty, logs to console.
func WithZapLogger(filePath string) Option {
    return func(c *Checker) {
        var err error
        if filePath != "" {
            cfg := zap.NewProductionConfig()
            cfg.OutputPaths = []string{"stdout", filePath}
            c.logger, err = cfg.Build()
        } else {
            c.logger, err = zap.NewProduction(zap.AddCallerSkip(1))
        }
        if err != nil {
            panic(fmt.Sprintf("Failed to initialize Zap logger: %v", err))
        }
    }
}

// WithLogger allows injecting a custom zap logger (useful in tests).
func WithLogger(l *zap.Logger) Option {
    return func(c *Checker) { c.logger = l }
}

// WithLogRetention sets the max number of in-memory logs kept per endpoint.
func WithLogRetention(n int) Option {
    return func(c *Checker) { if n > 0 { c.logRetention = n } }
}
