package staleness

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestStalenessDetector(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	detector := NewDetector(logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}

		// Set Last-Modified header to 10 minutes ago
		lastModified := time.Now().Add(-10 * time.Minute)
		w.Header().Set("Last-Modified", lastModified.Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name           string
		threshold      time.Duration
		behavior       string
		expectedStale  bool
		expectedSkip   bool
		expectedAlert  bool
	}{
		{
			name:          "fresh file with skip behavior",
			threshold:     15 * time.Minute,
			behavior:      "skip",
			expectedStale: false,
			expectedSkip:  false,
			expectedAlert: false,
		},
		{
			name:          "stale file with skip behavior",
			threshold:     5 * time.Minute,
			behavior:      "skip",
			expectedStale: true,
			expectedSkip:  true,
			expectedAlert: false,
		},
		{
			name:          "stale file with alert behavior",
			threshold:     5 * time.Minute,
			behavior:      "alert",
			expectedStale: true,
			expectedSkip:  false,
			expectedAlert: true,
		},
		{
			name:          "stale file with continue behavior",
			threshold:     5 * time.Minute,
			behavior:      "continue",
			expectedStale: true,
			expectedSkip:  false,
			expectedAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.CheckStaleness(server.URL, tt.threshold, tt.behavior)

			if result.Error != nil {
				t.Fatalf("Unexpected error: %v", result.Error)
			}

			if result.IsStale != tt.expectedStale {
				t.Errorf("Expected IsStale=%v, got %v", tt.expectedStale, result.IsStale)
			}

			if result.ShouldSkip != tt.expectedSkip {
				t.Errorf("Expected ShouldSkip=%v, got %v", tt.expectedSkip, result.ShouldSkip)
			}

			if result.ShouldAlert != tt.expectedAlert {
				t.Errorf("Expected ShouldAlert=%v, got %v", tt.expectedAlert, result.ShouldAlert)
			}

			if result.Threshold != tt.threshold {
				t.Errorf("Expected Threshold=%v, got %v", tt.threshold, result.Threshold)
			}

			if result.Behavior != tt.behavior {
				t.Errorf("Expected Behavior=%s, got %s", tt.behavior, result.Behavior)
			}

			// File age should be approximately 10 minutes
			expectedAge := 10 * time.Minute
			tolerance := 1 * time.Minute
			if result.FileAge < expectedAge-tolerance || result.FileAge > expectedAge+tolerance {
				t.Errorf("Expected FileAge around %v, got %v", expectedAge, result.FileAge)
			}
		})
	}
}

func TestStalenessDetectorErrors(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	detector := NewDetector(logger)

	// Test with invalid URL
	result := detector.CheckStaleness("invalid-url", 5*time.Minute, "skip")
	if result.Error == nil {
		t.Error("Expected error for invalid URL, got none")
	}

	// Test with server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result = detector.CheckStaleness(server.URL, 5*time.Minute, "skip")
	if result.Error == nil {
		t.Error("Expected error for 500 status, got none")
	}
}

func TestStalenessDetectorNoLastModified(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	detector := NewDetector(logger)

	// Server without Last-Modified header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := detector.CheckStaleness(server.URL, 5*time.Minute, "skip")
	if result.Error != nil {
		t.Fatalf("Unexpected error: %v", result.Error)
	}

	// Should use current time, so file should be fresh
	if result.IsStale {
		t.Error("Expected fresh file when Last-Modified header is missing")
	}
}

func TestCheckMultiple(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	detector := NewDetector(logger)

	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastModified := time.Now().Add(-2 * time.Minute)
		w.Header().Set("Last-Modified", lastModified.Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastModified := time.Now().Add(-10 * time.Minute)
		w.Header().Set("Last-Modified", lastModified.Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	checks := []StalenessCheck{
		{URL: server1.URL, Threshold: 5 * time.Minute, Behavior: "skip"},
		{URL: server2.URL, Threshold: 5 * time.Minute, Behavior: "alert"},
	}

	results := detector.CheckMultiple(checks)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// First server should be fresh
	if results[0].IsStale {
		t.Error("Expected first result to be fresh")
	}

	// Second server should be stale
	if !results[1].IsStale {
		t.Error("Expected second result to be stale")
	}
}