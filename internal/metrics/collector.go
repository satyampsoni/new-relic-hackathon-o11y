package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/sirupsen/logrus"
)

// Collector handles New Relic metrics collection and submission
type Collector struct {
	config     config.NewRelicConfig
	client     *http.Client
	logger     *logrus.Logger
	eventBatch []map[string]interface{}
	metricBatch []Metric
	batchMutex  sync.Mutex
	stats       CollectorStats
}

// CollectorStats tracks collector performance
type CollectorStats struct {
	EventsSent       int64     `json:"events_sent"`
	MetricsSent      int64     `json:"metrics_sent"`
	EventsErrorCount int64     `json:"events_error_count"`
	MetricsErrorCount int64    `json:"metrics_error_count"`
	LastEventSent    time.Time `json:"last_event_sent"`
	LastMetricSent   time.Time `json:"last_metric_sent"`
	StartTime        time.Time `json:"start_time"`
}

// Metric represents a New Relic dimensional metric
type Metric struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Value      float64                `json:"value"`
	Timestamp  int64                  `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

// NewCollector creates a new metrics collector
func NewCollector(cfg config.NewRelicConfig, logger *logrus.Logger) *Collector {
	return &Collector{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger,
		eventBatch:  make([]map[string]interface{}, 0),
		metricBatch: make([]Metric, 0),
		stats: CollectorStats{
			StartTime: time.Now(),
		},
	}
}

// AddEvent adds an event to the batch for sending to New Relic
func (c *Collector) AddEvent(eventType string, attributes map[string]interface{}) {
	c.batchMutex.Lock()
	defer c.batchMutex.Unlock()

	event := map[string]interface{}{
		"eventType": eventType,
		"timestamp": time.Now().Unix(),
	}

	// Add all attributes
	for k, v := range attributes {
		event[k] = v
	}

	// Add collector metadata
	event["collector.version"] = "1.0.0"
	event["collector.host"] = getHostname()

	c.eventBatch = append(c.eventBatch, event)

	c.logger.WithFields(logrus.Fields{
		"event_type":   eventType,
		"batch_size":   len(c.eventBatch),
		"attributes":   len(attributes),
	}).Debug("Event added to batch")
}

// AddMetric adds a dimensional metric to the batch
func (c *Collector) AddMetric(name string, metricType string, value float64, attributes map[string]interface{}) {
	c.batchMutex.Lock()
	defer c.batchMutex.Unlock()

	metric := Metric{
		Name:       name,
		Type:       metricType,
		Value:      value,
		Timestamp:  time.Now().UnixMilli(),
		Attributes: make(map[string]interface{}),
	}

	// Add attributes
	for k, v := range attributes {
		metric.Attributes[k] = v
	}

	// Add collector metadata
	metric.Attributes["collector.version"] = "1.0.0"
	metric.Attributes["collector.host"] = getHostname()

	c.metricBatch = append(c.metricBatch, metric)

	c.logger.WithFields(logrus.Fields{
		"metric_name":  name,
		"metric_type":  metricType,
		"value":        value,
		"batch_size":   len(c.metricBatch),
		"attributes":   len(attributes),
	}).Debug("Metric added to batch")
}

// SendEvents sends all batched events to New Relic
func (c *Collector) SendEvents() error {
	c.batchMutex.Lock()
	events := make([]map[string]interface{}, len(c.eventBatch))
	copy(events, c.eventBatch)
	c.eventBatch = c.eventBatch[:0] // Clear batch
	c.batchMutex.Unlock()

	if len(events) == 0 {
		c.logger.Debug("No events to send")
		return nil
	}

	eventsURL := fmt.Sprintf(c.config.EventsURL, c.config.AccountID)
	
	jsonData, err := json.Marshal(events)
	if err != nil {
		c.stats.EventsErrorCount++
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	req, err := http.NewRequest("POST", eventsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.stats.EventsErrorCount++
		return fmt.Errorf("failed to create events request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Insert-Key", c.config.APIKey)
	req.Header.Set("User-Agent", "Enhanced-Flex-Monitor/1.0")

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.stats.EventsErrorCount++
		c.logger.WithError(err).WithFields(logrus.Fields{
			"url":        eventsURL,
			"event_count": len(events),
			"duration":   duration,
		}).Error("Failed to send events to New Relic")
		return fmt.Errorf("failed to send events: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.stats.EventsErrorCount++
		return fmt.Errorf("New Relic Events API returned status %d", resp.StatusCode)
	}

	c.stats.EventsSent += int64(len(events))
	c.stats.LastEventSent = time.Now()

	c.logger.WithFields(logrus.Fields{
		"event_count": len(events),
		"duration":    duration,
		"status":      resp.StatusCode,
	}).Info("Events sent to New Relic successfully")

	return nil
}

// SendMetrics sends all batched metrics to New Relic
func (c *Collector) SendMetrics() error {
	c.batchMutex.Lock()
	metrics := make([]Metric, len(c.metricBatch))
	copy(metrics, c.metricBatch)
	c.metricBatch = c.metricBatch[:0] // Clear batch
	c.batchMutex.Unlock()

	if len(metrics) == 0 {
		c.logger.Debug("No metrics to send")
		return nil
	}

	// New Relic Metrics API format
	payload := []map[string]interface{}{
		{
			"common": map[string]interface{}{
				"timestamp": time.Now().UnixMilli(),
				"interval.ms": 30000,
				"attributes": map[string]interface{}{
					"service.name": "enhanced-flex-monitor",
					"host":         getHostname(),
				},
			},
			"metrics": metrics,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.stats.MetricsErrorCount++
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.MetricsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.stats.MetricsErrorCount++
		return fmt.Errorf("failed to create metrics request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.config.APIKey)
	req.Header.Set("User-Agent", "Enhanced-Flex-Monitor/1.0")

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.stats.MetricsErrorCount++
		c.logger.WithError(err).WithFields(logrus.Fields{
			"url":          c.config.MetricsURL,
			"metric_count": len(metrics),
			"duration":     duration,
		}).Error("Failed to send metrics to New Relic")
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.stats.MetricsErrorCount++
		return fmt.Errorf("New Relic Metrics API returned status %d", resp.StatusCode)
	}

	c.stats.MetricsSent += int64(len(metrics))
	c.stats.LastMetricSent = time.Now()

	c.logger.WithFields(logrus.Fields{
		"metric_count": len(metrics),
		"duration":     duration,
		"status":       resp.StatusCode,
	}).Info("Metrics sent to New Relic successfully")

	return nil
}

// SendBatch sends both events and metrics
func (c *Collector) SendBatch() error {
	var errors []error

	if err := c.SendEvents(); err != nil {
		errors = append(errors, fmt.Errorf("events: %w", err))
	}

	if err := c.SendMetrics(); err != nil {
		errors = append(errors, fmt.Errorf("metrics: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch send failed: %v", errors)
	}

	return nil
}

// RecordProcessingMetrics records file processing performance metrics
func (c *Collector) RecordProcessingMetrics(apiName string, duration time.Duration, recordCount int, isStale bool, hasError bool) {
	attributes := map[string]interface{}{
		"api.name":    apiName,
		"is_stale":    isStale,
		"has_error":   hasError,
	}

	// Duration metric
	c.AddMetric("flex.processing.duration", "gauge", duration.Seconds(), attributes)

	// Record count metric
	c.AddMetric("flex.processing.records", "count", float64(recordCount), attributes)

	// Processing status
	status := 1.0 // success
	if hasError {
		status = 0.0 // failure
	}
	c.AddMetric("flex.processing.status", "gauge", status, attributes)
}

// RecordStalenessMetrics records staleness detection metrics
func (c *Collector) RecordStalenessMetrics(apiName string, fileAge time.Duration, threshold time.Duration, isStale bool) {
	attributes := map[string]interface{}{
		"api.name":  apiName,
		"is_stale":  isStale,
	}

	// File age metric
	c.AddMetric("flex.staleness.file_age", "gauge", fileAge.Seconds(), attributes)

	// Threshold metric
	c.AddMetric("flex.staleness.threshold", "gauge", threshold.Seconds(), attributes)

	// Staleness ratio (file_age / threshold)
	ratio := fileAge.Seconds() / threshold.Seconds()
	c.AddMetric("flex.staleness.ratio", "gauge", ratio, attributes)
}

// GetStats returns collector statistics
func (c *Collector) GetStats() CollectorStats {
	c.batchMutex.Lock()
	defer c.batchMutex.Unlock()

	stats := c.stats
	stats.StartTime = c.stats.StartTime
	return stats
}

// HealthCheck performs a health check by sending a test metric
func (c *Collector) HealthCheck() error {
	testMetric := Metric{
		Name:      "flex.health.check",
		Type:      "gauge",
		Value:     1.0,
		Timestamp: time.Now().UnixMilli(),
		Attributes: map[string]interface{}{
			"service.name": "enhanced-flex-monitor",
			"check.type":   "health",
		},
	}

	payload := []map[string]interface{}{
		{
			"common": map[string]interface{}{
				"timestamp":   time.Now().UnixMilli(),
				"interval.ms": 30000,
			},
			"metrics": []Metric{testMetric},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal health check: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.MetricsURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.config.APIKey)
	req.Header.Set("User-Agent", "Enhanced-Flex-Monitor/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	c.logger.Info("Health check completed successfully")
	return nil
}

// getHostname returns the hostname for metrics attribution
func getHostname() string {
	// In a real implementation, you might want to use os.Hostname()
	// For now, return a default value
	return "enhanced-flex-monitor"
}