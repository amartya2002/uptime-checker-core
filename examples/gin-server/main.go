package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/amartya2002/uptime-checker-core/uptime"
)

type Site struct {
	URL            string `json:"url" binding:"required"`
	Name           string `json:"name" binding:"required"`
	ExpectedStatus int    `json:"expected_status,omitempty"`
	CheckInterval  int    `json:"check_interval,omitempty"`
}
type LogResponse struct {
	Timestamp  time.Time `json:"timestamp"`
	StatusCode int       `json:"status_code"`
	LatencyMS  int64     `json:"latency_ms"`
	Status     string    `json:"status"`
}

type SiteLogResponse struct {
	Site uptime.Endpoint `json:"site"`
	Logs []LogResponse   `json:"logs"`
}

func main() {
	// Initialize uptime checker
    checker := uptime.New(
        uptime.WithWorkers(5),
        uptime.WithTimeout(10*time.Second),
        uptime.WithLogLevel(uptime.LogInfo), // site check logs
        uptime.WithInternalLogs(true),
        // Logging outputs: use explicit options for clarity
        uptime.LogConsole(true),            // console on/off
        // uptime.LogFile("/tmp/uptime.log"), // add file sink (repeatable)
        // uptime.DisableLogs(),              // disable logging entirely
    )
	checker.Start()
	defer checker.Stop()

	r := gin.Default()

	// Add single site
	r.POST("/sites", func(c *gin.Context) {
		var site Site
		if err := c.ShouldBindJSON(&site); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ep := uptime.Endpoint{
			ID:             uuid.NewString(), // generate ID here
			Name:           site.Name,
			URL:            site.URL,
			Method:         "GET",
			Frequency:      time.Duration(site.CheckInterval) * time.Second,
			ExpectedStatus: site.ExpectedStatus,
		}

		checker.AddSite(ep)

		// TODO: save ep to DB here

		c.JSON(http.StatusCreated, gin.H{
			"message": "Site added successfully",
			"site":    ep,
		})
	})

	// Add multiple sites
	r.POST("/sites/batch", func(c *gin.Context) {
		var sites []Site
		if err := c.ShouldBindJSON(&sites); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var eps []uptime.Endpoint
		for _, site := range sites {
			ep := uptime.Endpoint{
				ID:             uuid.NewString(),
				Name:           site.Name,
				URL:            site.URL,
				Method:         "GET",
				Frequency:      time.Duration(site.CheckInterval) * time.Second,
				ExpectedStatus: site.ExpectedStatus,
			}
			if ep.Frequency == 0 {
				ep.Frequency = 30 * time.Second
			}
			if ep.ExpectedStatus == 0 {
				ep.ExpectedStatus = 200
			}
			eps = append(eps, ep)
		}

		checker.AddSitesBulk(eps)

		// TODO: persist eps to DB here

		c.JSON(http.StatusCreated, gin.H{
			"message": "Sites added successfully",
			"count":   len(eps),
			"sites":   eps,
		})
	})

	// Get logs of a site by ID
	r.GET("/sites/:id/logs", func(c *gin.Context) {
		id := c.Param("id")
		rawLogs := checker.GetLogs(id, 50)

		if len(rawLogs) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No logs found for site"})
			return
		}

		var logs []LogResponse
		for _, l := range rawLogs {
			logs = append(logs, LogResponse{
				Timestamp:  l.Timestamp,
				StatusCode: l.StatusCode,
				LatencyMS:  l.Latency.Milliseconds(),
				Status:     map[bool]string{true: "UP", false: "DOWN"}[l.Success],
			})
		}

		// Attach site metadata from the latest log
		response := SiteLogResponse{
			Site: rawLogs[len(rawLogs)-1].Endpoint,
			Logs: logs,
		}

		c.JSON(http.StatusOK, response)
	})

	// Health check for API
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("🚀 Server running at http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
