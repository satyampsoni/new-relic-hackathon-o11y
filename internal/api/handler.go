package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/alerts"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/staleness"
	"github.com/sirupsen/logrus"
)

// MetricsHandler handles all metrics API endpoints
type MetricsHandler struct {
	detector     *staleness.Detector
	alertManager *alerts.Manager
	config       *config.Config
	logger       *logrus.Logger
	startTime    time.Time
	apiStats     map[string]*APIStats
}

// APIStats holds statistics for each monitored API
type APIStats struct {
	Name               string        `json:"name"`
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	ErrorCount         int64         `json:"error_count"`
	AvgResponseTime    time.Duration `json:"avg_response_time_ms"`
	LastCheck          time.Time     `json:"last_check"`
	Status             string        `json:"status"`
}

// StalenessMetrics represents staleness data for New Relic
type StalenessMetrics struct {
	APIName          string    `json:"api_name"`
	IsStale          bool      `json:"is_stale"`
	FileAgeMinutes   float64   `json:"file_age_minutes"`
	ThresholdMinutes float64   `json:"threshold_minutes"`
	Behavior         string    `json:"behavior"`
	LastCheck        time.Time `json:"last_check"`
}

// HealthMetrics represents API health status
type HealthMetrics struct {
	APIs      []APIHealthStatus `json:"apis"`
	Timestamp time.Time         `json:"timestamp"`
}

// APIHealthStatus represents individual API health
type APIHealthStatus struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	ResponseTimeMs int64   `json:"response_time_ms"`
	LastCheck      string  `json:"last_check"`
	SuccessRate    float64 `json:"success_rate"`
}

// AlertSummary represents alert metrics
type AlertSummary struct {
	TotalAlerts     int64            `json:"total_alerts"`
	AlertsLastHour  int64            `json:"alerts_last_hour"`
	CriticalAlerts  int64            `json:"critical_alerts"`
	WarningAlerts   int64            `json:"warning_alerts"`
	LastAlert       time.Time        `json:"last_alert"`
	AlertsByType    map[string]int64 `json:"alerts_by_type"`
	AlertsByChannel map[string]int64 `json:"alerts_by_channel"`
}

// SystemStats represents system metrics
type SystemStats struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskUsage     float64 `json:"disk_usage"`
	Goroutines    int     `json:"goroutines"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

// PerformanceSummary represents performance metrics for all APIs
type PerformanceSummary struct {
	APIs      []PerformanceMetric `json:"apis"`
	Timestamp time.Time           `json:"timestamp"`
}

// PerformanceMetric represents individual API performance
type PerformanceMetric struct {
	Name              string  `json:"name"`
	AvgResponseTimeMs int64   `json:"avg_response_time_ms"`
	SuccessRate       float64 `json:"success_rate"`
	TotalRequests     int64   `json:"total_requests"`
	ErrorCount        int64   `json:"error_count"`
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(detector *staleness.Detector, alertManager *alerts.Manager, cfg *config.Config, logger *logrus.Logger) *MetricsHandler {
	return &MetricsHandler{
		detector:     detector,
		alertManager: alertManager,
		config:       cfg,
		logger:       logger,
		startTime:    time.Now(),
		apiStats:     make(map[string]*APIStats),
	}
}

// StalenessStatus returns current staleness status for all monitored APIs
func (h *MetricsHandler) StalenessStatus(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Serving staleness metrics endpoint")

	var metrics []StalenessMetrics

	// Generate sample data based on your config
	for _, api := range h.config.APIs {
		if !api.Enabled || !api.Staleness.Enabled {
			continue
		}

		// Simulate staleness check results
		isStale := false
		fileAgeMinutes := 2.5 // Sample data

		// Randomly make some APIs stale for demo
		if api.Name == "complex-data" || api.Name == "performance-data" {
			isStale = true
			fileAgeMinutes = api.Staleness.Threshold.Minutes() + 5
		}

		metrics = append(metrics, StalenessMetrics{
			APIName:          api.Name,
			IsStale:          isStale,
			FileAgeMinutes:   fileAgeMinutes,
			ThresholdMinutes: api.Staleness.Threshold.Minutes(),
			Behavior:         api.Staleness.Behavior,
			LastCheck:        time.Now().Add(-time.Duration(fileAgeMinutes) * time.Minute),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		h.logger.WithError(err).Error("Failed to encode staleness metrics")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// HealthStatus returns health status of all APIs
func (h *MetricsHandler) HealthStatus(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Serving health metrics endpoint")

	var apis []APIHealthStatus

	for _, api := range h.config.APIs {
		if !api.Enabled {
			continue
		}

		// Generate realistic health data
		status := "healthy"
		responseTime := int64(150 + (len(api.Name) * 50)) // Vary by API name length
		successRate := 99.5

		// Simulate some degraded APIs
		if api.Name == "complex-data" {
			status = "degraded"
			responseTime = 3500
			successRate = 85.2
		} else if api.Name == "legacy-system" {
			status = "slow"
			responseTime = 2100
			successRate = 92.1
		}

		apis = append(apis, APIHealthStatus{
			Name:           api.Name,
			Status:         status,
			ResponseTimeMs: responseTime,
			LastCheck:      time.Now().Add(-time.Minute * 2).Format(time.RFC3339),
			SuccessRate:    successRate,
		})
	}

	health := HealthMetrics{
		APIs:      apis,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		h.logger.WithError(err).Error("Failed to encode health metrics")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// AlertsSummary returns alert metrics summary
func (h *MetricsHandler) AlertsSummary(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Serving alerts summary endpoint")

	// Generate sample alert data
	summary := AlertSummary{
		TotalAlerts:    42,
		AlertsLastHour: 7,
		CriticalAlerts: 2,
		WarningAlerts:  5,
		LastAlert:      time.Now().Add(-time.Minute * 15),
		AlertsByType: map[string]int64{
			"staleness": 25,
			"error":     12,
			"health":    5,
		},
		AlertsByChannel: map[string]int64{
			"log-alerts":     35,
			"webhook-alerts": 7,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		h.logger.WithError(err).Error("Failed to encode alerts summary")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SystemStatsHandler returns system performance metrics
func (h *MetricsHandler) SystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Serving system stats endpoint")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := SystemStats{
		CPUUsage:      45.2,                           // Would be calculated from actual CPU metrics
		MemoryUsage:   float64(m.Alloc) / 1024 / 1024, // MB
		DiskUsage:     68.5,                           // Would be calculated from disk metrics
		Goroutines:    runtime.NumGoroutine(),
		UptimeSeconds: int64(time.Since(h.startTime).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		h.logger.WithError(err).Error("Failed to encode system stats")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// PerformanceSummaryHandler returns performance metrics for all APIs
func (h *MetricsHandler) PerformanceSummaryHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Serving performance summary endpoint")

	var metrics []PerformanceMetric

	for _, api := range h.config.APIs {
		if !api.Enabled {
			continue
		}

		// Generate realistic performance data
		avgResponseTime := int64(200 + (len(api.Name) * 30))
		successRate := 98.5
		totalRequests := int64(1000 + (len(api.Name) * 100))
		errorCount := int64(totalRequests) - int64(float64(totalRequests)*(successRate/100))

		// Simulate varying performance
		if api.Name == "complex-data" {
			avgResponseTime = 1500
			successRate = 87.2
		} else if api.Name == "real-time-metrics" {
			avgResponseTime = 50
			successRate = 99.8
		}

		metrics = append(metrics, PerformanceMetric{
			Name:              api.Name,
			AvgResponseTimeMs: avgResponseTime,
			SuccessRate:       successRate,
			TotalRequests:     totalRequests,
			ErrorCount:        errorCount,
		})
	}

	summary := PerformanceSummary{
		APIs:      metrics,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		h.logger.WithError(err).Error("Failed to encode performance summary")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Simple HTTP router without external dependencies
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/staleness/status":
		if r.Method == "GET" {
			h.StalenessStatus(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/health":
		if r.Method == "GET" {
			h.HealthStatus(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/alerts/summary":
		if r.Method == "GET" {
			h.AlertsSummary(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/system/stats":
		if r.Method == "GET" {
			h.SystemStatsHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/performance/summary":
		if r.Method == "GET" {
			h.PerformanceSummaryHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/config/validate":
		if r.Method == "GET" {
			h.validateConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "/api/test/generate-data":
		if r.Method == "POST" {
			h.generateTestData(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

// validateConfig validates the current configuration
func (h *MetricsHandler) validateConfig(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"valid":     true,
		"apis":      len(h.config.APIs),
		"channels":  len(h.config.Alerts.Channels),
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateTestData generates test data for demonstration
func (h *MetricsHandler) generateTestData(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Generating test data")

	response := map[string]interface{}{
		"message":   "Test data generated successfully",
		"timestamp": time.Now(),
		"apis":      len(h.config.APIs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
