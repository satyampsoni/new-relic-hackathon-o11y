# New Relic Flex Integration Setup

This directory contains the New Relic Flex configuration for collecting staleness metrics from Enhanced Flex Monitor.

## Files

- `flex-integration.yml`: New Relic Flex configuration file

## Setup Instructions

### Prerequisites

- New Relic Infrastructure Agent installed
- Enhanced Flex Monitor running on localhost:8080
- Valid New Relic Ingest License Key

### Installation

1. **Copy Flex Configuration**
   ```bash
   sudo cp flex-integration.yml /etc/newrelic-infra/integrations.d/
   ```

2. **Restart Infrastructure Agent**
   ```bash
   sudo systemctl restart newrelic-infra
   ```

3. **Verify Configuration**
   ```bash
   sudo tail -f /var/log/newrelic-infra/newrelic-infra.log
   ```

### Verification

Check that data is flowing to New Relic:

```bash
# Test Enhanced Flex Monitor endpoints
curl http://localhost:8080/api/staleness/status | jq '.'
curl http://localhost:8080/api/health | jq '.'

# Query data in New Relic
SELECT * FROM StalenessMetrics SINCE 10 minutes ago
```

## Data Collection

New Relic Flex collects data from these Enhanced Flex Monitor endpoints:

- `/api/staleness/status` - Current staleness status for all monitored APIs
- `/api/health` - Health status and response times for all APIs

## Troubleshooting

**No data in New Relic?**
1. Ensure Enhanced Flex Monitor is running on port 8080
2. Check Infrastructure Agent logs: `sudo tail -f /var/log/newrelic-infra/newrelic-infra.log`
3. Verify endpoints are accessible: `curl http://localhost:8080/api/health`
4. Confirm valid Ingest License Key is configured

**Flex integration not working?**
1. Verify file permissions: `ls -la /etc/newrelic-infra/integrations.d/`
2. Restart Infrastructure Agent: `sudo systemctl restart newrelic-infra`
3. Check YAML syntax in `flex-integration.yml`
- New Relic agent runs with root privileges for system metrics collection
- License keys are stored in environment variables and config files

## Next Steps

1. Monitor your dashboard for incoming data (2-3 minutes after setup)
2. Create alert conditions based on staleness metrics
3. Set up notification channels (Slack, email, PagerDuty)
4. Customize the metrics collected by modifying the API endpoints
5. Scale the monitoring by adding more API endpoints to your config
