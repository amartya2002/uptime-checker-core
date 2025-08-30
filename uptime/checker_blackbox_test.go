package uptime_test

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    "time"

    "net/http"
    "net/http/httptest"

    up "github.com/amartya2002/uptime-checker-core/uptime"
)

// Test that adding a site applies defaults and ListSites works as an external user would expect.
func TestAddSiteDefaultsAndListSites(t *testing.T) {
    c := up.New()
    ep := up.Endpoint{ID: "s1", Name: "Test", URL: "http://example"}
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

// End-to-end success path using a real HTTP server and the public API.
func TestSchedulerAndWorkerFlow_Success(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    }))
    defer ts.Close()

    c := up.New(up.WithWorkers(1), up.WithResultBuffer(10))
    c.Start()
    defer c.Stop()

    c.AddSite(up.Endpoint{
        ID:             "ok",
        Name:           "ok-site",
        URL:            ts.URL,
        Method:         "GET",
        Frequency:      20 * time.Millisecond,
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

// Verify per-endpoint log retention cap via public GetLogs API.
func TestLogsRetentionLimit(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    c := up.New(up.WithWorkers(1), up.WithLogRetention(10))
    c.Start()
    defer c.Stop()

    c.AddSite(up.Endpoint{
        ID:             "keep",
        Name:           "fast",
        URL:            ts.URL,
        Method:         "GET",
        Frequency:      3 * time.Millisecond,
        ExpectedStatus: 200,
    })

    time.Sleep(150 * time.Millisecond)
    logs := c.GetLogs("keep", 1000)
    if len(logs) != 10 {
        t.Fatalf("expected retention of 10 logs, got %d", len(logs))
    }
}

// Ensure LoadFromFile reads JSON where frequency is in seconds and applies defaults.
func TestLoadFromFile_SecondsUnit(t *testing.T) {
    // Use an inline temp file to avoid dependency on working directory
    eps := []up.Endpoint{{ID: "a", Name: "A", URL: "http://example", Method: "GET", Frequency: 1, ExpectedStatus: 0}}
    b, _ := json.Marshal(eps)
    f, err := os.CreateTemp(t.TempDir(), "eps-*.json")
    if err != nil {
        t.Fatalf("create temp file: %v", err)
    }
    _, _ = f.Write(b)
    _ = f.Close()

    c := up.New()
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

// Validate logging options: file-only, console off, no panic; file created and non-empty after a check.
func TestLogging_FileOnlyProducesOutput(t *testing.T) {
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer ts.Close()

    dir := t.TempDir()
    path := filepath.Join(dir, "uptime.log")

    c := up.New(
        up.WithWorkers(1),
        up.WithLogLevel(up.LogInfo),
        up.LogConsole(false),
        up.LogFile(path),
    )
    c.Start()
    defer c.Stop()

    c.AddSite(up.Endpoint{ID: "one", Name: "one", URL: ts.URL, Method: "GET", Frequency: 15 * time.Millisecond, ExpectedStatus: 200})

    // wait for at least one cycle and a small write window
    time.Sleep(120 * time.Millisecond)

    info, err := os.Stat(path)
    if err != nil {
        t.Fatalf("expected log file to exist: %v", err)
    }
    if info.Size() == 0 {
        t.Fatalf("expected log file to be non-empty")
    }
}

