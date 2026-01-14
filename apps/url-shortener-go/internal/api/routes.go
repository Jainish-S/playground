package api

import (
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/auth"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/services"
	"github.com/gofiber/fiber/v2"
)

// Handlers holds all API handlers
type Handlers struct {
	Redirect  *RedirectHandler
	URL       *URLHandler
	QR        *QRHandler
	Analytics *AnalyticsHandler
}

// NewHandlers creates all handlers with dependencies
func NewHandlers(
	redisCache *cache.RedisCache,
	database *db.DB,
	cfg *config.Config,
) *Handlers {
	// Create services
	shortener := services.NewShortenerService(redisCache, database, cfg)
	validator := services.NewValidatorService()
	qrService := services.NewQRService()

	return &Handlers{
		Redirect:  NewRedirectHandler(redisCache, database, cfg),
		URL:       NewURLHandler(redisCache, database, cfg, shortener, validator),
		QR:        NewQRHandler(redisCache, database, cfg, qrService),
		Analytics: NewAnalyticsHandler(database, cfg),
	}
}

// RegisterRoutes registers all API routes
func RegisterRoutes(app *fiber.App, handlers *Handlers, cfg *config.Config) {
	// Public routes (no auth)
	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "URL Shortener API",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// API v1 routes (protected with Auth0)
	v1 := app.Group("/v1")
	
	// Apply Auth0 middleware only if Auth0 is configured
	if cfg.Auth0Domain != "" && cfg.Auth0Audience != "" {
		v1.Use(auth.Middleware(cfg))
	}

	// URL management endpoints
	v1.Post("/urls", handlers.URL.CreateURL)
	v1.Get("/urls", handlers.URL.ListURLs)
	v1.Get("/urls/check/:code", handlers.URL.CheckCode)
	v1.Get("/urls/:id", handlers.URL.GetURL)
	v1.Patch("/urls/:id", handlers.URL.UpdateURL)
	v1.Delete("/urls/:id", handlers.URL.DeleteURL)

	// QR code endpoints
	v1.Get("/urls/:id/qr", handlers.QR.GetQRPNG)
	v1.Get("/urls/:id/qr.svg", handlers.QR.GetQRSVG)

	// Analytics endpoints
	v1.Get("/urls/:id/analytics", handlers.Analytics.GetAnalytics)
	v1.Get("/urls/:id/analytics/clicks", handlers.Analytics.GetClicksOverTime)
	v1.Get("/urls/:id/analytics/geo", handlers.Analytics.GetGeoBreakdown)
	v1.Get("/urls/:id/analytics/devices", handlers.Analytics.GetDeviceBreakdown)
	v1.Get("/dashboard", handlers.Analytics.GetDashboard)

	// HOT PATH: Redirect handler - must be registered LAST
	// This catches all unmatched GET /{code} requests
	app.Get("/:code", handlers.Redirect.HandleRedirect)
}

