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
- **Continuous Aggregates** - hierarchical downsampling (5-min, hourly, daily buckets)
- **Tiered Data Retention** - raw: 7 days, hourly: 6 months, daily: 1 year (~90% storage savings)

## Quick Start

### Prerequisites

**For Docker deployment:**
- Docker (with Compose plugin)
- At least 10GB free disk space
- Ports 3000, 5432, 8080 available

**For local development:**
- Go 1.21+ (for Go service)
- Python 3.10+ (for Python simulator)
- pip and virtualenv (for Python dependencies)

### 1. Start Infrastructure

```bash
# Start TimescaleDB and Grafana
docker compose up -d timescaledb grafana

# Wait for database to initialize (check logs)
docker compose logs -f timescaledb
# Look for: "database system is ready to accept connections"
```

### 2. Start Go Ingestion Service

```bash
# Start the Go service
docker compose up -d go-service

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
docker compose --profile testing up simulator

# Option B: Run directly (requires Python 3.10+)
cd python-simulator
pip install -r requirements.txt
python -m satellite_sim --satellites 100 --rate 100 --duration 60
# Or: PYTHONPATH=$(pwd) python satellite_sim.py --satellites 100 --rate 100 --duration 60
```

### 5. View Grafana Dashboard

1. Open http://localhost:3000
2. Login: admin / admin
3. Navigate to **Dashboards → OrbitStream** folder
4. Available dashboards:
   - **Downsampling Performance** - Compare raw vs aggregated query speeds

Or explore data manually:
- Navigate to Explore → Select TimescaleDB datasource
- Run queries like:

```sql
-- Real-time throughput (raw data)
SELECT
    time_bucket('1 second', time) AS bucket,
    COUNT(*) AS points_per_second
FROM telemetry
WHERE time > NOW() - INTERVAL '5 minutes'
GROUP BY bucket
ORDER BY bucket DESC;

-- Use continuous aggregates for historical queries (much faster!)
SELECT * FROM satellite_stats_hourly
WHERE bucket > NOW() - INTERVAL '24 hours'
ORDER BY bucket DESC;
```

## API Endpoints

| Endpoint | Method | Description | Request Body |
|----------|--------|-------------|--------------|
| `/health` | GET | Health check | - |
| `/telemetry` | POST | Send single telemetry point | `{"satellite_id": "...", "battery_charge_percent": 85.5, ...}` |
| `/telemetry/batch` | POST | Send batch of telemetry points | Array of telemetry points |

## Configuration

### Environment Variables (Go Service)

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| DATABASE_URL | `postgres://postgres:postgres@timescaledb:5432/orbitstream?sslmode=disable` | TimescaleDB connection |
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
- Check retention policies:
  - Raw telemetry: 7 days
  - Hourly aggregates: 6 months
  - Daily aggregates: 1 year
- Use continuous aggregates for historical queries instead of raw data

### Go Service Won't Start

1. Check if TimescaleDB is ready:
   ```bash
   docker compose logs timescaledb
   ```

2. Verify database connection:
   ```bash
   docker compose exec go-service ping timescaledb
   ```

## Testing

### Go Service Tests

The Go service has comprehensive unit tests for all major components:

```bash
cd go-service

# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific package tests
go test ./config -v
go test ./handlers -v
go test ./db -v
go test ./models -v
go test ./db/aggregates_test.go -v  # Continuous aggregate tests (requires Docker)
```

**Test Coverage:**
- `config/` - Configuration loading and environment variable parsing (100%)
- `handlers/` - HTTP request handling and responses (95%)
- `db/` - Anomaly detection logic (batch processing requires DB)
- `models/` - Data model validation
- `db/aggregates_test.go` - Continuous aggregate tests (schema, functionality, integration)
- `db/schema_test.go` - Database schema validation
- `db/integration_test.go` - Full pipeline integration tests

### Python Simulator Tests

The Python simulator has unit tests for the telemetry generator and simulator:

```bash
cd python-simulator

# Install test dependencies
pip install -r requirements.txt

# Run all tests (PYTHONPATH is required for imports)
PYTHONPATH=$(pwd) pytest -v

# Run with coverage
PYTHONPATH=$(pwd) pytest --cov=. --cov-report=html

# Run specific test file
PYTHONPATH=$(pwd) pytest tests/test_telemetry_gen.py -v
```

**Note:** The `PYTHONPATH` environment variable is required so Python can find the `generators` and `config` modules when running tests.

**Test Coverage:**
- `tests/test_telemetry_gen.py` - Data generation, anomaly injection, value validation
- `tests/test_satellite_sim.py` - Satellite initialization, statistics tracking
- `tests/test_config.py` - Configuration dataclass

### Test Files

| Go Tests | Description |
|----------|-------------|
| `config/config_test.go` | Environment variable parsing, defaults, validation |
| `handlers/telemetry_test.go` | HTTP endpoints, JSON binding, error handling |
| `db/batch_test.go` | Anomaly detection thresholds |
| `models/telemetry_test.go` | JSON serialization, model validation |

| Python Tests | Description |
|---------------|-------------|
| `tests/test_telemetry_gen.py` | Generator behavior, anomaly distribution, bounds checking |
| `tests/test_satellite_sim.py` | Swarm initialization, statistics, ID formatting |
| `tests/test_config.py` | Dataclass validation |

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
│   │   ├── telemetry.go        # Telemetry endpoints
│   │   └── telemetry_test.go   # Handler tests
│   ├── models/                 # Data structures
│   │   ├── telemetry.go        # Telemetry models
│   │   └── telemetry_test.go   # Model tests
│   ├── db/                     # Database layer
│   │   ├── connection.go       # Connection pool
│   │   ├── batch.go            # Batch processor
│   │   ├── batch_test.go       # Anomaly detection tests
│   │   ├── aggregates_test.go  # Continuous aggregate tests
│   │   ├── schema_test.go      # Schema validation tests
│   │   ├── integration_test.go # Pipeline integration tests
│   │   ├── test_helpers.go     # Test utilities
│   │   └── init.sql            # Schema (hypertables + continuous aggregates)
│   ├── config/                 # Configuration
│   │   ├── config.go           # Config loading
│   │   └── config_test.go      # Config tests
│   ├── test/                   # Test utilities
│   │   └── setup.go            # Mocks and helpers
│   └── Dockerfile
│
├── python-simulator/           # Satellite simulator
│   ├── satellite_sim.py        # Main script
│   ├── config.py               # Configuration
│   ├── generators/             # Telemetry data generator
│   │   └── telemetry_gen.py
│   ├── requirements.txt        # Python deps
│   ├── pytest.ini              # Pytest configuration
│   ├── tests/                  # Test suite
│   │   ├── conftest.py         # Shared fixtures
│   │   ├── test_telemetry_gen.py
│   │   ├── test_satellite_sim.py
│   │   └── test_config.py
│   └── Dockerfile
│
└── grafana/                    # Grafana configuration
    └── provisioning/
        ├── datasources/
        │   └── timescaledb.yml
        └── dashboards/
            ├── dashboards.yml              # Dashboard provisioning config
            └── downsampling-comparison.json # Performance comparison dashboard
```

## Database Architecture

### Continuous Aggregates (Hierarchical Downsampling)

OrbitStream uses TimescaleDB continuous aggregates to provide fast queries at any time scale:

| View | Bucket Size | Retention | Use Case |
|------|-------------|-----------|----------|
| `telemetry` (raw) | - | 7 days | Real-time monitoring, debugging |
| `satellite_stats` | 5 minutes | - | Recent trends |
| `satellite_stats_hourly` | 1 hour | 6 months | Weekly/monthly analysis |
| `satellite_stats_daily` | 1 day | 1 year | Long-term trends, capacity planning |

### Aggregate Metrics

Each aggregate view includes:
- **AVG/MIN/MAX** for battery, storage, and signal
- **data_points** - count of raw records
- **anomaly_count** - number of anomalies in the bucket

### Storage Optimization

- **Compression**: 90-95% space savings on all tables
- **Tiered Retention**: Automatic cleanup of old data
- **Expected Savings**: ~90%+ reduction in long-term storage costs

## License

MIT

## Author

Built as a high-throughput telemetry demonstration project.
