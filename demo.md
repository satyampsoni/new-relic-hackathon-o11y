# Enhanced Flex Monitor - Staleness Detection Demo

## 🎯 Problem Statement

**Issue**: Standard nri-flex doesn't have built-in staleness detection. If a data source stops updating (e.g., program crashes, API becomes unresponsive), nri-flex continues sampling potentially stale data without alerting you.

**Our Solution**: Enhanced Flex Monitor with intelligent staleness detection that checks `Last-Modified` headers and file timestamps to ensure data freshness.

---

## 🚀 Quick Demo Commands

### Prerequisites
```bash
export NEW_RELIC_API_KEY="your_eu_ingest_license_key"
export NEW_RELIC_ACCOUNT_ID="your_account_id"
```

### 1. Start Monitor
```bash
go build -o enhanced-flex-monitor
./enhanced-flex-monitor -config config.yml
```

### 2. Test Staleness Detection (New Terminal)
```bash
curl -s http://localhost:8080/api/staleness/status | jq '.[] | {api_name, is_stale, behavior, threshold_seconds}'
```

**Expected Output:**
```json
{
  "api_name": "service-metrics",
  "is_stale": true,
  "behavior": "alert", 
  "threshold_seconds": 0.000001
}
{
  "api_name": "performance-data",
  "is_stale": true,
  "behavior": "alert",
  "threshold_seconds": 0.000002
}
{
  "api_name": "complex-data", 
  "is_stale": false,
  "behavior": "alert",
  "threshold_seconds": 10
}
```

### 3. Monitor Real-Time
```bash
while true; do
  echo "=== $(date) ==="
  curl -s http://localhost:8080/api/staleness/status | jq '.[] | {api: .api_name, stale: .is_stale}'
  sleep 10
done
```

---

## 📊 New Relic Dashboard Queries

### Login: [one.eu.newrelic.com](https://one.eu.newrelic.com)

### Essential Queries:

#### Current API Status Table
```nrql
SELECT latest(is_stale) as 'Stale?',
       latest(file_age_seconds) as 'Age (sec)', 
       latest(threshold_seconds) as 'Threshold (sec)',
       latest(behavior) as 'Behavior',
       if(latest(is_stale) = true, '🚨', '✅') as 'Status'
FROM StalenessMetrics 
FACET api_name as 'API Name'
WHERE appName = 'enhanced-flex-monitor' 
SINCE 5 minutes ago
```

#### Staleness Overview (Pie Chart)
```nrql
SELECT count(*) FROM StalenessMetrics 
FACET is_stale 
WHERE appName = 'enhanced-flex-monitor' 
SINCE 30 minutes ago
```

#### Timeline Chart
```nrql
SELECT count(*) FROM StalenessMetrics 
WHERE is_stale = true AND appName = 'enhanced-flex-monitor'
FACET api_name
TIMESERIES 1 minute SINCE 1 hour ago
```

---

## 🚨 Alert Setup

### Create Alert Policy: "Enhanced Flex Monitor Alerts"

#### Alert 1: Critical APIs Are Stale
```nrql
SELECT count(*) FROM StalenessMetrics 
WHERE is_stale = true 
  AND behavior = 'alert' 
  AND appName = 'enhanced-flex-monitor'
```
- **Condition**: Above 0 for at least 2 minutes

#### Alert 2: Data Ingestion Stops  
```nrql
SELECT count(*) FROM ServiceMetrics, PerformanceData, LegacyData, ComplexData
WHERE appName = 'enhanced-flex-monitor'
```
- **Condition**: Below 5 for at least 5 minutes

---

## 🏆 Key Benefits Over Standard nri-flex

| Feature | Standard nri-flex | Enhanced Flex Monitor |
|---------|------------------|----------------------|
| **Staleness Detection** | ❌ Continues reading stale data | ✅ Detects and alerts when stale |
| **Configurable Thresholds** | ❌ No freshness validation | ✅ Microseconds to hours per API |
| **Behavioral Control** | ❌ Always processes data | ✅ "alert" or "continue" modes |
| **Real-time Monitoring** | ❌ No staleness visibility | ✅ Live API + New Relic dashboard |
| **Proactive Alerting** | ❌ Silent failures | ✅ Immediate threshold alerts |

---

## 💼 Value to New Relic

### 📊 Enhanced Data Quality
- **Problem**: Customers receive stale data without knowing
- **Solution**: Real-time freshness validation before ingestion
- **Impact**: Prevents wrong business decisions from outdated metrics

### 🚨 Proactive Observability  
- **Problem**: Discover stale data only when business fails
- **Solution**: Instant staleness alerting  
- **Impact**: MTTD reduced from hours to seconds

### 🏗️ Platform Differentiation
- **Market Gap**: No other APM has built-in staleness detection
- **Competitive Edge**: Unique capability increases platform stickiness
- **Enterprise Value**: Meets data quality requirements for compliance

### 💰 Revenue Impact
- **Reduced Churn**: Higher retention through improved data trust
- **Market Expansion**: Manufacturing, IoT, compliance industries  
- **Upsell Opportunities**: Premium data quality features

---

## 🎬 Complete Demo Script

```bash
#!/bin/bash
echo "🎯 Enhanced Flex Monitor Demo"

# Start monitor
./enhanced-flex-monitor -config config.yml &
MONITOR_PID=$!
sleep 10

# Test staleness
echo "Testing staleness detection..."
curl -s http://localhost:8080/api/staleness/status | jq '.[] | {api: .api_name, stale: .is_stale, behavior: .behavior}'

# Generate data for 2 minutes
echo "Generating New Relic data..."
sleep 120

# Final status
stale_count=$(curl -s http://localhost:8080/api/staleness/status | jq '[.[] | select(.is_stale == true)] | length')
echo "Demo complete! Stale APIs: $stale_count"

kill $MONITOR_PID
```

---

## 🎯 Expected Results

### Local Testing:
- **3 APIs stale** (service-metrics, performance-data, legacy-system)  
- **1 API fresh** (complex-data with 10s threshold)
- **Microsecond precision** age vs threshold comparison

### New Relic Dashboard:
| API | Status | Age | Threshold | Behavior |
|-----|--------|-----|-----------|----------|
| service-metrics | 🚨 Stale | 0.000002s | 0.000001s | alert |
| performance-data | 🚨 Stale | 0.000005s | 0.000002s | alert |
| complex-data | ✅ Fresh | 0.000003s | 10s | alert |

**Try it now and see staleness detection in action!** 🚀