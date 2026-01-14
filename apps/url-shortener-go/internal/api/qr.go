package api

import (
	"context"
	"strconv"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/auth"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// QRHandler handles QR code generation endpoints
type QRHandler struct {
	cache     *cache.RedisCache
	db        *db.DB
	cfg       *config.Config
	qrService *services.QRService
}

// NewQRHandler creates a new QR handler
func NewQRHandler(cache *cache.RedisCache, database *db.DB, cfg *config.Config, qrService *services.QRService) *QRHandler {
	return &QRHandler{
		cache:     cache,
		db:        database,
		cfg:       cfg,
		qrService: qrService,
	}
}

// getUser retrieves or creates the user from Auth0 claims
func (h *QRHandler) getUser(c *fiber.Ctx) (*db.User, error) {
	auth0Sub := auth.GetAuth0Sub(c)
	email := auth.GetEmail(c)
	name := auth.GetName(c)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return h.db.GetOrCreateUser(ctx, auth0Sub, email, name)
}

// GetQRPNG handles GET /v1/urls/:id/qr - generates QR code as PNG
func (h *QRHandler) GetQRPNG(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

	// Get size from query param (default 256)
	size, _ := strconv.Atoi(c.Query("size", "256"))

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get URL to verify ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Verify ownership
	if url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Check cache first
	cached, err := h.cache.GetQRCode(ctx, id.String(), "png", size)
	if err == nil {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "public, max-age=86400")
		return c.Send(cached)
	}

	// Generate QR code
	shortURL := h.cfg.BaseURL + "/" + url.ShortCode
	qrData, err := h.qrService.GeneratePNG(shortURL, size)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to generate QR code",
		})
	}

	// Cache the QR code
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.SetQRCode(bgCtx, id.String(), "png", size, qrData)
	}()

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(qrData)
}

// GetQRSVG handles GET /v1/urls/:id/qr.svg - generates QR code as SVG
func (h *QRHandler) GetQRSVG(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

	// Get size from query param (default 256)
	size, _ := strconv.Atoi(c.Query("size", "256"))

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get URL to verify ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Verify ownership
	if url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Check cache first
	cached, err := h.cache.GetQRCode(ctx, id.String(), "svg", size)
	if err == nil {
		c.Set("Content-Type", "image/svg+xml")
		c.Set("Cache-Control", "public, max-age=86400")
		return c.Send(cached)
	}

	// Generate QR code
	shortURL := h.cfg.BaseURL + "/" + url.ShortCode
	qrData, err := h.qrService.GenerateSVG(shortURL, size)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to generate QR code",
		})
	}

	// Cache the QR code
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.SetQRCode(bgCtx, id.String(), "svg", size, []byte(qrData))
	}()

	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.SendString(qrData)
}
