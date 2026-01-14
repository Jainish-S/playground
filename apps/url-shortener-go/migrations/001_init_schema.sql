-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table (Auth0-backed)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_sub VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_auth0_sub ON users(auth0_sub);

-- URLs table
CREATE TABLE IF NOT EXISTS urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    short_code VARCHAR(12) UNIQUE NOT NULL,
    destination_url TEXT NOT NULL,
    notes TEXT,
    metadata JSONB,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;

-- URL history table (audit trail)
CREATE TABLE IF NOT EXISTS url_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url_id UUID NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    previous_url TEXT NOT NULL,
    new_url TEXT NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    changed_by UUID REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_url_history_url_id ON url_history(url_id, changed_at DESC);

-- Clicks table (time-series data)
CREATE TABLE IF NOT EXISTS clicks (
    time TIMESTAMPTZ NOT NULL,
    url_id UUID NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    ip_hash VARCHAR(64),
    user_agent TEXT,
    referrer TEXT,
    country VARCHAR(2),
    city VARCHAR(100),
    latitude FLOAT,
    longitude FLOAT,
    device_type VARCHAR(20),
    browser VARCHAR(50),
    os VARCHAR(50),
    PRIMARY KEY (time, url_id, ip_hash)
);

-- Convert clicks table to hypertable (7-day chunks)
SELECT create_hypertable('clicks', 'time', chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);

-- Enable compression on clicks table (90%+ storage savings)
ALTER TABLE clicks SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'url_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- Add compression policy (compress chunks older than 7 days)
SELECT add_compression_policy('clicks', INTERVAL '7 days', if_not_exists => TRUE);

-- Continuous aggregates for fast analytics

-- Hourly stats
CREATE MATERIALIZED VIEW IF NOT EXISTS hourly_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', time) AS bucket,
    url_id,
    COUNT(*) AS click_count,
    COUNT(DISTINCT ip_hash) AS unique_visitors,
    COUNT(*) FILTER (WHERE device_type = 'mobile') AS mobile_clicks,
    COUNT(*) FILTER (WHERE device_type = 'desktop') AS desktop_clicks,
    COUNT(*) FILTER (WHERE device_type = 'tablet') AS tablet_clicks
FROM clicks
GROUP BY bucket, url_id
WITH NO DATA;

-- Add refresh policy for hourly stats (refresh every hour)
SELECT add_continuous_aggregate_policy('hourly_stats',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- Daily stats
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS bucket,
    url_id,
    COUNT(*) AS click_count,
    COUNT(DISTINCT ip_hash) AS unique_visitors
FROM clicks
GROUP BY bucket, url_id
WITH NO DATA;

-- Add refresh policy for daily stats (refresh every day)
SELECT add_continuous_aggregate_policy('daily_stats',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Refresh the materialized views to populate with existing data (if any)
CALL refresh_continuous_aggregate('hourly_stats', NULL, NULL);
CALL refresh_continuous_aggregate('daily_stats', NULL, NULL);
