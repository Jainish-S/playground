package api

import (
	"context"
	"strconv"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/auth"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// AnalyticsHandler handles analytics query endpoints
type AnalyticsHandler struct {
	db  *db.DB
	cfg *config.Config
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(database *db.DB, cfg *config.Config) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:  database,
		cfg: cfg,
	}
}

// getUser retrieves or creates the user from Auth0 claims
func (h *AnalyticsHandler) getUser(c *fiber.Ctx) (*db.User, error) {
	auth0Sub := auth.GetAuth0Sub(c)
	email := auth.GetEmail(c)
	name := auth.GetName(c)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return h.db.GetOrCreateUser(ctx, auth0Sub, email, name)
}

// GetAnalytics handles GET /v1/urls/:id/analytics - gets comprehensive analytics
func (h *AnalyticsHandler) GetAnalytics(c *fiber.Ctx) error {
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

	// Verify URL ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil || url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Get stats
	stats, err := h.db.GetURLStats(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get analytics",
		})
	}

	// Get device breakdown
	devices, err := h.db.GetDeviceBreakdown(ctx, id)
	if err != nil {
		devices = []db.DeviceBreakdown{} // Fallback to empty
	}

	// Get browser breakdown
	browsers, err := h.db.GetBrowserBreakdown(ctx, id)
	if err != nil {
		browsers = []db.BrowserBreakdown{} // Fallback to empty
	}

	return c.JSON(fiber.Map{
		"url_id":          id.String(),
		"short_code":      url.ShortCode,
		"total_clicks":    stats.TotalClicks,
		"unique_visitors": stats.UniqueVisitors,
		"mobile_clicks":   stats.MobileClicks,
		"desktop_clicks":  stats.DesktopClicks,
		"tablet_clicks":   stats.TabletClicks,
		"devices":         devices,
		"browsers":        browsers,
	})
}

// GetClicksOverTime handles GET /v1/urls/:id/analytics/clicks
func (h *AnalyticsHandler) GetClicksOverTime(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid URL ID",
		})
	}

	// Get days parameter (default 7)
	days, _ := strconv.Atoi(c.Query("days", "7"))
	if days < 1 {
		days = 1
	}
	if days > 90 {
		days = 90
	}

	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Verify URL ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil || url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Get time series data
	points, err := h.db.GetClicksOverTime(ctx, id, days)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get click data",
		})
	}

	return c.JSON(fiber.Map{
		"url_id": id.String(),
		"days":   days,
		"data":   points,
	})
}

// GetGeoBreakdown handles GET /v1/urls/:id/analytics/geo
func (h *AnalyticsHandler) GetGeoBreakdown(c *fiber.Ctx) error {
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

	// Verify URL ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil || url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Get geo breakdown
	breakdown, err := h.db.GetGeoBreakdown(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get geo data",
		})
	}

	return c.JSON(fiber.Map{
		"url_id": id.String(),
		"data":   breakdown,
	})
}

// GetDeviceBreakdown handles GET /v1/urls/:id/analytics/devices
func (h *AnalyticsHandler) GetDeviceBreakdown(c *fiber.Ctx) error {
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

	// Verify URL ownership
	url, err := h.db.GetURLByID(ctx, id)
	if err != nil || url.UserID != user.ID {
		return c.Status(404).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	// Get device breakdown
	devices, err := h.db.GetDeviceBreakdown(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get device data",
		})
	}

	// Get browser breakdown
	browsers, err := h.db.GetBrowserBreakdown(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get browser data",
		})
	}

	return c.JSON(fiber.Map{
		"url_id":   id.String(),
		"devices":  devices,
		"browsers": browsers,
	})
}

// GetDashboard handles GET /v1/dashboard - user dashboard stats
func (h *AnalyticsHandler) GetDashboard(c *fiber.Ctx) error {
	// Get user
	user, err := h.getUser(c)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get user",
		})
	}

	ctx := c.Context()

	// Get dashboard stats
	stats, err := h.db.GetUserDashboardStats(ctx, user.ID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to get dashboard stats",
		})
	}

	return c.JSON(stats)
}
