package auth

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gofiber/fiber/v2"
)

// CustomClaims contains custom claims from Auth0 token
type CustomClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Validate validates the custom claims (required by validator.CustomClaims interface)
func (c *CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// Middleware creates an Auth0 JWT validation middleware for Fiber
func Middleware(cfg *config.Config) fiber.Handler {
	issuerURL, err := url.Parse("https://" + cfg.Auth0Domain + "/")
	if err != nil {
		log.Fatalf("Failed to parse Auth0 issuer URL: %v", err)
	}

	// Setup JWKS provider with caching
	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	// Create JWT validator
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{cfg.Auth0Audience},
		validator.WithCustomClaims(func() validator.CustomClaims {
			return &CustomClaims{}
		}),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create JWT validator: %v", err)
	}

	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// Remove "Bearer " prefix
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}
		token := authHeader[7:]

		// Validate token
		claims, err := jwtValidator.ValidateToken(c.Context(), token)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		// Extract validated claims
		validatedClaims, ok := claims.(*validator.ValidatedClaims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "invalid claims format",
			})
		}

		// Store claims in context for use by handlers
		c.Locals("auth0_sub", validatedClaims.RegisteredClaims.Subject)
		
		if customClaims, ok := validatedClaims.CustomClaims.(*CustomClaims); ok {
			c.Locals("email", customClaims.Email)
			c.Locals("name", customClaims.Name)
		}

		return c.Next()
	}
}

// GetAuth0Sub extracts the Auth0 subject from the context
func GetAuth0Sub(c *fiber.Ctx) string {
	if sub, ok := c.Locals("auth0_sub").(string); ok {
		return sub
	}
	return ""
}

// GetEmail extracts the email from the context
func GetEmail(c *fiber.Ctx) string {
	if email, ok := c.Locals("email").(string); ok {
		return email
	}
	return ""
}

// GetName extracts the name from the context
func GetName(c *fiber.Ctx) string {
	if name, ok := c.Locals("name").(string); ok {
		return name
	}
	return ""
}
