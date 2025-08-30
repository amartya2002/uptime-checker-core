// Package uptime implements the high-level Checker public API.
package uptime

import (
    "encoding/json"
    "net/http"
    "os"
    "sync"
    "time"

    "go.uber.org/zap"
)

type Checker struct {
    httpClient *http.Client
    numWorkers int
    logLevel   LogLevel
    logRetention int

    enableInternalLogs bool
    logger             *zap.Logger
    loggerExplicit     bool // set when WithLogger/WithZapLogger used

    // logging configuration accumulated by options
    logConsoleOpt *bool
    logFilesOpt   []string
    logDisableOpt bool

    jobs    chan Job
    results chan Result
    wg      sync.WaitGroup

    mu        sync.Mutex
    endpoints []Endpoint
    logs      map[string][]Result
    stopCh    chan struct{}
}

// ===== Constructor =====
func New(opts ...Option) *Checker {
    c := &Checker{
        httpClient: &http.Client{Timeout: 10 * time.Second},
        numWorkers: 50,
        logLevel:   LogInfo,
        logRetention: 100,
        jobs:       make(chan Job, 1000),
        results:    make(chan Result, 1000),
        stopCh:     make(chan struct{}),
        logs:       make(map[string][]Result),
        logger:     nil, // build after applying options
    }
    for _, opt := range opts {
        opt(c)
    }
    // Build logger after options applied unless explicitly provided
    if !c.loggerExplicit {
        c.logger = c.buildLoggerFromConfig()
    }
    // Safety fallback
    if c.logger == nil {
        c.logger = defaultConsoleLogger()
    }
    return c
}

func defaultConsoleLogger() *zap.Logger {
    l, err := zap.NewProduction(zap.AddCallerSkip(1))
    if err != nil {
        return zap.NewNop()
    }
    return l
}

func (c *Checker) buildLoggerFromConfig() *zap.Logger {
    // If disabled explicitly
    if c.logDisableOpt {
        return zap.NewNop()
    }

    // Determine console default: true unless explicitly set to false
    console := true
    if c.logConsoleOpt != nil {
        console = *c.logConsoleOpt
    }

    // Build output paths
    var paths []string
    seen := map[string]struct{}{}
    if console {
        paths = append(paths, "stdout")
        seen["stdout"] = struct{}{}
    }
    for _, f := range c.logFilesOpt {
        if f == "" {
            continue
        }
        if _, ok := seen[f]; ok {
            continue
        }
        seen[f] = struct{}{}
        paths = append(paths, f)
    }

    if len(paths) == 0 {
        // No outputs selected: default to console
        return defaultConsoleLogger()
    }

    cfg := zap.NewProductionConfig()
    cfg.OutputPaths = paths
    l, err := cfg.Build()
    if err != nil {
        return zap.NewNop()
    }
    return l
}

// ===== Public API =====
func (c *Checker) Start() {
    for i := 0; i < c.numWorkers; i++ {
        c.wg.Add(1)
        go c.worker(i)
        c.ilog("Started worker %d", i)
    }
    c.wg.Add(1)
    go c.scheduler()
    c.ilog("Scheduler started")
}

func (c *Checker) Stop() {
    // Signal all goroutines to stop, then close jobs to unblock workers
    close(c.stopCh)
    close(c.jobs)
    c.wg.Wait()
    close(c.results)
    c.ilog("Checker stopped")
}

// AddSite (requires caller to supply ID)
func (c *Checker) AddSite(ep Endpoint) {
    if ep.Frequency == 0 {
        ep.Frequency = 30 * time.Second
    }
    if ep.ExpectedStatus == 0 {
        ep.ExpectedStatus = 200
    }
    if ep.Method == "" {
        ep.Method = "GET"
    }
    c.mu.Lock()
    c.endpoints = append(c.endpoints, ep)
    c.mu.Unlock()

    c.ilog("Registered site: %s (%s)", ep.Name, ep.URL)

    if c.isRunning() {
        c.scheduleEndpoint(ep)
    }
}

// AddSitesBulk
func (c *Checker) AddSitesBulk(sites []Endpoint) {
    c.mu.Lock()
    c.endpoints = append(c.endpoints, sites...)
    c.mu.Unlock()

    c.ilog("Registered %d sites", len(sites))

    if c.isRunning() {
        for _, ep := range sites {
            if ep.Method == "" {
                ep.Method = "GET"
            }
            if ep.Frequency == 0 {
                ep.Frequency = 30 * time.Second
            }
            if ep.ExpectedStatus == 0 {
                ep.ExpectedStatus = 200
            }
            c.scheduleEndpoint(ep)
        }
    }
}

// LoadFromFile
func (c *Checker) LoadFromFile(filePath string) error {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }
    var eps []Endpoint
    if err := json.Unmarshal(data, &eps); err != nil {
        return err
    }
    for i := range eps {
        eps[i].Frequency *= time.Second
        if eps[i].ExpectedStatus == 0 {
            eps[i].ExpectedStatus = 200
        }
    }
    c.ilog("Loaded %d sites from file: %s", len(eps), filePath)
    c.AddSitesBulk(eps)
    return nil
}

// Results channel
func (c *Checker) Results() <-chan Result { return c.results }

// GetLogs returns last N results
func (c *Checker) GetLogs(id string, limit int) []Result {
    c.mu.Lock()
    defer c.mu.Unlock()
    logs := c.logs[id]
    if len(logs) > limit {
        return logs[len(logs)-limit:]
    }
    return logs
}

// ListSites returns all registered sites
func (c *Checker) ListSites() []Endpoint {
    c.mu.Lock()
    defer c.mu.Unlock()
    return append([]Endpoint(nil), c.endpoints...)
}
