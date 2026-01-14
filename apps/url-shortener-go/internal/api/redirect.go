package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/gofiber/fiber/v2"
)

// RedirectHandler handles the hot path redirect endpoint
type RedirectHandler struct {
	cache *cache.RedisCache
	db    *db.DB
	cfg   *config.Config
}

// NewRedirectHandler creates a new redirect handler
func NewRedirectHandler(cache *cache.RedisCache, database *db.DB, cfg *config.Config) *RedirectHandler {
	return &RedirectHandler{
		cache: cache,
		db:    database,
		cfg:   cfg,
	}
}

// HandleRedirect handles GET /{code} - the critical hot path
// Target latency: P95 < 50ms, P99 < 100ms
func (h *RedirectHandler) HandleRedirect(c *fiber.Ctx) error {
	shortCode := c.Params("code")
	if shortCode == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "short code required",
		})
	}

	ctx := c.Context()

	// Extract metadata synchronously to avoid race conditions with Fiber context
	userAgent := c.Get("User-Agent")
	referrer := c.Get("Referer")
	ip := c.IP()

	// FAST PATH: Try cache first (~1ms)
	destinationURL, err := h.cache.GetURL(ctx, shortCode)
	if err == nil {
		// Cache hit - fast path success!
		// Record click event asynchronously (non-blocking)
		go h.recordClickEvent(shortCode, userAgent, referrer, ip)

		// Redirect immediately
		return c.Redirect(destinationURL, 302)
	}

	// SLOW PATH: Cache miss, fallback to database (~15ms)
	url, err := h.db.GetURLByShortCode(ctx, shortCode)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found or expired",
		})
	}

	// Write-through cache for future requests
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.SetURL(bgCtx, shortCode, url.DestinationURL)
	}()

	// Record click event asynchronously (non-blocking)
	go h.recordClickEvent(shortCode, userAgent, referrer, ip)

	// Redirect
	return c.Redirect(url.DestinationURL, 302)
}

// recordClickEvent records a click event to Redis Stream for async processing
// This function runs in a goroutine and should not block the redirect
func (h *RedirectHandler) recordClickEvent(shortCode, userAgent, referrer, ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Hash IP for privacy (GDPR-compliant)
	ipHash := hashIP(ip, h.cfg.IPHashSalt)

	// Create click event
	event := map[string]interface{}{
		"short_code": shortCode,
		"ip_hash":    ipHash,
		"user_agent": userAgent,
		"referrer":   referrer,
		"timestamp":  time.Now().Unix(),
	}

	// Convert to JSON for storage
	eventJSON, err := json.Marshal(event)
	if err != nil {
		// Log error but don't fail the redirect
		return
	}

	// Send to Redis Stream (will be processed by worker)
	eventData := map[string]interface{}{
		"data": string(eventJSON),
	}

	// Best-effort send (don't block if Redis is slow)
	h.cache.RecordClickEvent(ctx, eventData)
}

// hashIP hashes an IP address with a salt for privacy
func hashIP(ip, salt string) string {
	h := sha256.New()
	h.Write([]byte(ip + salt))
	return hex.EncodeToString(h.Sum(nil))
}
