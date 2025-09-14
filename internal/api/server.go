package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/alerts"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/config"
	"github.com/satyampsoni/new-relic-hackathon-o11y/internal/staleness"
	"github.com/sirupsen/logrus"
)

// Server wraps the HTTP server for metrics
type Server struct {
	server  *http.Server
	handler *MetricsHandler
	logger  *logrus.Logger
}

// NewServer creates a new metrics server
func NewServer(port int, detector *staleness.Detector, alertManager *alerts.Manager, cfg *config.Config, logger *logrus.Logger) *Server {
	handler := NewMetricsHandler(detector, alertManager, cfg, logger)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return &Server{
		server:  server,
		handler: handler,
		logger:  logger,
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	s.logger.WithField("port", s.server.Addr).Info("Starting metrics server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	return nil
}

// Stop gracefully stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping metrics server")
	return s.server.Shutdown(ctx)
}
