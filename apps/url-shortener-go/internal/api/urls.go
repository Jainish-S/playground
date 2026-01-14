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

// URLHandler handles URL CRUD operations
type URLHandler struct {
	cache     *cache.RedisCache
	db        *db.DB
	cfg       *config.Config
	shortener *services.ShortenerService
	validator *services.ValidatorService
}

// NewURLHandler creates a new URL handler
func NewURLHandler(cache *cache.RedisCache, database *db.DB, cfg *config.Config, shortener *services.ShortenerService, validator *services.ValidatorService) *URLHandler {
	return &URLHandler{
		cache:     cache,
		db:        database,
		cfg:       cfg,
		shortener: shortener,
		validator: validator,
	}
}

// CreateURLRequest represents the request body for creating a URL
type CreateURLRequest struct {
	DestinationURL string `json:"destination_url"`
	CustomCode     string `json:"custom_code,omitempty"`
	Notes          string `json:"notes,omitempty"`
	ExpiresIn      *int   `json:"expires_in,omitempty"` // seconds
}

// URLResponse represents a URL in API responses
type URLResponse struct {
	ID             string     `json:"id"`
	ShortCode      string     `json:"short_code"`
	ShortURL       string     `json:"short_url"`
	DestinationURL string     `json:"destination_url"`
	Notes          *string    `json:"notes,omitempty"`
	IsActive       bool       `json:"is_active"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// toResponse converts db.URL to URLResponse
func (h *URLHandler) toResponse(url *db.URL) *URLResponse {
	return &URLResponse{
		ID:             url.ID.String(),
		ShortCode:      url.ShortCode,
		ShortURL:       h.cfg.BaseURL + "/" + url.ShortCode,
		DestinationURL: url.DestinationURL,
		Notes:          url.Notes,
		IsActive:       url.IsActive,
		ExpiresAt:      url.ExpiresAt,
		CreatedAt:      url.CreatedAt,
		UpdatedAt:      url.UpdatedAt,
	}
}

// getUser retrieves or creates the user from Auth0 claims
func (h *URLHandler) getUser(c *fiber.Ctx) (*db.User, error) {
	auth0Sub := auth.GetAuth0Sub(c)
	email := auth.GetEmail(c)
	name := auth.GetName(c)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return h.db.GetOrCreateUser(ctx, auth0Sub, email, name)
}

// CreateURL handles POST /v1/urls - creates a new short URL
func (h *URLHandler) CreateURL(c *fiber.Ctx) error {
	var req CreateURLRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate destination URL
	sanitizedURL, err := h.validator.ValidateAndSanitizeURL(req.DestinationURL)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	ctx := c.Context()

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	// Generate or validate short code
	var shortCode string
	if req.CustomCode != "" {
		// Validate custom code
		if err := h.shortener.ValidateCustomCode(ctx, req.CustomCode); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		shortCode = req.CustomCode
	} else {
		// Generate short code
		shortCode, err = h.shortener.GenerateCode(ctx)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to generate short code",
			})
		}
	}

	// Create URL in database
	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	url, err := h.db.CreateURL(ctx, user.ID, shortCode, sanitizedURL, req.ExpiresIn, notes)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to create URL",
		})
	}

	// Pre-cache the URL for fast redirects
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.SetURL(bgCtx, shortCode, sanitizedURL)
	}()

	return c.Status(201).JSON(h.toResponse(url))
}

// ListURLs handles GET /v1/urls - lists user's URLs with pagination and filters
func (h *URLHandler) ListURLs(c *fiber.Ctx) error {
	// Get pagination params
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}

	// Parse filter parameters
	filters := &db.URLFilters{}
	
	// is_active filter
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filters.IsActive = &isActive
	}

	// sort_order filter
	sortOrder := c.Query("sort_order", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	filters.SortOrder = sortOrder

	// created_after filter
	if createdAfterStr := c.Query("created_after"); createdAfterStr != "" {
		createdAfter, err := time.Parse(time.RFC3339, createdAfterStr)
		if err == nil {
			filters.CreatedAfter = &createdAfter
		}
	}

	// created_before filter
	if createdBeforeStr := c.Query("created_before"); createdBeforeStr != "" {
		createdBefore, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err == nil {
			filters.CreatedBefore = &createdBefore
		}
	}

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get URLs with filters
	urls, err := h.db.ListUserURLs(ctx, user.ID, limit, offset, filters)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to list URLs",
		})
	}

	// Get total count with filters
	total, err := h.db.CountUserURLs(ctx, user.ID, filters)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to count URLs",
		})
	}

	// Convert to response
	response := make([]*URLResponse, len(urls))
	for i, url := range urls {
		response[i] = h.toResponse(url)
	}

	return c.JSON(fiber.Map{
		"urls":   response,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetURL handles GET /v1/urls/:id - gets a specific URL
func (h *URLHandler) GetURL(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get URL
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

	return c.JSON(h.toResponse(url))
}

// UpdateURLRequest represents the request body for updating a URL
type UpdateURLRequest struct {
	DestinationURL *string `json:"destination_url,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	ExpiresIn      *int    `json:"expires_in,omitempty"` // seconds
	IsActive       *bool   `json:"is_active,omitempty"`
}

// UpdateURL handles PATCH /v1/urls/:id - updates a URL
func (h *URLHandler) UpdateURL(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

	var req UpdateURLRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get existing URL to verify ownership
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

	// Validate new destination URL if provided
	var sanitizedURL *string
	if req.DestinationURL != nil {
		validated, err := h.validator.ValidateAndSanitizeURL(*req.DestinationURL)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		sanitizedURL = &validated
	}

	// Update URL
	if err := h.db.UpdateURL(ctx, id, sanitizedURL, req.Notes, req.ExpiresIn, req.IsActive); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to update URL",
		})
	}

	// Invalidate cache
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.DeleteURL(bgCtx, url.ShortCode)
	}()

	// Get updated URL
	updatedURL, err := h.db.GetURLByID(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get updated URL",
		})
	}

	return c.JSON(h.toResponse(updatedURL))
}

// DeleteURL handles DELETE /v1/urls/:id - soft deletes a URL
func (h *URLHandler) DeleteURL(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

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

	// Deactivate URL
	if err := h.db.DeactivateURL(ctx, id); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to delete URL",
		})
	}

	// Invalidate cache
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		h.cache.DeleteURL(bgCtx, url.ShortCode)
		h.cache.DeleteQRCodes(bgCtx, url.ID.String())
	}()

	return c.SendStatus(204)
}

// CheckCode handles GET /v1/urls/check/:code - checks if a custom code is available
func (h *URLHandler) CheckCode(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "code is required",
		})
	}

	ctx := c.Context()

	err := h.shortener.ValidateCustomCode(ctx, code)
	if err != nil {
		return c.JSON(fiber.Map{
			"available": false,
			"reason":    err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"available": true,
	})
}
