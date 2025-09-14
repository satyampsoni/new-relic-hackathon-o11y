package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/alerts"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/metrics"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/processor"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/staleness"
	"github.com/sirupsen/logrus"
)

const (
	version = "1.0.0"
	appName = "Enhanced Flex Monitor"
)

var (
	configPath  = flag.String("config", "config.yml", "Path to configuration file")
	showVersion = flag.Bool("version", false, "Show version information")
	logLevel    = flag.String("log-level", "", "Override log level")
)

type Application struct {
	config            *config.Config
	logger            *logrus.Logger
	metricsCollector  *metrics.Collector
	alertManager      *alerts.Manager
	stalenessDetector *staleness.Detector
	fileProcessor     *processor.FileProcessor
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s v%s\n", appName, version)
		os.Exit(0)
	}
	app, err := initializeApplication()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	app.logger.WithFields(logrus.Fields{
		"version":  version,
		"config":   *configPath,
		"apis":     len(app.config.APIs),
		"interval": app.config.Global.Interval,
	}).Info("Starting Enhanced Flex Monitor")

	app.setupGracefulShutdown()
	app.run()
	app.logger.Info("Enhanced Flex Monitor stopped")
}

func initializeApplication() (*Application, error) {
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	if *logLevel != "" {
		cfg.Global.LogLevel = *logLevel
	}

	level, err := logrus.ParseLevel(cfg.Global.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// Initialize components
	metricsCollector := metrics.NewCollector(cfg.NewRelic, logger)
	alertManager := alerts.NewManager(cfg.GetEnabledAlertChannels(), logger)
	stalenessDetector := staleness.NewDetector(logger)
	fileProcessor := processor.NewFileProcessor(logger, metricsCollector, stalenessDetector)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		config:            cfg,
		logger:            logger,
		metricsCollector:  metricsCollector,
		alertManager:      alertManager,
		stalenessDetector: stalenessDetector,
		fileProcessor:     fileProcessor,
		ctx:               ctx,
		cancel:            cancel,
	}

	return app, nil
}

// setupGracefulShutdown configures signal handling for graceful shutdown
func (app *Application) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-c
		app.logger.WithField("signal", sig).Info("Received shutdown signal")
		app.cancel()
	}()
}

// run starts the main processing loop
func (app *Application) run() {
	ticker := time.NewTicker(app.config.Global.Interval)
	defer ticker.Stop()

	// Send startup health alert
	if app.config.Global.EnableAlerts {
		app.alertManager.SendHealthAlert("enhanced-flex-monitor", "started", map[string]interface{}{
			"version":  version,
			"apis":     len(app.config.APIs),
			"interval": app.config.Global.Interval.String(),
		})
	}

	// Initial processing
	app.processAPIs()

	for {
		select {
		case <-app.ctx.Done():
			app.logger.Info("Shutdown requested, stopping processing")
			app.shutdown()
			return
		case <-ticker.C:
			app.processAPIs()
		}
	}
}

// processAPIs processes all configured APIs
func (app *Application) processAPIs() {
	start := time.Now()
	enabledAPIs := app.config.GetEnabledAPIs()

	if len(enabledAPIs) == 0 {
		app.logger.Warn("No enabled APIs found")
		return
	}

	app.logger.WithField("api_count", len(enabledAPIs)).Info("Starting API processing cycle")

	// Process APIs concurrently
	results := app.fileProcessor.ProcessAPIs(enabledAPIs, app.config.Global.WorkerCount)

	// Analyze results and send alerts if needed
	var totalRecords int
	var errors []error
	var staleCount int

	for _, result := range results {
		totalRecords += result.RecordCount

		if result.IsStale {
			staleCount++
		}

		if result.HasError && result.Error != nil {
			errors = append(errors, result.Error)

			// Send error alert if alerts are enabled
			if app.config.Global.EnableAlerts {
				app.alertManager.SendErrorAlert(result.APIName, "processing", result.Error)
			}
		}

		// Send staleness alert if needed
		if result.IsStale && app.config.Global.EnableAlerts {
			// Find the API config to get staleness details
			for _, api := range enabledAPIs {
				if api.Name == result.APIName && api.Staleness.Behavior == "alert" {
					// We need more details for a proper alert, but this is the structure
					app.alertManager.SendStalenessAlert(api.Name, api.URL, 0, api.Staleness.Threshold)
					break
				}
			}
		}
	}

	duration := time.Since(start)

	// Send metrics if enabled
	if app.config.Global.EnableMetrics {
		app.sendCycleMetrics(duration, totalRecords, len(errors), staleCount)

		// Send batch to New Relic
		if err := app.metricsCollector.SendBatch(); err != nil {
			app.logger.WithError(err).Error("Failed to send metrics batch")
		}
	}

	app.logger.WithFields(logrus.Fields{
		"duration":      duration,
		"total_records": totalRecords,
		"errors":        len(errors),
		"stale_count":   staleCount,
		"api_count":     len(enabledAPIs),
	}).Info("API processing cycle completed")

	// Log errors
	for _, err := range errors {
		app.logger.WithError(err).Error("Processing error occurred")
	}
}

// sendCycleMetrics sends processing cycle metrics
func (app *Application) sendCycleMetrics(duration time.Duration, recordCount, errorCount, staleCount int) {
	attributes := map[string]interface{}{
		"service.name": app.config.Global.Name,
		"version":      version,
	}

	app.metricsCollector.AddMetric("flex.cycle.duration", "gauge", duration.Seconds(), attributes)
	app.metricsCollector.AddMetric("flex.cycle.records", "count", float64(recordCount), attributes)
	app.metricsCollector.AddMetric("flex.cycle.errors", "count", float64(errorCount), attributes)
	app.metricsCollector.AddMetric("flex.cycle.stale_files", "count", float64(staleCount), attributes)
}

// shutdown performs graceful shutdown
func (app *Application) shutdown() {
	app.logger.Info("Starting graceful shutdown")

	// Send final metrics batch
	if app.config.Global.EnableMetrics {
		if err := app.metricsCollector.SendBatch(); err != nil {
			app.logger.WithError(err).Error("Failed to send final metrics batch")
		}
	}

	// Send shutdown alert
	if app.config.Global.EnableAlerts {
		app.alertManager.SendHealthAlert("enhanced-flex-monitor", "stopped", map[string]interface{}{
			"version": version,
			"uptime":  time.Since(time.Now()).String(),
		})
	}

	// Wait for any ongoing operations
	app.wg.Wait()

	app.logger.Info("Graceful shutdown completed")
}
