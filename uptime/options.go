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

// Deprecated: prefer LogConsole/LogFile/DisableLogs.
// WithZapLogger sets up a logger. If filePath is empty, logs to console.
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
        c.loggerExplicit = true
    }
}

// WithLogger allows injecting a custom zap logger (useful in tests).
func WithLogger(l *zap.Logger) Option {
    return func(c *Checker) { c.logger = l; c.loggerExplicit = true }
}

// WithLogRetention sets the max number of in-memory logs kept per endpoint.
func WithLogRetention(n int) Option {
    return func(c *Checker) { if n > 0 { c.logRetention = n } }
}

// Log configures outputs in a single call.
// Values: "console" (stdout), "none" (disable), or one/more file paths.
func Log(outputs ...string) Option {
    return func(c *Checker) {
        // Reset config
        c.logConsoleOpt = nil
        c.logFilesOpt = nil
        c.logDisableOpt = false
        if len(outputs) == 0 {
            // default console
            b := true
            c.logConsoleOpt = &b
            return
        }
        for _, o := range outputs {
            switch o {
            case "none":
                c.logDisableOpt = true
            case "console":
                b := true
                c.logConsoleOpt = &b
            default:
                c.logFilesOpt = append(c.logFilesOpt, o)
            }
        }
    }
}

// LogConsole enables/disables console logging (stdout).
func LogConsole(enabled bool) Option {
    return func(c *Checker) {
        c.logConsoleOpt = &enabled
    }
}

// LogFile adds a file path to log outputs. Can be used multiple times.
func LogFile(path string) Option {
    return func(c *Checker) {
        if path != "" {
            c.logFilesOpt = append(c.logFilesOpt, path)
        }
    }
}

// DisableLogs disables logging entirely.
func DisableLogs() Option {
    return func(c *Checker) {
        c.logDisableOpt = true
    }
}
