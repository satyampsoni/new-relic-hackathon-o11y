package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Global   GlobalConfig   `yaml:"global"`
	NewRelic NewRelicConfig `yaml:"newrelic"`
	Alerts   AlertsConfig   `yaml:"alerts"`
	APIs     []APIConfig    `yaml:"apis"`
}

// GlobalConfig contains global application settings
type GlobalConfig struct {
	Name          string        `yaml:"name"`
	Interval      time.Duration `yaml:"interval"`
	LogLevel      string        `yaml:"log_level"`
	EnableMetrics bool          `yaml:"enable_metrics"`
	EnableAlerts  bool          `yaml:"enable_alerts"`
	WorkerCount   int           `yaml:"worker_count"`
}

// NewRelicConfig contains New Relic integration settings
type NewRelicConfig struct {
	APIKey     string `yaml:"api_key"`
	Region     string `yaml:"region"`
	AccountID  string `yaml:"account_id"`
	EventsURL  string `yaml:"events_url"`
	MetricsURL string `yaml:"metrics_url"`
}

// AlertsConfig contains alert configuration
type AlertsConfig struct {
	Channels []AlertChannel `yaml:"channels"`
}

// AlertChannel represents different alert delivery methods
type AlertChannel struct {
	Type     string            `yaml:"type"`
	Name     string            `yaml:"name"`
	Enabled  bool              `yaml:"enabled"`
	Settings map[string]string `yaml:"settings"`
}

// APIConfig represents a file monitoring configuration
type APIConfig struct {
	Name        string            `yaml:"name"`
	URL         string            `yaml:"url"`
	FallbackURL string            `yaml:"fallback_url"`
	Format      string            `yaml:"format"`
	JQ          string            `yaml:"jq"`
	Attributes  map[string]string `yaml:"attributes"`
	EventType   string            `yaml:"event_type"`
	Staleness   StalenessConfig   `yaml:"staleness"`
	Enabled     bool              `yaml:"enabled"`
}

// StalenessConfig contains file staleness detection settings
type StalenessConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Threshold time.Duration `yaml:"threshold"`
	Behavior  string        `yaml:"behavior"` // skip, alert, continue
	CheckURL  string        `yaml:"check_url"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	content := os.ExpandEnv(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults and validate
	if err := config.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults applies default values to configuration
func (c *Config) setDefaults() error {
	// Global defaults
	if c.Global.Name == "" {
		c.Global.Name = "enhanced-flex-monitor"
	}
	if c.Global.Interval == 0 {
		c.Global.Interval = 30 * time.Second
	}
	if c.Global.LogLevel == "" {
		c.Global.LogLevel = "info"
	}
	if c.Global.WorkerCount == 0 {
		c.Global.WorkerCount = 4
	}

	// New Relic defaults
	if c.NewRelic.Region == "" {
		c.NewRelic.Region = "US"
	}
	if c.NewRelic.EventsURL == "" {
		if c.NewRelic.Region == "EU" {
			c.NewRelic.EventsURL = "https://insights-collector.eu01.nr-data.net/v1/accounts/%s/events"
		} else {
			c.NewRelic.EventsURL = "https://insights-collector.newrelic.com/v1/accounts/%s/events"
		}
	}
	if c.NewRelic.MetricsURL == "" {
		if c.NewRelic.Region == "EU" {
			c.NewRelic.MetricsURL = "https://metric-api.eu.newrelic.com/metric/v1"
		} else {
			c.NewRelic.MetricsURL = "https://metric-api.newrelic.com/metric/v1"
		}
	}

	// API defaults
	for i := range c.APIs {
		api := &c.APIs[i]
		if api.Format == "" {
			api.Format = "json"
		}
		if api.EventType == "" {
			api.EventType = "FlexSample"
		}
		if !api.Staleness.Enabled {
			continue
		}
		if api.Staleness.Threshold == 0 {
			api.Staleness.Threshold = 5 * time.Minute
		}
		if api.Staleness.Behavior == "" {
			api.Staleness.Behavior = "continue"
		}
		if api.Staleness.CheckURL == "" && api.URL != "" {
			api.Staleness.CheckURL = api.URL
		}
	}

	return nil
}

// validate checks configuration for required fields and consistency
func (c *Config) validate() error {
	// Validate New Relic config
	if c.NewRelic.APIKey == "" {
		return fmt.Errorf("newrelic.api_key is required")
	}
	if c.NewRelic.AccountID == "" {
		return fmt.Errorf("newrelic.account_id is required")
	}

	// Validate global settings
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if !contains(validLogLevels, strings.ToLower(c.Global.LogLevel)) {
		return fmt.Errorf("invalid log_level: %s, must be one of %v", c.Global.LogLevel, validLogLevels)
	}

	if c.Global.WorkerCount < 1 || c.Global.WorkerCount > 100 {
		return fmt.Errorf("worker_count must be between 1 and 100, got %d", c.Global.WorkerCount)
	}

	// Validate APIs
	if len(c.APIs) == 0 {
		return fmt.Errorf("at least one API configuration is required")
	}

	for i, api := range c.APIs {
		if api.Name == "" {
			return fmt.Errorf("api[%d].name is required", i)
		}
		if api.URL == "" {
			return fmt.Errorf("api[%d].url is required", i)
		}

		validFormats := []string{"json", "csv"}
		if !contains(validFormats, strings.ToLower(api.Format)) {
			return fmt.Errorf("api[%d].format must be one of %v, got %s", i, validFormats, api.Format)
		}

		if api.Staleness.Enabled {
			validBehaviors := []string{"skip", "alert", "continue"}
			if !contains(validBehaviors, strings.ToLower(api.Staleness.Behavior)) {
				return fmt.Errorf("api[%d].staleness.behavior must be one of %v, got %s", i, validBehaviors, api.Staleness.Behavior)
			}
			if api.Staleness.Threshold <= 0 {
				return fmt.Errorf("api[%d].staleness.threshold must be positive", i)
			}
		}
	}

	// Validate alert channels
	for i, channel := range c.Alerts.Channels {
		if channel.Name == "" {
			return fmt.Errorf("alerts.channels[%d].name is required", i)
		}
		validTypes := []string{"webhook", "slack", "log"}
		if !contains(validTypes, strings.ToLower(channel.Type)) {
			return fmt.Errorf("alerts.channels[%d].type must be one of %v, got %s", i, validTypes, channel.Type)
		}
	}

	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetEnabledAPIs returns only enabled API configurations
func (c *Config) GetEnabledAPIs() []APIConfig {
	var enabled []APIConfig
	for _, api := range c.APIs {
		if api.Enabled {
			enabled = append(enabled, api)
		}
	}
	return enabled
}

// GetEnabledAlertChannels returns only enabled alert channels
func (c *Config) GetEnabledAlertChannels() []AlertChannel {
	var enabled []AlertChannel
	for _, channel := range c.Alerts.Channels {
		if channel.Enabled {
			enabled = append(enabled, channel)
		}
	}
	return enabled
}
