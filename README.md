# Uptime Checker (Go)

Concurrent, configurable HTTP uptime checker with a worker pool, simple scheduling, structured logging, and real‑time results streaming.



## Features

* Concurrent worker pool for efficient endpoint checks
* Per-endpoint scheduling with customizable frequency
* Structured logging with a simple, logger‑agnostic API
* In-memory logs with retention cap per endpoint
* Stream results in real time via channel
* Functional options for configuration (timeouts, workers, logging, buffers)
* Load endpoints from JSON for easy bulk setup


## Installation

```bash
go get github.com/amartya2002/uptime-checker-core/uptime
```

In your code:

```go
import "github.com/amartya2002/uptime-checker-core/uptime"
```


## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/amartya2002/uptime-checker-core/uptime"
)

func main() {
    checker := uptime.New(
        uptime.WithWorkers(20),
        uptime.WithTimeout(5*time.Second),
        uptime.WithLogLevel(uptime.LogInfo),
        uptime.LogConsole(true),
        // uptime.LogFile("/var/log/uptime.log"), // optional file sink
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
                res.Timestamp.Format(time.RFC3339),
                res.Endpoint.Name, res.StatusCode, res.Success, res.Latency)
        }
    }()

    select {} // keep running
}
```


## Loading Endpoints from JSON

`endpoints.json`:

```json
[
  {"id":"s1","name":"Google","url":"https://google.com","method":"GET","frequency":30,"expected_status":200},
  {"id":"s2","name":"HTTPBin","url":"https://httpbin.org/status/204","method":"GET","frequency":15,"expected_status":204}
]


Usage:

```go
if err := checker.LoadFromFile("endpoints.json"); err != nil {
    panic(err)
}
```

> Note: `frequency` is expressed in **seconds** in the JSON file.





## Configuration Options

| Option                                                     | Description                                                                                                                                                                                 | Default           | Example                                                                                             |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- | --------------------------------------------------------------------------------------------------- |
| `WithWorkers(int)`                                         | Number of worker goroutines used to check endpoints concurrently                                                                                                                            | `50`              | `WithWorkers(20)`                                                                                   |
| `WithTimeout(time.Duration)`                               | HTTP client timeout per request                                                                                                                                                             | `10s`             | `WithTimeout(5*time.Second)`                                                                        |
| `WithLogLevel(LogNone \| LogError \| LogInfo \| LogDebug)` | Controls site-check log output: <br>• `LogNone`: no logs <br>• `LogError`: only failures <br>• `LogInfo`: successes + failures (default) <br>• `LogDebug`: verbose (status, latency, error) | `LogInfo`         | `WithLogLevel(uptime.LogInfo)`                                                                      |
| `LogConsole(bool)`                                         | Enable/disable console (stdout) output                                                                                                                                                      | `true`            | Console only → `LogConsole(true)` <br> File only → `LogConsole(false)` + one or more `LogFile(...)` |
| `LogFile(string)`                                          | Add a file sink for logs. Repeatable for multiple files.                                                                                                                                    | none              | `LogFile("/var/log/uptime.log")`                                                                    |
| `DisableLogs()`                                            | Disable **all** logging outputs                                                                                                                                                             | enabled by config | `DisableLogs()`                                                                                     |
| `WithResultBuffer(int)`                                    | Results channel buffer size                                                                                                                                                                 | `1000`            | `WithResultBuffer(200)`                                                                             |
| `WithInternalLogs(bool)`                                   | Enable lifecycle logs (scheduler/worker flow)                                                                                                                                               | `false`           | `WithInternalLogs(true)`                                                                            |
| `WithLogRetention(int)`                                    | Per-endpoint in-memory log retention                                                                                                                                                        | `100`             | `WithLogRetention(500)`                                                                             |


Examples:

```go
// Console only (default if not specified)
uptime.New(uptime.LogConsole(true))

// File only
uptime.New(uptime.LogConsole(false), uptime.LogFile("/var/log/uptime.log"))

// Console + file
uptime.New(uptime.LogConsole(true), uptime.LogFile("/var/log/uptime.log"))

// No logs
uptime.New(uptime.DisableLogs())
```



## Example: HTTP API Wrapper

See `examples/gin-server` for a Gin-based API exposing:

* `POST /sites` → register a new site
* `GET /sites/:id/logs` → fetch recent uptime logs

Gin is **not required**; it’s only used for the example.


## Project Structure

```
.
├── uptime/               # Core reusable package
│   ├── checker.go        # Public API (New, Start/Stop, AddSite, etc.)
│   ├── options.go        # Functional options (workers, timeouts, logging)
│   ├── types.go          # Endpoint, Result, Job, LogLevel
│   ├── workers.go        # Worker pool, scheduler, logging internals
│   └── doc.go            # Package docs
├── examples/
│   └── gin-server/       # Example API integration
│       └── main.go
└── go.mod
```

## Development

### Clone & Setup

```bash
git clone https://github.com/amartya2002/uptime-checker-core.git
cd uptime-checker-core
```

### Run Tests

```bash
go test ./...
```

### Code Quality

Format and lint your code before committing:

```bash
go fmt ./...
go vet ./...
```

### Run Example

To run the Gin demo API:

```bash
cd examples/gin-server
go run main.go
```

Then visit: [http://localhost:8080](http://localhost:8080)

## 🤝 Contributing

We welcome contributions!

### Steps

1. **Fork** the repository
2. Create a feature branch:

   ```bash
   git checkout -b feature/your-feature
   ```
3. Write tests for your changes:

   ```bash
   go test ./...
   ```
4. Format & lint:

   ```bash
   go fmt ./...
   go vet ./...
   ```
5. Commit with a descriptive message:

   ```bash
   git commit -m "Add support for custom request headers"
   ```
6. Push to your fork:

   ```bash
   git push origin feature/your-feature
   ```
7. Open a Pull Request with:

   * A clear description of the change
   * Rationale for why it’s needed
   * Example usage if applicable
