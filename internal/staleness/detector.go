package staleness

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Detector handles file staleness detection
type Detector struct {
	client *http.Client
	logger *logrus.Logger
}

// NewDetector creates a new staleness detector
func NewDetector(logger *logrus.Logger) *Detector {
	return &Detector{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Result represents the result of staleness detection
type Result struct {
	IsStale      bool
	FileAge      time.Duration
	LastModified time.Time
	Threshold    time.Duration
	Behavior     string
	ShouldSkip   bool
	ShouldAlert  bool
	Error        error
}

// CheckStaleness checks if a file is stale based on its last modification time
func (d *Detector) CheckStaleness(urlStr string, threshold time.Duration, behavior string) *Result {
	result := &Result{
		Threshold: threshold,
		Behavior:  behavior,
	}

	// Validate URL before making request
	if err := d.validateURL(urlStr); err != nil {
		result.Error = fmt.Errorf("invalid URL: %w", err)
		d.logger.WithError(err).WithField("url", urlStr).Error("URL validation failed")
		return result
	}

	// Get the last modified time from HTTP headers
	lastModified, err := d.getLastModified(urlStr)
	if err != nil {
		result.Error = fmt.Errorf("failed to get last modified time: %w", err)
		d.logger.WithError(err).WithField("url", urlStr).Error("Failed to check file staleness")
		return result
	}

	result.LastModified = lastModified
	result.FileAge = time.Since(lastModified)
	result.IsStale = result.FileAge > threshold

	if result.IsStale {
		d.logger.WithFields(logrus.Fields{
			"url":           urlStr,
			"file_age":      result.FileAge,
			"threshold":     threshold,
			"last_modified": lastModified,
			"behavior":      behavior,
		}).Warn("File is stale")

		switch behavior {
		case "skip":
			result.ShouldSkip = true
			d.logger.WithField("url", urlStr).Info("Skipping stale file processing")
		case "alert":
			result.ShouldAlert = true
			d.logger.WithField("url", urlStr).Info("Will generate alert for stale file")
		case "continue":
			d.logger.WithField("url", urlStr).Info("Continuing to process stale file")
		}
	} else {
		d.logger.WithFields(logrus.Fields{
			"url":           urlStr,
			"file_age":      result.FileAge,
			"threshold":     threshold,
			"last_modified": lastModified,
		}).Debug("File is fresh")
	}

	return result
}

// getLastModified retrieves the last modified time of a file via HTTP HEAD request
func (d *Detector) getLastModified(url string) (time.Time, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	start := time.Now()
	resp, err := d.client.Do(req)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to execute HEAD request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	d.logger.WithFields(logrus.Fields{
		"url":      url,
		"duration": duration,
		"status":   resp.StatusCode,
	}).Debug("HEAD request completed")

	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Try to parse Last-Modified header
	lastModifiedStr := resp.Header.Get("Last-Modified")
	if lastModifiedStr == "" {
		// Fallback to current time if Last-Modified header is not present
		d.logger.WithField("url", url).Warn("Last-Modified header not found, using current time")
		return time.Now(), nil
	}

	// Parse the Last-Modified header (RFC 1123 format)
	lastModified, err := time.Parse(time.RFC1123, lastModifiedStr)
	if err != nil {
		// Try alternative formats
		formats := []string{
			time.RFC1123Z,
			time.RFC822,
			time.RFC822Z,
			"Mon, 02 Jan 2006 15:04:05 GMT",
		}

		for _, format := range formats {
			if parsed, parseErr := time.Parse(format, lastModifiedStr); parseErr == nil {
				lastModified = parsed
				err = nil
				break
			}
		}

		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse Last-Modified header '%s': %w", lastModifiedStr, err)
		}
	}

	return lastModified, nil
}

// CheckMultiple checks staleness for multiple URLs concurrently
func (d *Detector) CheckMultiple(checks []StalenessCheck) []Result {
	results := make([]Result, len(checks))
	resultChan := make(chan indexedResult, len(checks))

	// Start concurrent checks
	for i, check := range checks {
		go func(index int, c StalenessCheck) {
			result := d.CheckStaleness(c.URL, c.Threshold, c.Behavior)
			resultChan <- indexedResult{Index: index, Result: *result}
		}(i, check)
	}

	// Collect results
	for i := 0; i < len(checks); i++ {
		indexed := <-resultChan
		results[indexed.Index] = indexed.Result
	}

	return results
}

// StalenessCheck represents a staleness check configuration
type StalenessCheck struct {
	URL       string
	Threshold time.Duration
	Behavior  string
}

// indexedResult is used for concurrent processing
type indexedResult struct {
	Index  int
	Result Result
}

// validateURL validates the URL format and scheme
func (d *Detector) validateURL(urlStr string) error {
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include a scheme (http or https)")
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme '%s', only http and https are supported", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// GetMetrics returns performance metrics for the detector
func (d *Detector) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"client_timeout": d.client.Timeout.Seconds(),
		"timestamp":      time.Now().Unix(),
	}
}
