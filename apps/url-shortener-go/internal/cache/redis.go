package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisCache wraps the Redis client for caching operations
type RedisCache struct {
	client *redis.Client
	cfg    *config.Config
}

// New creates a new Redis cache client
func New(cfg *config.Config) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr(),
		DB:           cfg.RedisDB,
		Password:     "", // No password by default
		DialTimeout:  5 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     50,
		MinIdleConns: 10,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		cfg:    cfg,
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// HealthCheck performs a Redis health check
func (c *RedisCache) HealthCheck(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// GetURL retrieves a URL from cache by short code
func (c *RedisCache) GetURL(ctx context.Context, shortCode string) (string, error) {
	key := "url:" + shortCode
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("cache miss")
	}
	if err != nil {
		return "", fmt.Errorf("cache get error: %w", err)
	}
	return val, nil
}

// SetURL caches a URL with the configured TTL
func (c *RedisCache) SetURL(ctx context.Context, shortCode, destinationURL string) error {
	key := "url:" + shortCode
	err := c.client.Set(ctx, key, destinationURL, c.cfg.URLCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}
	return nil
}

// DeleteURL removes a URL from cache
func (c *RedisCache) DeleteURL(ctx context.Context, shortCode string) error {
	key := "url:" + shortCode
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("cache delete error: %w", err)
	}
	return nil
}

// DeleteQRCodes removes QR codes from cache for a URL
func (c *RedisCache) DeleteQRCodes(ctx context.Context, urlID string) error {
	// Delete all QR code variants for this URL
	pattern := "qr:" + urlID + ":*"
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("cache delete qr error: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("cache scan error: %w", err)
	}
	return nil
}

// GetQRCode retrieves a QR code from cache
func (c *RedisCache) GetQRCode(ctx context.Context, urlID, format string, size int) ([]byte, error) {
	key := fmt.Sprintf("qr:%s:%s:%d", urlID, format, size)
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache miss")
	}
	if err != nil {
		return nil, fmt.Errorf("cache get error: %w", err)
	}
	return val, nil
}

// SetQRCode caches a QR code
func (c *RedisCache) SetQRCode(ctx context.Context, urlID, format string, size int, data []byte) error {
	key := fmt.Sprintf("qr:%s:%s:%d", urlID, format, size)
	err := c.client.Set(ctx, key, data, c.cfg.QRCacheTTL).Err()
	if err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}
	return nil
}

// RecordClickEvent adds a click event to the Redis Stream for async processing
func (c *RedisCache) RecordClickEvent(ctx context.Context, event map[string]interface{}) error {
	err := c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: "analytics:stream",
		Values: event,
	}).Err()
	if err != nil {
		return fmt.Errorf("stream add error: %w", err)
	}
	return nil
}

// CheckRateLimit implements token bucket rate limiting
// Returns true if request is allowed, false if rate limited
func (c *RedisCache) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateLimitKey := "ratelimit:" + key

	// Increment counter
	count, err := c.client.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		return false, fmt.Errorf("rate limit error: %w", err)
	}

	// Set expiry on first request
	if count == 1 {
		c.client.Expire(ctx, rateLimitKey, window)
	}

	// Check if over limit
	return count <= int64(limit), nil
}

// IncrementCounter atomically increments a distributed counter
func (c *RedisCache) IncrementCounter(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("counter increment error: %w", err)
	}
	return val, nil
}

// Stats returns Redis statistics
func (c *RedisCache) Stats(ctx context.Context) (*redis.PoolStats, error) {
	stats := c.client.PoolStats()
	return stats, nil
}

// ReadStream reads events from a Redis Stream
func (c *RedisCache) ReadStream(ctx context.Context, stream string, count int) ([]map[string]interface{}, error) {
	// Read from stream with XREAD
	result, err := c.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{stream, "0"},
		Count:   int64(count),
		Block:   0, // Non-blocking
	}).Result()
	if err != nil {
		return nil, err
	}

	events := make([]map[string]interface{}, 0)
	for _, stream := range result {
		for _, msg := range stream.Messages {
			event := make(map[string]interface{})
			for k, v := range msg.Values {
				event[k] = v
			}
			event["_id"] = msg.ID
			events = append(events, event)

			// Acknowledge and delete the message
			c.client.XDel(ctx, stream.Stream, msg.ID)
		}
	}

	return events, nil
}
