package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/sirupsen/logrus"
)

// Manager handles alert delivery across multiple channels
type Manager struct {
	channels []config.AlertChannel
	client   *http.Client
	logger   *logrus.Logger
}

// NewManager creates a new alert manager
func NewManager(channels []config.AlertChannel, logger *logrus.Logger) *Manager {
	return &Manager{
		channels: channels,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Alert represents an alert to be sent
type Alert struct {
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
	Tags      []string               `json:"tags"`
}

// SendAlert sends an alert through all enabled channels
func (m *Manager) SendAlert(alert Alert) error {
	if len(m.channels) == 0 {
		m.logger.Warn("No alert channels configured, skipping alert")
		return nil
	}

	var errors []error
	for _, channel := range m.channels {
		if !channel.Enabled {
			continue
		}

		if err := m.sendToChannel(alert, channel); err != nil {
			m.logger.WithError(err).WithFields(logrus.Fields{
				"channel": channel.Name,
				"type":    channel.Type,
			}).Error("Failed to send alert to channel")
			errors = append(errors, fmt.Errorf("channel %s: %w", channel.Name, err))
		} else {
			m.logger.WithFields(logrus.Fields{
				"channel": channel.Name,
				"type":    channel.Type,
				"alert":   alert.Type,
			}).Info("Alert sent successfully")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send alert to %d channels: %v", len(errors), errors)
	}

	return nil
}

// sendToChannel sends an alert to a specific channel
func (m *Manager) sendToChannel(alert Alert, channel config.AlertChannel) error {
	switch channel.Type {
	case "webhook":
		return m.sendWebhook(alert, channel)
	case "slack":
		return m.sendSlack(alert, channel)
	case "log":
		return m.sendLog(alert, channel)
	default:
		return fmt.Errorf("unsupported alert channel type: %s", channel.Type)
	}
}

// sendWebhook sends alert via webhook
func (m *Manager) sendWebhook(alert Alert, channel config.AlertChannel) error {
	webhookURL, ok := channel.Settings["url"]
	if !ok {
		return fmt.Errorf("webhook URL not configured for channel %s", channel.Name)
	}

	// Validate webhook URL
	if err := m.validateWebhookURL(webhookURL); err != nil {
		return fmt.Errorf("invalid webhook URL for channel %s: %w", channel.Name, err)
	}

	payload := map[string]interface{}{
		"alert":   alert,
		"channel": channel.Name,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Enhanced-Flex-Monitor/1.0")

	// Add authentication headers if configured
	if apiKey, ok := channel.Settings["api_key"]; ok {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if token, ok := channel.Settings["token"]; ok {
		req.Header.Set("X-Auth-Token", token)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// sendSlack sends alert to Slack
func (m *Manager) sendSlack(alert Alert, channel config.AlertChannel) error {
	webhookURL, ok := channel.Settings["webhook_url"]
	if !ok {
		return fmt.Errorf("Slack webhook URL not configured for channel %s", channel.Name)
	}

	// Create Slack message format
	color := "warning"
	switch alert.Severity {
	case "critical":
		color = "danger"
	case "warning":
		color = "warning"
	case "info":
		color = "good"
	}

	slackPayload := map[string]interface{}{
		"username":   "Enhanced Flex Monitor",
		"icon_emoji": ":warning:",
		"attachments": []map[string]interface{}{
			{
				"color":       color,
				"title":       alert.Title,
				"text":        alert.Message,
				"timestamp":   alert.Timestamp.Unix(),
				"footer":      "Enhanced Flex Monitor",
				"footer_icon": ":chart_with_upwards_trend:",
				"fields": []map[string]interface{}{
					{
						"title": "Source",
						"value": alert.Source,
						"short": true,
					},
					{
						"title": "Type",
						"value": alert.Type,
						"short": true,
					},
					{
						"title": "Severity",
						"value": alert.Severity,
						"short": true,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(slackPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// sendLog sends alert to log
func (m *Manager) sendLog(alert Alert, channel config.AlertChannel) error {
	level := channel.Settings["level"]
	if level == "" {
		level = "warn"
	}

	logEntry := m.logger.WithFields(logrus.Fields{
		"alert_type": alert.Type,
		"severity":   alert.Severity,
		"source":     alert.Source,
		"timestamp":  alert.Timestamp,
		"metadata":   alert.Metadata,
		"tags":       alert.Tags,
		"channel":    channel.Name,
	})

	switch level {
	case "debug":
		logEntry.Debug(alert.Message)
	case "info":
		logEntry.Info(alert.Message)
	case "warn":
		logEntry.Warn(alert.Message)
	case "error":
		logEntry.Error(alert.Message)
	case "fatal":
		logEntry.Fatal(alert.Message)
	default:
		logEntry.Warn(alert.Message)
	}

	return nil
}

// SendStalenessAlert creates and sends a staleness-specific alert
func (m *Manager) SendStalenessAlert(apiName, url string, fileAge, threshold time.Duration) error {
	alert := Alert{
		Type:      "file_staleness",
		Severity:  "warning",
		Title:     fmt.Sprintf("File Staleness Detected: %s", apiName),
		Message:   fmt.Sprintf("File at %s is stale. Age: %v, Threshold: %v", url, fileAge, threshold),
		Source:    apiName,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"url":       url,
			"file_age":  fileAge.Seconds(),
			"threshold": threshold.Seconds(),
			"api_name":  apiName,
		},
		Tags: []string{"staleness", "file_monitor", apiName},
	}

	return m.SendAlert(alert)
}

// SendErrorAlert creates and sends an error alert
func (m *Manager) SendErrorAlert(apiName, operation string, err error) error {
	alert := Alert{
		Type:      "error",
		Severity:  "error",
		Title:     fmt.Sprintf("Error in %s: %s", apiName, operation),
		Message:   fmt.Sprintf("Operation '%s' failed for API '%s': %v", operation, apiName, err),
		Source:    apiName,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"api_name":  apiName,
			"operation": operation,
			"error":     err.Error(),
		},
		Tags: []string{"error", "file_monitor", apiName},
	}

	return m.SendAlert(alert)
}

// SendHealthAlert creates and sends a health check alert
func (m *Manager) SendHealthAlert(component, status string, metadata map[string]interface{}) error {
	severity := "info"
	if status != "healthy" {
		severity = "warning"
	}

	alert := Alert{
		Type:      "health_check",
		Severity:  severity,
		Title:     fmt.Sprintf("Health Check: %s is %s", component, status),
		Message:   fmt.Sprintf("Component %s reported status: %s", component, status),
		Source:    component,
		Timestamp: time.Now(),
		Metadata:  metadata,
		Tags:      []string{"health_check", component},
	}

	return m.SendAlert(alert)
}

// TestChannels tests all configured alert channels
func (m *Manager) TestChannels() error {
	testAlert := Alert{
		Type:      "test",
		Severity:  "info",
		Title:     "Test Alert",
		Message:   "This is a test alert to verify channel configuration",
		Source:    "enhanced-flex-monitor",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test": true,
		},
		Tags: []string{"test"},
	}

	return m.SendAlert(testAlert)
}

// validateWebhookURL validates webhook URL format and scheme
func (m *Manager) validateWebhookURL(webhookURL string) error {
	if strings.TrimSpace(webhookURL) == "" {
		return fmt.Errorf("webhook URL cannot be empty")
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("webhook URL must include a scheme (http or https)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported webhook URL scheme '%s', only http and https are supported", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("webhook URL must include a host")
	}

	return nil
}
