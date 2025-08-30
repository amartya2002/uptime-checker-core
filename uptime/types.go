// Package uptime defines core types for the uptime checker.
package uptime

import "time"

type LogLevel int

const (
    LogNone LogLevel = iota // no logs
    LogError                // only errors
    LogInfo                 // info + errors
    LogDebug                // verbose
)

type Endpoint struct {
    ID             string        `json:"id"`
    Name           string        `json:"name"`
    URL            string        `json:"url"`
    Method         string        `json:"method"`
    Frequency      time.Duration `json:"frequency"`
    ExpectedStatus int           `json:"expected_status,omitempty"`
}

// Result represents the outcome of a check
type Result struct {
    Endpoint   Endpoint      `json:"endpoint"`
    Timestamp  time.Time     `json:"timestamp"`
    StatusCode int           `json:"status_code"`
    Latency    time.Duration `json:"latency"`
    Success    bool          `json:"success"`
    Error      string        `json:"error,omitempty"`
}

type Job struct {
    Endpoint Endpoint
    RunAt    time.Time
}
