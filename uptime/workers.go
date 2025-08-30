package uptime

import (
    "fmt"
    "net/http"
    "time"

    "go.uber.org/zap"
)

// ===== Workers, Scheduler, and Internals =====
func (c *Checker) worker(id int) {
    defer c.wg.Done()
    for {
        select {
        case <-c.stopCh:
            return
        case job, ok := <-c.jobs:
            if !ok {
                return
            }
            c.ilog("Worker %d picked job for site %s (scheduled at %s)", id, job.Endpoint.Name, job.RunAt.Format(time.RFC3339))
            result := c.checkEndpoint(job.Endpoint)
            c.results <- result
            c.saveLog(result)
            c.log(result)
            c.ilog("Worker %d finished job for site %s (success=%v, latency=%v)", id, result.Endpoint.Name, result.Success, result.Latency)
        }
    }
}

func (c *Checker) scheduler() {
    defer c.wg.Done()
    c.mu.Lock()
    for _, ep := range c.endpoints {
        c.scheduleEndpoint(ep)
    }
    c.mu.Unlock()
    <-c.stopCh
}

func (c *Checker) scheduleEndpoint(ep Endpoint) {
    c.ilog("Scheduling site %s (%s) every %v", ep.Name, ep.URL, ep.Frequency)
    ticker := time.NewTicker(ep.Frequency)
    go func(e Endpoint, t *time.Ticker) {
        for {
            select {
            case <-c.stopCh:
                t.Stop()
                return
            case <-t.C:
                c.ilog("Job scheduled for site %s at %s", e.Name, time.Now().Format(time.RFC3339))
                select {
                case c.jobs <- Job{Endpoint: e, RunAt: time.Now()}:
                case <-c.stopCh:
                    t.Stop()
                    return
                }
            }
        }
    }(ep, ticker)
}

func (c *Checker) checkEndpoint(ep Endpoint) Result {
    start := time.Now()
    currentTime := time.Now()

    req, err := http.NewRequest(ep.Method, ep.URL, nil)
    if err != nil {
        return Result{
            Endpoint:  ep,
            Timestamp: currentTime,
            Latency:   time.Since(start),
            Success:   false,
            Error:     fmt.Sprintf("Error creating request: %v", err),
        }
    }
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return Result{
            Endpoint:  ep,
            Timestamp: currentTime,
            Latency:   time.Since(start),
            Success:   false,
            Error:     err.Error(),
        }
    }
    defer resp.Body.Close()

    success := resp.StatusCode == ep.ExpectedStatus
    return Result{
        Endpoint:   ep,
        Timestamp:  currentTime,
        StatusCode: resp.StatusCode,
        Latency:    time.Since(start),
        Success:    success,
    }
}

func (c *Checker) saveLog(res Result) {
    c.mu.Lock()
    defer c.mu.Unlock()
    id := res.Endpoint.ID
    c.logs[id] = append(c.logs[id], res)
    if len(c.logs[id]) > c.logRetention {
        c.logs[id] = c.logs[id][len(c.logs[id])-c.logRetention:]
    }
}

func (c *Checker) isRunning() bool {
    select {
    case <-c.stopCh:
        return false
    default:
        return true
    }
}

func (c *Checker) log(res Result) {
    switch c.logLevel {
    case LogNone:
        return
    case LogError:
        if !res.Success {
            c.logger.Error("Site DOWN", zap.String("name", res.Endpoint.Name), zap.String("error", res.Error))
        }
    case LogInfo:
        if res.Success {
            c.logger.Info("Site UP", zap.String("name", res.Endpoint.Name), zap.Int("status_code", res.StatusCode))
        } else {
            c.logger.Warn("Site DOWN", zap.String("name", res.Endpoint.Name), zap.String("error", res.Error))
        }
    case LogDebug:
        c.logger.Debug("Site check", zap.String("name", res.Endpoint.Name),
            zap.Int("status_code", res.StatusCode), zap.Duration("latency", res.Latency), zap.String("error", res.Error))
    }
}

// ===== Internal Logging Helper =====
func (c *Checker) ilog(format string, args ...interface{}) {
    if c.enableInternalLogs {
        c.logger.Info(fmt.Sprintf("[INTERNAL] "+format, args...))
    }
}
