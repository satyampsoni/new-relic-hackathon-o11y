package processor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/metrics"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/staleness"
	"github.com/sirupsen/logrus"
)

// FileProcessor handles file processing and data transformation
type FileProcessor struct {
	client           *http.Client
	logger           *logrus.Logger
	metricsCollector *metrics.Collector
	stalenessDetector *staleness.Detector
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(logger *logrus.Logger, metricsCollector *metrics.Collector, stalenessDetector *staleness.Detector) *FileProcessor {
	return &FileProcessor{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger:            logger,
		metricsCollector:  metricsCollector,
		stalenessDetector: stalenessDetector,
	}
}

// ProcessResult represents the result of file processing
type ProcessResult struct {
	APIName     string
	RecordCount int
	Duration    time.Duration
	IsStale     bool
	HasError    bool
	Error       error
	Samples     []map[string]interface{}
}

// ProcessAPI processes a single API configuration
func (fp *FileProcessor) ProcessAPI(api config.APIConfig) *ProcessResult {
	start := time.Now()
	result := &ProcessResult{
		APIName: api.Name,
	}

	fp.logger.WithField("api", api.Name).Info("Starting API processing")

	// Check staleness if enabled
	if api.Staleness.Enabled {
		stalenessResult := fp.stalenessDetector.CheckStaleness(
			api.Staleness.CheckURL,
			api.Staleness.Threshold,
			api.Staleness.Behavior,
		)

		result.IsStale = stalenessResult.IsStale

		if stalenessResult.Error != nil {
			result.Error = fmt.Errorf("staleness check failed: %w", stalenessResult.Error)
			result.HasError = true
			fp.recordMetrics(result, time.Since(start))
			return result
		}

		// Record staleness metrics
		fp.metricsCollector.RecordStalenessMetrics(
			api.Name,
			stalenessResult.FileAge,
			stalenessResult.Threshold,
			stalenessResult.IsStale,
		)

		// Handle staleness behavior
		if result.IsStale && stalenessResult.ShouldSkip {
			fp.logger.WithField("api", api.Name).Info("Skipping processing due to stale file")
			fp.recordMetrics(result, time.Since(start))
			return result
		}
	}

	// Fetch and process data
	data, err := fp.fetchData(api.URL)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch data: %w", err)
		result.HasError = true
		fp.recordMetrics(result, time.Since(start))
		return result
	}

	// Process data based on format
	var samples []map[string]interface{}
	switch strings.ToLower(api.Format) {
	case "json":
		samples, err = fp.processJSON(data, api)
	case "csv":
		samples, err = fp.processCSV(data, api)
	default:
		err = fmt.Errorf("unsupported format: %s", api.Format)
	}

	if err != nil {
		result.Error = fmt.Errorf("failed to process data: %w", err)
		result.HasError = true
		fp.recordMetrics(result, time.Since(start))
		return result
	}

	result.Samples = samples
	result.RecordCount = len(samples)
	result.Duration = time.Since(start)

	// Send samples to New Relic
	fp.sendSamplesToNewRelic(samples, api.EventType)

	fp.logger.WithFields(logrus.Fields{
		"api":          api.Name,
		"record_count": result.RecordCount,
		"duration":     result.Duration,
		"is_stale":     result.IsStale,
	}).Info("API processing completed successfully")

	fp.recordMetrics(result, result.Duration)
	return result
}

// fetchData retrieves data from the specified URL
func (fp *FileProcessor) fetchData(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Enhanced-Flex-Monitor/1.0")
	req.Header.Set("Accept", "application/json, text/csv, */*")

	fp.logger.WithField("url", url).Debug("Fetching data")

	resp, err := fp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fp.logger.WithFields(logrus.Fields{
		"url":       url,
		"data_size": len(data),
		"status":    resp.StatusCode,
	}).Debug("Data fetched successfully")

	return data, nil
}

// processJSON processes JSON data with optional JQ transformation
func (fp *FileProcessor) processJSON(data []byte, api config.APIConfig) ([]map[string]interface{}, error) {
	var rawData interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Apply JQ transformation if specified
	if api.JQ != "" {
		transformed, err := fp.applyJQTransformation(rawData, api.JQ)
		if err != nil {
			return nil, fmt.Errorf("JQ transformation failed: %w", err)
		}
		rawData = transformed
	}

	// Convert to samples
	samples, err := fp.convertToSamples(rawData, api)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to samples: %w", err)
	}

	return samples, nil
}

// processCSV processes CSV data
func (fp *FileProcessor) processCSV(data []byte, api config.APIConfig) ([]map[string]interface{}, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = ','
	reader.Comment = '#'

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Use first row as headers
	headers := records[0]
	var samples []map[string]interface{}

	for i, record := range records[1:] {
		if len(record) != len(headers) {
			fp.logger.WithFields(logrus.Fields{
				"row":              i + 2,
				"expected_columns": len(headers),
				"actual_columns":   len(record),
			}).Warn("CSV row column count mismatch, skipping")
			continue
		}

		sample := make(map[string]interface{})
		for j, header := range headers {
			value := record[j]
			
			// Try to convert to number if possible
			if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
				sample[header] = floatVal
			} else if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
				sample[header] = intVal
			} else if boolVal, err := strconv.ParseBool(value); err == nil {
				sample[header] = boolVal
			} else {
				sample[header] = value
			}
		}

		// Add custom attributes
		fp.addCustomAttributes(sample, api)
		samples = append(samples, sample)
	}

	return samples, nil
}

// applyJQTransformation applies JQ transformation to data
func (fp *FileProcessor) applyJQTransformation(data interface{}, jqQuery string) (interface{}, error) {
	query, err := gojq.Parse(jqQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JQ query: %w", err)
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile JQ query: %w", err)
	}

	iter := code.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("JQ execution error: %w", err)
		}
		// Return first result
		return v, nil
	}

	return data, nil
}

// convertToSamples converts raw data to New Relic samples
func (fp *FileProcessor) convertToSamples(data interface{}, api config.APIConfig) ([]map[string]interface{}, error) {
	var samples []map[string]interface{}

	switch v := data.(type) {
	case []interface{}:
		// Array of objects
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				sample := make(map[string]interface{})
				for k, val := range itemMap {
					sample[k] = val
				}
				fp.addCustomAttributes(sample, api)
				samples = append(samples, sample)
			}
		}
	case map[string]interface{}:
		// Single object
		sample := make(map[string]interface{})
		for k, val := range v {
			sample[k] = val
		}
		fp.addCustomAttributes(sample, api)
		samples = append(samples, sample)
	default:
		return nil, fmt.Errorf("unsupported data type for conversion: %T", data)
	}

	return samples, nil
}

// addCustomAttributes adds custom attributes to a sample
func (fp *FileProcessor) addCustomAttributes(sample map[string]interface{}, api config.APIConfig) {
	// Add API attributes
	for k, v := range api.Attributes {
		sample[k] = v
	}

	// Add processing metadata
	sample["api.name"] = api.Name
	sample["processed.timestamp"] = time.Now().Unix()
	sample["processor.version"] = "1.0.0"
}

// sendSamplesToNewRelic sends processed samples to New Relic
func (fp *FileProcessor) sendSamplesToNewRelic(samples []map[string]interface{}, eventType string) {
	for _, sample := range samples {
		fp.metricsCollector.AddEvent(eventType, sample)
	}
}

// recordMetrics records processing metrics
func (fp *FileProcessor) recordMetrics(result *ProcessResult, duration time.Duration) {
	fp.metricsCollector.RecordProcessingMetrics(
		result.APIName,
		duration,
		result.RecordCount,
		result.IsStale,
		result.HasError,
	)
}

// ProcessAPIs processes multiple APIs concurrently
func (fp *FileProcessor) ProcessAPIs(apis []config.APIConfig, maxWorkers int) []*ProcessResult {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}

	jobs := make(chan config.APIConfig, len(apis))
	results := make(chan *ProcessResult, len(apis))

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		go func() {
			for api := range jobs {
				if api.Enabled {
					result := fp.ProcessAPI(api)
					results <- result
				} else {
					// Send empty result for disabled APIs
					results <- &ProcessResult{
						APIName: api.Name,
					}
				}
			}
		}()
	}

	// Send jobs
	for _, api := range apis {
		jobs <- api
	}
	close(jobs)

	// Collect results
	var processResults []*ProcessResult
	for i := 0; i < len(apis); i++ {
		result := <-results
		processResults = append(processResults, result)
	}

	return processResults
}

// GetStats returns processor statistics
func (fp *FileProcessor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"client_timeout": fp.client.Timeout.Seconds(),
		"timestamp":      time.Now().Unix(),
	}
}