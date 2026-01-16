-- OrbitStream TimescaleDB Schema
-- High-throughput satellite telemetry storage

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Main telemetry table
CREATE TABLE IF NOT EXISTS telemetry (
    time TIMESTAMPTZ NOT NULL,
    satellite_id VARCHAR(50) NOT NULL,
    battery_charge_percent DECIMAL(5,2) NOT NULL,
    storage_usage_mb DECIMAL(10,2) NOT NULL,
    signal_strength_dbm DECIMAL(6,2) NOT NULL,
    received_at TIMESTAMPTZ DEFAULT NOW(),
    is_anomaly BOOLEAN DEFAULT FALSE
);

-- Convert to hypertable with 1-hour chunks for optimal performance
SELECT create_hypertable('telemetry', 'time',
    chunk_time_interval => INTERVAL '1 hour'
);

-- Create indexes for efficient querying
CREATE INDEX idx_telemetry_satellite_time ON telemetry (satellite_id, time DESC);
CREATE INDEX idx_telemetry_anomaly ON telemetry (is_anomaly, time DESC) WHERE is_anomaly = TRUE;

-- Configure compression settings (90% space savings)
ALTER TABLE telemetry SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'satellite_id',
    timescaledb.compress_orderby = 'time'
);

-- Add compression policy (compress data older than 1 day)
SELECT add_compression_policy('telemetry',
    INTERVAL '1 day'
);

-- Add retention policy (keep data for 30 days)
SELECT add_retention_policy('telemetry',
    INTERVAL '30 days'
);

-- Create continuous aggregate for real-time stats (TimescaleDB feature)
-- This is automatically refreshed by TimescaleDB
CREATE MATERIALIZED VIEW satellite_stats
WITH (timescaledb.continuous) AS
SELECT
    satellite_id,
    time_bucket('5 minutes', time) AS bucket,
    AVG(battery_charge_percent) AS avg_battery,
    AVG(storage_usage_mb) AS avg_storage,
    AVG(signal_strength_dbm) AS avg_signal,
    COUNT(*) AS data_points
FROM telemetry
GROUP BY satellite_id, bucket;

-- Create index on continuous aggregate (not unique, as continuous aggregates don't support it)
CREATE INDEX idx_satellite_stats_lookup
ON satellite_stats (satellite_id, bucket);

-- Set refresh policy for the continuous aggregate
-- Note: bucket size is 5 minutes, so we need at least 10 minutes of coverage
SELECT add_continuous_aggregate_policy('satellite_stats',
    start_offset => INTERVAL '30 minutes',
    end_offset => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes'
);
