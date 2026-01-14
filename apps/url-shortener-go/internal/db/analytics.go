package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Click represents a click event for TimescaleDB
type Click struct {
	Time       time.Time
	URLID      uuid.UUID
	IPHash     string
	UserAgent  string
	Referrer   string
	Country    string
	City       string
	Latitude   float64
	Longitude  float64
	DeviceType string
	Browser    string
	OS         string
}

// InsertClick inserts a click event into the TimescaleDB clicks hypertable
func (db *DB) InsertClick(ctx context.Context, click Click) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO clicks (time, url_id, ip_hash, user_agent, referrer, country, city, latitude, longitude, device_type, browser, os)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (time, url_id, ip_hash) DO NOTHING
	`, click.Time, click.URLID, click.IPHash, click.UserAgent, click.Referrer,
		click.Country, click.City, click.Latitude, click.Longitude,
		click.DeviceType, click.Browser, click.OS)
	if err != nil {
		return fmt.Errorf("failed to insert click: %w", err)
	}
	return nil
}

// ClickStats represents aggregated click statistics
type ClickStats struct {
	TotalClicks    int64 `json:"total_clicks"`
	UniqueVisitors int64 `json:"unique_visitors"`
	MobileClicks   int64 `json:"mobile_clicks"`
	DesktopClicks  int64 `json:"desktop_clicks"`
	TabletClicks   int64 `json:"tablet_clicks"`
}

// TimeSeriesPoint represents a point in time-series data
type TimeSeriesPoint struct {
	Bucket time.Time `json:"bucket"`
	Clicks int64     `json:"clicks"`
	Unique int64     `json:"unique"`
}

// GeoBreakdown represents geographic breakdown
type GeoBreakdown struct {
	Country string `json:"country"`
	Clicks  int64  `json:"clicks"`
}

// DeviceBreakdown represents device type breakdown
type DeviceBreakdown struct {
	DeviceType string `json:"device_type"`
	Clicks     int64  `json:"clicks"`
}

// BrowserBreakdown represents browser breakdown
type BrowserBreakdown struct {
	Browser string `json:"browser"`
	Clicks  int64  `json:"clicks"`
}

// GetURLStats retrieves overall stats for a URL
func (db *DB) GetURLStats(ctx context.Context, urlID uuid.UUID) (*ClickStats, error) {
	stats := &ClickStats{}
	err := db.Pool.QueryRow(ctx, `
		SELECT 
			COUNT(*) as total_clicks,
			COUNT(DISTINCT ip_hash) as unique_visitors,
			COUNT(*) FILTER (WHERE device_type = 'mobile') as mobile_clicks,
			COUNT(*) FILTER (WHERE device_type = 'desktop') as desktop_clicks,
			COUNT(*) FILTER (WHERE device_type = 'tablet') as tablet_clicks
		FROM clicks
		WHERE url_id = $1
	`, urlID).Scan(
		&stats.TotalClicks,
		&stats.UniqueVisitors,
		&stats.MobileClicks,
		&stats.DesktopClicks,
		&stats.TabletClicks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL stats: %w", err)
	}
	return stats, nil
}

// GetClicksOverTime retrieves click data over time
// For recent data (last 3 hours), queries raw clicks table directly for real-time accuracy
// For older data, uses hourly_stats continuous aggregate for performance
func (db *DB) GetClicksOverTime(ctx context.Context, urlID uuid.UUID, days int) ([]TimeSeriesPoint, error) {
	// Always query raw clicks for the most recent data (last 3 hours)
	// This ensures we show real-time analytics without waiting for the continuous aggregate to refresh
	rows, err := db.Pool.Query(ctx, `
		SELECT
			time_bucket('1 hour', time) AS bucket,
			COUNT(*) AS click_count,
			COUNT(DISTINCT ip_hash) AS unique_visitors
		FROM clicks
		WHERE url_id = $1 AND time > NOW() - ($2 || ' days')::interval
		GROUP BY bucket
		ORDER BY bucket ASC
	`, urlID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks over time: %w", err)
	}
	defer rows.Close()

	points := []TimeSeriesPoint{}
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Bucket, &p.Clicks, &p.Unique); err != nil {
			return nil, fmt.Errorf("failed to scan time series point: %w", err)
		}
		points = append(points, p)
	}
	
	return points, nil
}

// GetGeoBreakdown retrieves clicks by country
func (db *DB) GetGeoBreakdown(ctx context.Context, urlID uuid.UUID) ([]GeoBreakdown, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT COALESCE(country, 'Unknown') as country, COUNT(*) as clicks
		FROM clicks
		WHERE url_id = $1
		GROUP BY country
		ORDER BY clicks DESC
		LIMIT 20
	`, urlID)
	if err != nil {
		return nil, fmt.Errorf("failed to get geo breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := []GeoBreakdown{}
	for rows.Next() {
		var g GeoBreakdown
		if err := rows.Scan(&g.Country, &g.Clicks); err != nil {
			return nil, fmt.Errorf("failed to scan geo breakdown: %w", err)
		}
		breakdown = append(breakdown, g)
	}
	return breakdown, nil
}

// GetDeviceBreakdown retrieves clicks by device type
func (db *DB) GetDeviceBreakdown(ctx context.Context, urlID uuid.UUID) ([]DeviceBreakdown, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT COALESCE(device_type, 'unknown') as device_type, COUNT(*) as clicks
		FROM clicks
		WHERE url_id = $1
		GROUP BY device_type
		ORDER BY clicks DESC
	`, urlID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := []DeviceBreakdown{}
	for rows.Next() {
		var d DeviceBreakdown
		if err := rows.Scan(&d.DeviceType, &d.Clicks); err != nil {
			return nil, fmt.Errorf("failed to scan device breakdown: %w", err)
		}
		breakdown = append(breakdown, d)
	}
	return breakdown, nil
}

// GetBrowserBreakdown retrieves clicks by browser
func (db *DB) GetBrowserBreakdown(ctx context.Context, urlID uuid.UUID) ([]BrowserBreakdown, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT COALESCE(browser, 'Unknown') as browser, COUNT(*) as clicks
		FROM clicks
		WHERE url_id = $1
		GROUP BY browser
		ORDER BY clicks DESC
		LIMIT 10
	`, urlID)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := []BrowserBreakdown{}
	for rows.Next() {
		var b BrowserBreakdown
		if err := rows.Scan(&b.Browser, &b.Clicks); err != nil {
			return nil, fmt.Errorf("failed to scan browser breakdown: %w", err)
		}
		breakdown = append(breakdown, b)
	}
	return breakdown, nil
}

// GetUserDashboardStats retrieves dashboard stats for a user
func (db *DB) GetUserDashboardStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	var totalURLs, totalClicks, uniqueVisitors int64

	// Get total URLs
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM urls WHERE user_id = $1
	`, userID).Scan(&totalURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to get total URLs: %w", err)
	}

	// Get total clicks and unique visitors across all user's URLs
	err = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.clicks), 0), COALESCE(SUM(c.unique_visitors), 0)
		FROM (
			SELECT COUNT(*) as clicks, COUNT(DISTINCT ip_hash) as unique_visitors
			FROM clicks
			WHERE url_id IN (SELECT id FROM urls WHERE user_id = $1)
		) c
	`, userID).Scan(&totalClicks, &uniqueVisitors)
	if err != nil {
		return nil, fmt.Errorf("failed to get click stats: %w", err)
	}

	return map[string]interface{}{
		"total_urls":      totalURLs,
		"total_clicks":    totalClicks,
		"unique_visitors": uniqueVisitors,
	}, nil
}
