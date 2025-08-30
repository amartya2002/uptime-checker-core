# Uptime Checker

Concurrent, configurable HTTP uptime checker with a worker pool, simple scheduling, structured logging, and real‑time results streaming.

</div>

---

## Features

- Concurrent worker pool for high throughput checks
- Simple interval scheduler per endpoint
- Structured logging via `zap` with pluggable logger
- In‑memory logs per endpoint with retention cap
- Stream results via a channel for dashboards or alerts
- Functional options to configure timeouts, workers, log level, buffers
- Load endpoints from JSON file

## Install


```bash
go get github.com/amartya2002/uptime-checker-core/uptime
```

In code:

```go
import "github.com/amartya2002/uptime-checker-core/uptime"
```


## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/YOUR_GITHUB_USERNAME/uptime-checker-core/uptime"
)

func main() {
    checker := uptime.New(
        uptime.WithWorkers(50),
        uptime.WithTimeout(10*time.Second),
        uptime.WithLogLevel(uptime.LogInfo),
        uptime.WithLogRetention(200), // keep last 200 logs per endpoint
    )
    checker.Start()
    defer checker.Stop()

    checker.AddSite(uptime.Endpoint{
        ID:             "google",
        Name:           "Google",
        URL:            "https://google.com",
        Method:         "GET",
        Frequency:      30 * time.Second,
        ExpectedStatus: 200,
    })

    go func() {
        for res := range checker.Results() {
            fmt.Printf("[%s] %s -> %d (ok=%v, %v)\n",
                res.Timestamp.Format(time.RFC3339), res.Endpoint.Name, res.StatusCode, res.Success, res.Latency)
        }
    }()

    select {}
}
```

## Loading From JSON

```json
[
  {"id":"s1","name":"Google","url":"https://google.com","method":"GET","frequency":30,"expected_status":200},
  {"id":"s2","name":"HTTPBin","url":"https://httpbin.org/status/204","method":"GET","frequency":15,"expected_status":204}
]
```

```go
if err := checker.LoadFromFile("endpoints.json"); err != nil {
    // handle error
}
```

Note: `frequency` in the JSON is in seconds and is converted to `time.Duration` internally.

## Options Summary

- `WithWorkers(n int)`: number of worker goroutines
- `WithTimeout(d time.Duration)`: HTTP client timeout
- `WithLogLevel(level LogLevel)`: `LogNone`, `LogError`, `LogInfo`, `LogDebug`
- `WithResultBuffer(size int)`: channel buffer for results
- `WithInternalLogs(enabled bool)`: internal lifecycle logs
- `WithZapLogger(filePath string)`: file + console zap logger
- `WithLogger(l *zap.Logger)`: inject your own zap logger (tests)
- `WithLogRetention(n int)`: max logs kept per endpoint (default 100)

## Example: HTTP API Integration

See `main.go` for a small Gin example that exposes endpoints to add sites and to read recent logs.

```go
checker := uptime.New(
    uptime.WithWorkers(5),
    uptime.WithTimeout(10*time.Second),
    uptime.WithLogLevel(uptime.LogError),
)
checker.Start()
defer checker.Stop()

// Add site from request payload, then fetch logs via checker.GetLogs(id, limit)
```

## Folder Structure

```
.
├── main.go                      # Example HTTP API using the package
├── uptime/                      # Reusable library package
│   ├── checker.go               # Public API (New, Start/Stop, AddSite, etc.)
│   ├── options.go               # Functional options
│   ├── types.go                 # Core types and constants
│   ├── workers.go               # Workers, scheduler, internals, logging
│   └── doc.go                   # Package docs
└── uptime_test/                 # Black‑box tests that import the package
    └── checker_test.go
```

## Roadmap

- Retries and backoff strategy per endpoint
- Request headers, query params, and body support (POST/PUT)
- Custom success criteria (e.g., status range, response body match)
- Per‑endpoint jitter and initial delay
- Prometheus metrics and pprof hooks
- Pluggable alerting sinks (email, Slack, PagerDuty)
- Persistent storage backend for logs and endpoints
- Context propagation, graceful drain of workers

## Contributing

- Fork and create a feature branch
- Write tests for new behavior (`go test ./...`)
- Keep changes focused and documented
- Open a PR with a clear description and rationale

To run tests locally:

```bash
go test ./...
```
