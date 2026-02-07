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

-- Add retention policy (keep raw data for 7 days only)
-- Hourly aggregates cover 6 months, daily aggregates cover 1 year
SELECT add_retention_policy('telemetry',
    INTERVAL '7 days'
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

-- =====================================================
-- HOURLY CONTINUOUS AGGREGATE (for 1-7 day queries)
-- =====================================================
CREATE MATERIALIZED VIEW satellite_stats_hourly
WITH (timescaledb.continuous) AS
SELECT
    satellite_id,
    time_bucket('1 hour', time) AS bucket,
    AVG(battery_charge_percent) AS avg_battery,
    MIN(battery_charge_percent) AS min_battery,
    MAX(battery_charge_percent) AS max_battery,
    AVG(storage_usage_mb) AS avg_storage,
    MIN(storage_usage_mb) AS min_storage,
    MAX(storage_usage_mb) AS max_storage,
    AVG(signal_strength_dbm) AS avg_signal,
    MIN(signal_strength_dbm) AS min_signal,
    MAX(signal_strength_dbm) AS max_signal,
    COUNT(*) AS data_points,
    SUM(CASE WHEN is_anomaly THEN 1 ELSE 0 END) AS anomaly_count
FROM telemetry
GROUP BY satellite_id, bucket;

CREATE INDEX idx_satellite_stats_hourly_lookup
ON satellite_stats_hourly (satellite_id, bucket DESC);

-- Refresh policy: every hour, covering last 48 hours with 1-hour lag
SELECT add_continuous_aggregate_policy('satellite_stats_hourly',
    start_offset => INTERVAL '48 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour'
);

-- Enable compression on hourly aggregate (90%+ space savings)
ALTER MATERIALIZED VIEW satellite_stats_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'satellite_id',
    timescaledb.compress_orderby = 'bucket'
);

SELECT add_compression_policy('satellite_stats_hourly',
    INTERVAL '3 days'
);

-- Retention: keep hourly data for 6 months
SELECT add_retention_policy('satellite_stats_hourly',
    INTERVAL '6 months'
);

-- =====================================================
-- DAILY CONTINUOUS AGGREGATE (for 7-30 day queries)
-- =====================================================
CREATE MATERIALIZED VIEW satellite_stats_daily
WITH (timescaledb.continuous) AS
SELECT
    satellite_id,
    time_bucket('1 day', time) AS bucket,
    AVG(battery_charge_percent) AS avg_battery,
    MIN(battery_charge_percent) AS min_battery,
    MAX(battery_charge_percent) AS max_battery,
    AVG(storage_usage_mb) AS avg_storage,
    MIN(storage_usage_mb) AS min_storage,
    MAX(storage_usage_mb) AS max_storage,
    AVG(signal_strength_dbm) AS avg_signal,
    MIN(signal_strength_dbm) AS min_signal,
    MAX(signal_strength_dbm) AS max_signal,
    COUNT(*) AS data_points,
    SUM(CASE WHEN is_anomaly THEN 1 ELSE 0 END) AS anomaly_count
FROM telemetry
GROUP BY satellite_id, bucket;

CREATE INDEX idx_satellite_stats_daily_lookup
ON satellite_stats_daily (satellite_id, bucket DESC);

-- Refresh policy: daily, covering last 7 days with 1-day lag
SELECT add_continuous_aggregate_policy('satellite_stats_daily',
    start_offset => INTERVAL '7 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day'
);

-- Enable compression on daily aggregate (95%+ space savings)
ALTER MATERIALIZED VIEW satellite_stats_daily SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'satellite_id',
    timescaledb.compress_orderby = 'bucket'
);

SELECT add_compression_policy('satellite_stats_daily',
    INTERVAL '1 day'
);

-- Retention: keep daily data for 1 year
SELECT add_retention_policy('satellite_stats_daily',
    INTERVAL '1 year'
);
