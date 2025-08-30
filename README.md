# Uptime Checker

[![Go Reference](https://pkg.go.dev/badge/github.com/amartya2002/uptime-checker-core/uptime.svg)](https://pkg.go.dev/github.com/amartya2002/uptime-checker-core/uptime)
[![Go Report Card](https://goreportcard.com/badge/github.com/amartya2002/uptime-checker-core)](https://goreportcard.com/report/github.com/amartya2002/uptime-checker-core)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

A concurrent, configurable HTTP uptime checker for Go.
Provides a worker pool, scheduling, structured logging, and real-time results streaming.

---

## Features

* Concurrent worker pool for efficient endpoint checks
* Per-endpoint scheduling with customizable frequency
* Structured logging via [zap](https://github.com/uber-go/zap)
* In-memory logs with retention cap per endpoint
* Stream results in real time via channel
* Functional options for configuration (timeouts, workers, logging, buffers)
* Load endpoints from JSON for easy bulk setup

---

## Installation

```bash
go get github.com/amartya2002/uptime-checker-core/uptime
```

In your code:

```go
import "github.com/amartya2002/uptime-checker-core/uptime"
```

---

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

---

## Loading Endpoints from JSON

`endpoints.json`:

```json
[
  {"id":"s1","name":"Google","url":"https://google.com","method":"GET","frequency":30,"expected_status":200},
  {"id":"s2","name":"HTTPBin","url":"https://httpbin.org/status/204","method":"GET","frequency":15,"expected_status":204}
]
```

Usage:

```go
if err := checker.LoadFromFile("endpoints.json"); err != nil {
    panic(err)
}
```

> Note: `frequency` is expressed in **seconds** in the JSON file.

---

## ⚙Configuration Options

| Option                           | Description                                                                                                                                                       | Example                                | Default             |
| -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- | ------------------- |
| `WithWorkers(n int)`             | Number of worker goroutines for concurrent checks                                                                                                                 | `WithWorkers(50)`                      | `50`                |
| `WithTimeout(d time.Duration)`   | HTTP client timeout per request                                                                                                                                   | `WithTimeout(5*time.Second)`           | `10s`               |
| `WithLogLevel(level LogLevel)`   | Log verbosity: <br>• `LogNone` → no logs<br>• `LogError` → only failures<br>• `LogInfo` → successes + failures<br>• `LogDebug` → verbose (status, latency, error) | `WithLogLevel(uptime.LogInfo)`         | `LogInfo`           |
| `WithResultBuffer(size int)`     | Results channel buffer size                                                                                                                                       | `WithResultBuffer(500)`                | `1000`              |
| `WithInternalLogs(enabled bool)` | Enable lifecycle logs (scheduler, workers)                                                                                                                        | `WithInternalLogs(true)`               | `false`             |
| `WithZapLogger(filePath string)` | Output logs:<br>• empty → console only<br>• path → console + file                                                                                                 | `WithZapLogger("/var/log/uptime.log")` | Console only        |
| `WithLogger(l *zap.Logger)`      | Inject your own zap logger (tests, advanced configs)                                                                                                              | `WithLogger(zap.NewExample())`         | Internal zap logger |
| `WithLogRetention(n int)`        | Maximum in-memory logs kept per endpoint                                                                                                                          | `WithLogRetention(200)`                | `100`               |

---

## Example: HTTP API Wrapper

See [`examples/gin-server`](./examples/gin-server) for a Gin-based API exposing:

* `POST /sites` → register a new site
* `GET /sites/:id/logs` → fetch recent uptime logs

Gin is **not required**; it’s only used for the example.

---

## Project Structure

```
.
├── uptime/               # Core reusable package
│   ├── checker.go        # Checker struct + constructor
│   ├── endpoint.go       # Endpoint, Result, Job definitions
│   ├── options.go        # Functional options
│   ├── scheduler.go      # Worker pool + scheduling logic
│   ├── logging.go        # Logging + log levels
│   └── storage.go        # In-memory logs
├── examples/
│   └── gin-server/       # Example API integration
│       └── main.go
└── go.mod
```

---

## 🛠 Development

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

---

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
