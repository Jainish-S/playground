package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ClickEvent represents a click event from Redis Stream
type ClickEvent struct {
	ShortCode string `json:"short_code"`
	IPHash    string `json:"ip_hash"`
	UserAgent string `json:"user_agent"`
	Referrer  string `json:"referrer"`
	Timestamp int64  `json:"timestamp"`
}

// Flusher processes click events from Redis Stream and writes to TimescaleDB
type Flusher struct {
	cache    *cache.RedisCache
	db       *db.DB
	cfg      *config.Config
	batchSize int
	flushInterval time.Duration
}

// NewFlusher creates a new analytics flusher
func NewFlusher(cache *cache.RedisCache, database *db.DB, cfg *config.Config) *Flusher {
	return &Flusher{
		cache:         cache,
		db:            database,
		cfg:           cfg,
		batchSize:     100,
		flushInterval: 5 * time.Second,
	}
}

// Start begins the flusher worker
func (f *Flusher) Start(ctx context.Context) {
	log.Println("Analytics Flusher started - consuming from analytics:stream")

	ticker := time.NewTicker(f.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Flusher shutting down...")
			// Final flush before exit
			f.flush(context.Background())
			return
		case <-ticker.C:
			f.flush(ctx)
		}
	}
}

// flush processes a batch of events from the Redis Stream
func (f *Flusher) flush(ctx context.Context) {
	// Read events from stream
	events, err := f.readEvents(ctx)
	if err != nil {
		log.Printf("Error reading events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	log.Printf("Processing %d click events...", len(events))

	// Process each event
	for _, event := range events {
		if err := f.processEvent(ctx, event); err != nil {
			log.Printf("Error processing event: %v", err)
			// Continue processing other events
		}
	}

	log.Printf("Processed %d click events", len(events))
}

// readEvents reads events from the Redis Stream
func (f *Flusher) readEvents(ctx context.Context) ([]ClickEvent, error) {
	// Use XREAD to get events
	result, err := f.cache.ReadStream(ctx, "analytics:stream", f.batchSize)
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	events := make([]ClickEvent, 0, len(result))
	for _, msg := range result {
		data, ok := msg["data"].(string)
		if !ok {
			continue
		}

		var event ClickEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Printf("Error parsing event: %v", err)
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// processEvent processes a single click event and writes to TimescaleDB
func (f *Flusher) processEvent(ctx context.Context, event ClickEvent) error {
	// Get URL ID from short code
	url, err := f.db.GetURLByShortCode(ctx, event.ShortCode)
	if err != nil {
		return err
	}

	// Parse device info from user agent (simplified)
	deviceType := parseDeviceType(event.UserAgent)
	browser := parseBrowser(event.UserAgent)
	os := parseOS(event.UserAgent)

	// Create click time from timestamp
	clickTime := time.Unix(event.Timestamp, 0)

	// Insert into clicks table
	return f.db.InsertClick(ctx, db.Click{
		Time:       clickTime,
		URLID:      url.ID,
		IPHash:     event.IPHash,
		UserAgent:  event.UserAgent,
		Referrer:   event.Referrer,
		DeviceType: deviceType,
		Browser:    browser,
		OS:         os,
	})
}

// parseDeviceType extracts device type from user agent
func parseDeviceType(userAgent string) string {
	ua := userAgent
	if len(ua) == 0 {
		return "unknown"
	}

	// Simple detection
	if contains(ua, "Mobile") || contains(ua, "Android") || contains(ua, "iPhone") {
		return "mobile"
	}
	if contains(ua, "Tablet") || contains(ua, "iPad") {
		return "tablet"
	}
	if contains(ua, "bot") || contains(ua, "Bot") || contains(ua, "crawler") {
		return "bot"
	}
	return "desktop"
}

// parseBrowser extracts browser from user agent
func parseBrowser(userAgent string) string {
	ua := userAgent
	if contains(ua, "Chrome") && !contains(ua, "Chromium") {
		return "Chrome"
	}
	if contains(ua, "Firefox") {
		return "Firefox"
	}
	if contains(ua, "Safari") && !contains(ua, "Chrome") {
		return "Safari"
	}
	if contains(ua, "Edge") {
		return "Edge"
	}
	return "Other"
}

// parseOS extracts OS from user agent
func parseOS(userAgent string) string {
	ua := userAgent
	if contains(ua, "Windows") {
		return "Windows"
	}
	if contains(ua, "Mac OS") {
		return "macOS"
	}
	if contains(ua, "Linux") {
		return "Linux"
	}
	if contains(ua, "Android") {
		return "Android"
	}
	if contains(ua, "iOS") || contains(ua, "iPhone") || contains(ua, "iPad") {
		return "iOS"
	}
	return "Other"
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// InsertClick adds to db package the ability to insert clicks
// This is defined in db/analytics.go
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
