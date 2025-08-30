package uptime_test

import (
    "encoding/json"
    "os"
    "testing"
    "time"

    "net/http"
    "net/http/httptest"

    "core-v6/uptime"
    "go.uber.org/zap"
    "go.uber.org/zap/zaptest/observer"
)

// Basic defaults and listing
func TestAddSiteDefaultsAndListSites(t *testing.T) {
    c := uptime.New()
    ep := uptime.Endpoint{ID: "s1", Name: "Test", URL: "http://example"}
    c.AddSite(ep)

    sites := c.ListSites()
    if len(sites) != 1 {
        t.Fatalf("expected 1 site, got %d", len(sites))
    }
    got := sites[0]
    if got.Method != "GET" {
        t.Fatalf("expected default method GET, got %s", got.Method)
    }
    if got.ExpectedStatus != 200 {
        t.Fatalf("expected default expected status 200, got %d", got.ExpectedStatus)
    }
    if got.Frequency == 0 {
        t.Fatalf("expected non-zero default frequency, got 0")
    }
}

// End-to-end success path produces a result
func TestSchedulerAndWorkerFlow_Success(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))
    defer ts.Close()

    c := uptime.New(uptime.WithWorkers(1), uptime.WithResultBuffer(10))
    c.Start()
    defer c.Stop()

    c.AddSite(uptime.Endpoint{
        ID:             "ok",
        Name:           "ok-site",
        URL:            ts.URL,
        Method:         "GET",
        Frequency:      15 * time.Millisecond,
        ExpectedStatus: 200,
    })

    select {
    case res := <-c.Results():
        if !res.Success || res.StatusCode != 200 {
            t.Fatalf("expected success with 200, got success=%v status=%d error=%s", res.Success, res.StatusCode, res.Error)
        }
        if res.Endpoint.ID != "ok" {
            t.Fatalf("expected endpoint id 'ok', got %s", res.Endpoint.ID)
        }
    case <-time.After(2 * time.Second):
        t.Fatalf("timed out waiting for result")
    }
}

// Verify log retention using a small cap and rapid checks
func TestLogsRetentionLimit(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    c := uptime.New(uptime.WithWorkers(1), uptime.WithLogRetention(10))
    c.Start()
    defer c.Stop()

    c.AddSite(uptime.Endpoint{
        ID:             "keep",
        Name:           "fast",
        URL:            ts.URL,
        Method:         "GET",
        Frequency:      2 * time.Millisecond,
        ExpectedStatus: 200,
    })

    // wait for several cycles to occur
    time.Sleep(120 * time.Millisecond)

    logs := c.GetLogs("keep", 1000)
    if len(logs) != 10 {
        t.Fatalf("expected retention of 10 logs, got %d", len(logs))
    }
}

// Load endpoints from JSON and ensure defaults/units
func TestLoadFromFile(t *testing.T) {
    eps := []uptime.Endpoint{{ID: "a", Name: "A", URL: "http://example", Method: "GET", Frequency: 1, ExpectedStatus: 0}}
    b, _ := json.Marshal(eps)
    f, err := os.CreateTemp(t.TempDir(), "eps-*.json")
    if err != nil {
        t.Fatalf("create temp file: %v", err)
    }
    _, _ = f.Write(b)
    _ = f.Close()

    c := uptime.New()
    if err := c.LoadFromFile(f.Name()); err != nil {
        t.Fatalf("LoadFromFile error: %v", err)
    }
    sites := c.ListSites()
    if len(sites) != 1 {
        t.Fatalf("expected 1 site, got %d", len(sites))
    }
    if sites[0].Frequency != 1*time.Second {
        t.Fatalf("expected frequency 1s, got %v", sites[0].Frequency)
    }
    if sites[0].ExpectedStatus != 200 {
        t.Fatalf("expected default expected status 200, got %d", sites[0].ExpectedStatus)
    }
}

// Use zap observer to assert log emission indirectly via checks
func TestLogLevelsWithObserver(t *testing.T) {
    core, obs := observer.New(zap.InfoLevel)
    logger := zap.New(core)

    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    c := uptime.New(uptime.WithLogger(logger), uptime.WithLogLevel(uptime.LogInfo), uptime.WithWorkers(1))
    c.Start()
    defer c.Stop()

    c.AddSite(uptime.Endpoint{ID: "svc", Name: "svc", URL: ts.URL, Method: "GET", Frequency: 10 * time.Millisecond, ExpectedStatus: 200})

    // Wait for at least one log entry to be produced
    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) {
        if len(obs.All()) >= 1 {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("expected at least 1 log entry, got %d", len(obs.All()))
}

