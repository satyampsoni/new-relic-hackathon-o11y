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
- **Configurable Thresholds**: Set custom staleness limits per API/endpoint  
- **Flexible Behaviors**: Skip, alert, or continue processing when staleness is detected
- **New Relic Integration**: Native integration with New Relic Flex for seamless data collection
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
            
        - name: application_logs  
          file: /var/log/enhanced-flex-monitor.log
          split_by: \n
          jq: 'select(.level == "INFO")'
```

### Data Flow

1. **Enhanced Flex Monitor** runs staleness checks and exposes metrics via HTTP endpoints
2. **New Relic Flex** collects data every 30 seconds from these endpoints  
3. **Data transformation** occurs using JQ filters to structure metrics
4. **New Relic Platform** ingests the processed staleness metrics and logs
5. **Dashboards and alerts** provide visibility into data freshness status

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
   export NEW_RELIC_API_KEY="your_ingest_license_key"
   export NEW_RELIC_ACCOUNT_ID="your_account_id"
   ```

3. **Run the monitor**
   ```bash
   ./enhanced-flex-monitor -config config.yml
   ```

## Configuration

### Basic Configuration (`config.yml`)

```yaml
global:
  name: "enhanced-flex-monitor"
  interval: 30s
  log_level: "info"
  worker_count: 4

newrelic:
  api_key: "${NEW_RELIC_API_KEY}"
  account_id: "${NEW_RELIC_ACCOUNT_ID}"
  region: "US"  # or "EU"

alerts:
  channels:
    - name: "log-alerts"
      type: "log"
      enabled: true

apis:
  - name: "user-service"
    url: "https://api.example.com/users"
    format: "json"
    event_type: "UserServiceMetrics"
    staleness:
      enabled: true
      threshold: "10m"
      behavior: "alert"  # skip, alert, continue
```

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

| Metric | Description | Type |
|--------|-------------|------|
| `staleness_check_count` | Total staleness checks performed | Counter |
| `stale_data_detected` | APIs returning stale data | Gauge |
| `api_response_time_ms` | API response time in milliseconds | Gauge |
| `staleness_threshold_ratio` | Data age vs threshold ratio | Gauge |

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
  interval: 30s              # Monitoring frequency
  log_level: "info"          # Logging verbosity
  worker_count: 4            # Concurrent workers
  enable_metrics: true       # Metrics collection
  enable_alerts: true        # Alert generation

newrelic:
  region: "US"               # US or EU
  timeout: 30s               # API timeout
  
alerts:
  channels:
    - name: "slack-ops"
      type: "webhook"
      enabled: true
      settings:
        webhook_url: "${SLACK_WEBHOOK_URL}"
```


- **New Relic Docs**: [Flex Integration Guide](https://docs.newrelic.com/docs/infrastructure/host-integrations/host-integrations-list/flex-integration-tool-build-your-own-integration/)

---

Built with :love for New Relic hackathon
