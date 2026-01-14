# Database Migrations

This directory contains SQL migration files for the URL Shortener database schema.

## Prerequisites

- PostgreSQL 14+ with TimescaleDB extension installed
- Database created (`urlshortener` by default)

## Setup TimescaleDB

### Using Docker

```bash
docker run -d --name timescaledb \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=urlshortener \
  timescale/timescaledb:latest-pg16
```

### Manual Installation

Follow the [TimescaleDB installation guide](https://docs.timescale.com/self-hosted/latest/install/) for your platform.

## Applying Migrations

### Method 1: Using psql

```bash
psql -h localhost -U postgres -d urlshortener -f migrations/001_init_schema.sql
```

### Method 2: Using Docker

```bash
docker exec -i timescaledb psql -U postgres -d urlshortener < migrations/001_init_schema.sql
```

## Verification

Check that TimescaleDB is installed:

```sql
SELECT extversion FROM pg_extension WHERE extname = 'timescaledb';
```

Check hypertables:

```sql
SELECT * FROM timescaledb_information.hypertables;
```

Check continuous aggregates:

```sql
SELECT * FROM timescaledb_information.continuous_aggregates;
```

## Schema Overview

### Regular Tables

- **users**: Auth0-backed user accounts
- **urls**: Shortened URLs with metadata
- **url_history**: Audit trail of URL changes

### TimescaleDB Hypertables

- **clicks**: Time-series click events (partitioned by time)

### Continuous Aggregates

- **hourly_stats**: Pre-computed hourly click statistics
- **daily_stats**: Pre-computed daily click statistics

## Compression

Clicks data is automatically compressed after 7 days, providing 90%+ storage savings while maintaining query performance.
