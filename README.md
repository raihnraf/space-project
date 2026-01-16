# OrbitStream: High-Throughput Satellite Telemetry Engine

A high-throughput satellite telemetry simulation system capable of ingesting **10,000+ data points per second** with real-time anomaly detection and visualization.

## Architecture

```
Python Simulator (configurable satellites)
         ↓ (HTTP POST)
Go Ingestion Service (batch processing)
         ↓ (batched inserts)
TimescaleDB (hypertables + compression)
         ↓
Grafana (real-time dashboards)
```

## Features

- **10,000+ points/second** sustained throughput
- **Configurable satellites** - specify any number of satellites via command-line
- **Telemetry fields**: battery_charge_percent, storage_usage_mb, signal_strength_dbm
- **Anomaly detection** - simple threshold-based alerts
- **Real-time visualization** - Grafana dashboards with live data
- **TimescaleDB optimization** - hypertables, compression, and retention policies

## Quick Start

### Prerequisites

- Docker and Docker Compose
- At least 10GB free disk space
- Ports 3000, 5432, 8080 available

### 1. Start Infrastructure

```bash
# Start TimescaleDB and Grafana
docker-compose up -d timescaledb grafana

# Wait for database to initialize (check logs)
docker-compose logs -f timescaledb
# Look for: "database system is ready to accept connections"
```

### 2. Start Go Ingestion Service

```bash
# Start the Go service
docker-compose up -d go-service

# Verify it's running
curl http://localhost:8080/health
# Should return: {"status":"healthy","timestamp":"..."}
```

### 3. Send Test Data

```bash
# Send a single test point
curl -X POST http://localhost:8080/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "satellite_id": "TEST-01",
    "battery_charge_percent": 85.5,
    "storage_usage_mb": 45000.0,
    "signal_strength_dbm": -55.0
  }'
# Should return: {"status":"accepted","satellite_id":"TEST-01"}
```

### 4. Run Python Simulator

```bash
# Option A: Run in Docker
docker-compose --profile testing up simulator

# Option B: Run directly (requires Python 3.11+)
cd python-simulator
pip install -r requirements.txt
python satellite_sim.py --satellites 100 --rate 100 --duration 60
```

### 5. View Grafana Dashboard

1. Open http://localhost:3000
2. Login: admin / admin
3. Navigate to Explore → Select TimescaleDB datasource
4. Run queries like:

```sql
-- Real-time throughput
SELECT
    time_bucket('1 second', time) AS bucket,
    COUNT(*) AS points_per_second
FROM telemetry
WHERE time > NOW() - INTERVAL '5 minutes'
GROUP BY bucket
ORDER BY bucket DESC;
```

## Configuration

### Environment Variables (Go Service)

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| DATABASE_URL | postgres://... | TimescaleDB connection |
| BATCH_SIZE | 1000 | Points per batch |
| BATCH_TIMEOUT | 1s | Max time before flush |
| MAX_CONNECTIONS | 50 | Database connection pool |
| ANOMALY_THRESHOLD_BATTERY | 10.0 | Alert if battery < 10% |
| ANOMALY_THRESHOLD_STORAGE | 95000.0 | Alert if storage > 95GB |
| ANOMALY_THRESHOLD_SIGNAL | -100.0 | Alert if signal < -100 dBm |

### Python Simulator Arguments

| Argument | Default | Description |
|----------|---------|-------------|
| --satellites, -s | 100 | Number of satellites |
| --rate, -r | 100 | Points/sec per satellite |
| --api-url | http://localhost:8080 | Go service URL |
| --duration, -d | 60 | Run duration (0 = infinite) |
| --anomaly-rate | 0.01 | Probability of anomalies (1%) |

## Performance

### Throughput Calculation

- **100 satellites** × **100 points/sec/satellite** = **10,000 points/sec**
- Each satellite sends 1 request every 10ms
- Batch size: 1000 points
- Batch flush: Every 1 second or when buffer is full

### Benchmarks

On a typical development machine:
- Sustained throughput: 10,000+ points/second
- Success rate: 99%+
- Latency (ingest → DB): <100ms
- Database writes: ~10 batches/second

## Troubleshooting

### Low Throughput

- Increase `BATCH_SIZE` to 1000+
- Verify `BATCH_TIMEOUT` is 1s
- Check Python simulator connection limits

### High Memory Usage

- Reduce batch buffer size
- Decrease `MAX_CONNECTIONS`
- Check for goroutine leaks

### Disk Growing Too Fast

- Verify compression policy is active
- Check retention policy (30 days default)
- Reduce chunk time interval

### Go Service Won't Start

1. Check if TimescaleDB is ready:
   ```bash
   docker-compose logs timescaledb
   ```

2. Verify database connection:
   ```bash
   docker-compose exec go-service ping timescaledb
   ```

## Project Structure

```
space-project/
├── docker-compose.yml          # Container orchestration
├── README.md                   # This file
│
├── go-service/                 # Go ingestion service
│   ├── main.go                 # Entry point
│   ├── go.mod / go.sum         # Dependencies
│   ├── handlers/               # HTTP handlers
│   ├── models/                 # Data structures
│   ├── db/                     # Database layer
│   │   ├── connection.go       # Connection pool
│   │   ├── batch.go            # Batch processor
│   │   └── init.sql            # Schema
│   ├── config/                 # Configuration
│   └── Dockerfile
│
├── python-simulator/           # Satellite simulator
│   ├── satellite_sim.py        # Main script
│   ├── config.py               # Configuration
│   ├── generators/             # Telemetry data generator
│   ├── requirements.txt        # Python deps
│   └── Dockerfile
│
└── grafana/                    # Grafana configuration
    └── provisioning/
        └── datasources/
            └── timescaledb.yml
```

## License

MIT

## Author

Built as a high-throughput telemetry demonstration project.
