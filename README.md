# Enhanced Flex Monitor

> **Solving the Critical Data Freshness Gap in New Relic Flex**

## The Missing Functionality

> *"I have a file on disk which gets written to every 30 seconds. I would like to have nri-flex skip sampling from the file if it hasn't been modified within a configurable time. This is to prevent potentially inaccurate data from being sampled if, say, the program writing the file crashes."*

This is inspired by [Option to skip sample collection if file write time is too old](https://github.com/newrelic/nri-flex/issues/509)

**This functionality doesn't exist in New Relic Flex today.**

### The Real-World Problem
- **File-based monitoring** with nri-flex collecting data every 30 seconds
- **Source application crashes** but file remains on disk with stale data  
- **nri-flex continues sampling** outdated information indefinitely
- **False metrics** lead to incorrect dashboards and poor business decisions

Enhanced Flex Monitor fills this critical gap by providing intelligent staleness detection that prevents ingestion of outdated data before it reaches New Relic.

---

**Production-ready API staleness detection for New Relic observability pipelines**

A production-ready solution that monitors API endpoints and files for data staleness, preventing inaccurate metrics from being ingested when source systems become unresponsive or return outdated information.

## Key Features

- **Staleness Detection**: Monitors Last-Modified headers and timestamps to detect stale data
- **HTTP Metrics Server**: Built-in HTTP server (port 8080) exposing comprehensive monitoring endpoints
- **Configurable Thresholds**: Set custom staleness limits per API/endpoint  
- **Flexible Behaviors**: Skip, alert, or continue processing when staleness is detected
- **New Relic Integration**: Native integration with New Relic Flex for seamless data collection
- **Environment Configuration**: Support for .env files with secure credential management
- **Enhanced Logging**: Pretty console output with colored logs and startup banner
- **Real-time Monitoring**: Live dashboards and alerting for data freshness status
- **Production Ready**: Built with reliability, performance, and observability in mind

## Architecture

Enhanced Flex Monitor works in conjunction with New Relic Flex to provide comprehensive data freshness monitoring:

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   Enhanced Flex     │    │    New Relic        │    │    New Relic        │
│     Monitor         │────│       Flex          │────│     Platform        │
│                     │    │                     │    │                     │
│ • Staleness checks  │    │ • Data collection   │    │ • Metrics storage   │
│ • API monitoring    │    │ • Data transformation│    │ • Dashboards        │
│ • Alert generation  │    │ • Scheduled execution│    │ • Alerting          │
│ • Data validation   │    │ • Error handling    │    │ • Analytics         │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
```
## HTTP Monitoring Endpoints

Enhanced Flex Monitor provides a built-in HTTP server (port 8080) with comprehensive monitoring endpoints:

| Endpoint | Description | Use Case |
|----------|-------------|----------|
| `/api/staleness/status` | Current staleness metrics for all APIs | New Relic Flex integration |
| `/api/health` | API health status with response times | Service health monitoring |
| `/api/alerts/summary` | Alert statistics and history | Alert management dashboard |
| `/api/system/stats` | System performance (CPU, memory, goroutines) | Infrastructure monitoring |
| `/api/performance/summary` | API performance metrics | Performance analysis |
| `/api/config/validate` | Configuration validation status | Configuration verification |
| `/api/test/generate-data` | Test data generation endpoint | Testing and debugging |

### Example Response

```bash
# Check staleness status
curl http://localhost:8080/api/staleness/status | jq

# Monitor system health  
curl http://localhost:8080/api/health | jq

# View alert summary
curl http://localhost:8080/api/alerts/summary | jq
```

## New Relic Flex Integration

Enhanced Flex Monitor integrates with New Relic Flex through configuration-driven data collection:

### Flex Configuration (`configs/flex-integration.yml`)

```yaml
integrations:
  - name: nri-flex
    interval: 30s
    config:
      name: enhanced-flex-monitor
      apis:
        - name: staleness_metrics
          url: http://localhost:8080/api/staleness/status
          custom_attributes:
            service: enhanced-flex-monitor
            
        - name: health_metrics
          url: http://localhost:8080/api/health
          custom_attributes:
            service: enhanced-flex-monitor
            
        - name: alert_summary
          url: http://localhost:8080/api/alerts/summary
          custom_attributes:
            service: enhanced-flex-monitor
            
        - name: system_stats
          url: http://localhost:8080/api/system/stats
          custom_attributes:
            service: enhanced-flex-monitor
            
        - name: performance_summary
          url: http://localhost:8080/api/performance/summary
          custom_attributes:
            service: enhanced-flex-monitor
```

### Data Flow

1. **Enhanced Flex Monitor** runs staleness checks and exposes metrics via HTTP endpoints
2. **New Relic Flex** collects data every 30 seconds from these endpoints  
3. **Data transformation** occurs using JQ filters to structure metrics
4. **New Relic Platform** ingests the processed staleness metrics and logs
5. **Dashboards and alerts** provide visibility into data freshness status

Note: Enhanced Flex Monitor sits between your data source and New Relic Flex. It validates freshness before Flex ingests anything, acting as an automated “data quality filter.”

## Quick Start

### Prerequisites

- Go 1.19+
- New Relic account with Ingest License Key
- New Relic Infrastructure Agent (for Flex integration)

### Installation

1. **Clone and build**
   ```bash
   git clone https://github.com/satyampsoni/new-relic-hackathon-o11y.git
   cd new-relic-hackathon-o11y
   make build
   ```

2. **Configure environment**
   ```bash
   # Copy environment template
   cp .env.example .env
   
   # Edit .env with your New Relic credentials
   NEW_RELIC_API_KEY="your_ingest_license_key"
   NEW_RELIC_ACCOUNT_ID="your_account_id"
   NEW_RELIC_REGION="EU"  # or "US"
   ```

3. **Run the monitor**
   ```bash
   ./enhanced-flex-monitor -config config.yml
   ```

   The application will display a startup banner and begin monitoring:
   
   ```
   ╔═══════════════════════════════════════════════════════════════════╗
   ║                   🚀 Enhanced Flex Monitor                       ║
   ║              New Relic Data Staleness Detection                  ║
   ╚═══════════════════════════════════════════════════════════════════╝

   15:04:05  INFO Starting Enhanced Flex Monitor version=1.0.0 config=config.yml
   15:04:05  INFO Configuration loaded apis=5 interval=30s region=EU
   15:04:05  INFO HTTP server will start on port 8080 port=8080
   15:04:05  INFO Starting API processing cycle api_count=5
   ```

### Logging Configuration

Enhanced Flex Monitor supports two logging formats:

- **Development (default)**: Pretty colored console output with timestamps
- **Production**: Structured JSON logging for log aggregation

```bash
# Pretty console logging (default)
LOG_FORMAT=text ./enhanced-flex-monitor -config config.yml

# JSON logging for production
LOG_FORMAT=json ./enhanced-flex-monitor -config config.yml
```

## Configuration

### Basic Configuration (`config.yml`)

```yaml
global:
  name: "enhanced-flex-monitor"
  interval: 30s
  log_level: "info"
  enable_metrics: true
  enable_alerts: true
  worker_count: 4

newrelic:
  api_key: "${NEW_RELIC_API_KEY}"        # From .env file
  account_id: "${NEW_RELIC_ACCOUNT_ID}"  # From .env file  
  region: "EU"  # or "US"

alerts:
  channels:
    - name: "log-alerts"
      type: "log"
      enabled: true

apis:
  - name: "service-metrics"
    url: "https://jsonplaceholder.typicode.com/posts/1"
    fallback_url: "https://httpbin.org/json"
    format: "json"
    jq: ". | {id, title, userId}"
    event_type: "ServiceMetrics"
    enabled: true
    attributes:
      service: "example-api"
      environment: "production"
    staleness:
      enabled: true
      threshold: 5m
      behavior: "alert"  # skip, alert, continue

### Staleness Behaviors

- **`skip`**: Don't process stale data (recommended for critical metrics)
- **`alert`**: Process but generate alerts for stale data  
- **`continue`**: Process all data but log staleness warnings

## Staleness Detection

### Detection Methods

- **Last-Modified Headers**: Analyzes HTTP response headers
- **File Timestamps**: Monitors file modification times
- **Response Patterns**: Detects cached or repeated responses

### Behavior Options

- **`skip`**: Don't process stale data (recommended for critical metrics)
- **`alert`**: Process data but generate alerts for staleness
- **`continue`**: Process all data with staleness warnings

### Threshold Configuration

```yaml
apis:
  - name: "critical-service"
    staleness:
      threshold: "5m"    # 5 minutes
      behavior: "skip"   # Stop processing stale data
      
  - name: "monitoring-service"  
    staleness:
      threshold: "1h"    # 1 hour
      behavior: "alert"  # Alert but continue processing
```

## Monitoring & Dashboards  

### Key Metrics

| Metric | Description | Type | Endpoint |
|--------|-------------|------|----------|
| `staleness_check_count` | Total staleness checks performed | Counter | `/api/staleness/status` |
| `stale_data_detected` | APIs returning stale data | Gauge | `/api/staleness/status` |
| `api_response_time_ms` | API response time in milliseconds | Gauge | `/api/health` |
| `staleness_threshold_ratio` | Data age vs threshold ratio | Gauge | `/api/staleness/status` |
| `system_cpu_usage` | CPU usage percentage | Gauge | `/api/system/stats` |
| `system_memory_usage` | Memory usage in MB | Gauge | `/api/system/stats` |
| `alert_count_total` | Total alerts generated | Counter | `/api/alerts/summary` |
| `performance_success_rate` | API success rate percentage | Gauge | `/api/performance/summary` |

### Sample NRQL Queries

```sql
-- Overall staleness status
SELECT count(*) as 'Total APIs', 
       filter(count(*), WHERE is_stale = true) as 'Stale APIs'
FROM StalenessMetrics SINCE 30 minutes ago

-- Response time by staleness status  
SELECT average(response_time_ms) as 'Avg Response Time'
FROM StalenessMetrics 
FACET if(is_stale, 'Stale', 'Fresh') 
SINCE 1 hour ago

-- Staleness timeline
SELECT count(*) FROM StalenessMetrics 
FACET is_stale 
SINCE 4 hours ago TIMESERIES 10 minutes
```
## Use Cases

### Production Monitoring
Monitor critical APIs to ensure dashboards display accurate, real-time data rather than stale cached responses.

### Data Quality Assurance  
Validate data freshness before ingestion to maintain metric accuracy and prevent false alerting.

### SLA Compliance
Track and report on data freshness SLAs, ensuring upstream services meet contractual obligations.

### Dependency Monitoring
Detect when external APIs or services start returning outdated information due to failures.


### Building

```bash
# Development build
make build

# Run tests
make test

# Docker build
make docker-build
docker run -e NEW_RELIC_API_KEY=your_key enhanced-flex-monitor

# Clean build artifacts
make clean
```

### Configuration Options

```yaml
global:
  name: "enhanced-flex-monitor"  # Service name
  interval: 30s                  # Monitoring frequency
  log_level: "info"              # Logging verbosity (debug, info, warn, error)
  worker_count: 4                # Concurrent workers
  enable_metrics: true           # Metrics collection
  enable_alerts: true            # Alert generation

newrelic:
  api_key: "${NEW_RELIC_API_KEY}"     # Ingest license key
  account_id: "${NEW_RELIC_ACCOUNT_ID}" # Account ID
  region: "EU"                        # US or EU
  
alerts:
  channels:
    - name: "log-alerts"
      type: "log"
      enabled: true

# Environment variables (in .env file)
LOG_FORMAT=text              # text (colored) or json (structured)
PORT=8080                   # HTTP server port
NEW_RELIC_REGION=EU         # Override config region
```

### Command Line Options

```bash
# Basic usage
./enhanced-flex-monitor -config config.yml

# Override log level
./enhanced-flex-monitor -config config.yml -log-level debug

# Show version
./enhanced-flex-monitor -version

# Using environment variables
PORT=9090 LOG_LEVEL=debug ./enhanced-flex-monitor -config config.yml
```


- **New Relic Docs**: [Flex Integration Guide](https://docs.newrelic.com/docs/infrastructure/host-integrations/host-integrations-list/flex-integration-tool-build-your-own-integration/)

---

Built with :love for New Relic hackathon
