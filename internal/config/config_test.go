package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
global:
  name: "test-monitor"
  interval: 60s
  log_level: "debug"
  enable_metrics: true
  enable_alerts: false
  worker_count: 2

newrelic:
  api_key: "test-key"
  account_id: "123456"
  region: "US"

alerts:
  channels:
    - name: "test-log"
      type: "log"
      enabled: true
      settings:
        level: "info"

apis:
  - name: "test-api"
    url: "https://example.com/test.json"
    format: "json"
    event_type: "TestEvent"
    enabled: true
    staleness:
      enabled: true
      threshold: 5m
      behavior: "skip"
`

	tmpFile, err := os.CreateTemp("", "config-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	// Test loading configuration
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Validate global settings
	if config.Global.Name != "test-monitor" {
		t.Errorf("Expected name 'test-monitor', got '%s'", config.Global.Name)
	}

	if config.Global.Interval != 60*time.Second {
		t.Errorf("Expected interval 60s, got %v", config.Global.Interval)
	}

	if config.Global.WorkerCount != 2 {
		t.Errorf("Expected worker count 2, got %d", config.Global.WorkerCount)
	}

	// Validate New Relic settings
	if config.NewRelic.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", config.NewRelic.APIKey)
	}

	if config.NewRelic.AccountID != "123456" {
		t.Errorf("Expected account ID '123456', got '%s'", config.NewRelic.AccountID)
	}

	// Validate APIs
	if len(config.APIs) != 1 {
		t.Fatalf("Expected 1 API, got %d", len(config.APIs))
	}

	api := config.APIs[0]
	if api.Name != "test-api" {
		t.Errorf("Expected API name 'test-api', got '%s'", api.Name)
	}

	if !api.Staleness.Enabled {
		t.Error("Expected staleness to be enabled")
	}

	if api.Staleness.Threshold != 5*time.Minute {
		t.Errorf("Expected staleness threshold 5m, got %v", api.Staleness.Threshold)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				Global: GlobalConfig{
					Name:         "test",
					Interval:     30 * time.Second,
					LogLevel:     "info",
					WorkerCount:  4,
				},
				NewRelic: NewRelicConfig{
					APIKey:    "test-key",
					AccountID: "123456",
					Region:    "US",
				},
				APIs: []APIConfig{
					{
						Name:      "test-api",
						URL:       "https://example.com/test.json",
						Format:    "json",
						EventType: "TestEvent",
						Enabled:   true,
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing API key",
			config: Config{
				Global: GlobalConfig{
					LogLevel:    "info",
					WorkerCount: 4,
				},
				NewRelic: NewRelicConfig{
					AccountID: "123456",
				},
				APIs: []APIConfig{
					{
						Name:    "test-api",
						URL:     "https://example.com/test.json",
						Format:  "json",
						Enabled: true,
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid log level",
			config: Config{
				Global: GlobalConfig{
					LogLevel:    "invalid",
					WorkerCount: 4,
				},
				NewRelic: NewRelicConfig{
					APIKey:    "test-key",
					AccountID: "123456",
				},
				APIs: []APIConfig{
					{
						Name:    "test-api",
						URL:     "https://example.com/test.json",
						Format:  "json",
						Enabled: true,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.setDefaults()
			if err != nil {
				t.Fatalf("setDefaults failed: %v", err)
			}

			err = tt.config.validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, but got: %v", err)
			}
		})
	}
}

func TestGetEnabledAPIs(t *testing.T) {
	config := Config{
		APIs: []APIConfig{
			{Name: "enabled-1", Enabled: true},
			{Name: "disabled", Enabled: false},
			{Name: "enabled-2", Enabled: true},
		},
	}

	enabled := config.GetEnabledAPIs()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled APIs, got %d", len(enabled))
	}

	if enabled[0].Name != "enabled-1" || enabled[1].Name != "enabled-2" {
		t.Error("GetEnabledAPIs returned unexpected APIs")
	}
}