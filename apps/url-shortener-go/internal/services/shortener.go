package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/cache"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/config"
	"github.com/Jainish-S/playground/apps/url-shortener-go/internal/db"
)

const (
	// Base62 alphabet for URL-safe short codes
	base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// ShortenerService handles short code generation and validation
type ShortenerService struct {
	cache *cache.RedisCache
	db    *db.DB
	cfg   *config.Config
}

// NewShortenerService creates a new shortener service
func NewShortenerService(cache *cache.RedisCache, database *db.DB, cfg *config.Config) *ShortenerService {
	return &ShortenerService{
		cache: cache,
		db:    database,
		cfg:   cfg,
	}
}

// GenerateCode generates a new random alphanumeric short code
func (s *ShortenerService) GenerateCode(ctx context.Context) (string, error) {
	maxRetries := 5
	
	for i := 0; i < maxRetries; i++ {
		// Generate random code
		code := generateRandomCode(s.cfg.ShortCodeMinLength)
		
		// Check for collision
		exists, err := s.codeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
		
		if !exists {
			return code, nil
		}
		
		// Collision detected, retry with a longer code
	}
	
	// If all retries failed, generate a longer code
	code := generateRandomCode(s.cfg.ShortCodeMinLength + 1)
	return code, nil
}

// ValidateCustomCode validates a user-provided custom short code
func (s *ShortenerService) ValidateCustomCode(ctx context.Context, code string) error {
	// Check length
	if len(code) < 4 || len(code) > 12 {
		return errors.New("custom code must be 4-12 characters")
	}

	// Alphanumeric and hyphens only
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, code)
	if !matched {
		return errors.New("custom code must contain only letters, numbers, and hyphens")
	}

	// Reserved words
	reserved := []string{
		"api", "v1", "admin", "health", "ready", "metrics", "docs",
		"dashboard", "login", "logout", "signup", "auth",
	}
	codeLower := strings.ToLower(code)
	for _, word := range reserved {
		if codeLower == word {
			return fmt.Errorf("custom code '%s' is reserved", code)
		}
	}

	// Check if already taken
	exists, err := s.codeExists(ctx, code)
	if err != nil {
		return fmt.Errorf("failed to check code availability: %w", err)
	}
	if exists {
		return errors.New("custom code already taken")
	}

	return nil
}

// codeExists checks if a short code already exists in the database
func (s *ShortenerService) codeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := s.db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM urls
			WHERE short_code = $1 AND is_active = true
		)
	`, code).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// encodeBase62 encodes an integer to Base62 string
func encodeBase62(num int64) string {
	if num == 0 {
		return "0"
	}

	encoded := ""
	base := int64(len(base62Alphabet))

	for num > 0 {
		remainder := num % base
		encoded = string(base62Alphabet[remainder]) + encoded
		num /= base
	}

	return encoded
}

// decodeBase62 decodes a Base62 string to integer
func decodeBase62(str string) int64 {
	decoded := int64(0)
	base := int64(len(base62Alphabet))

	for _, char := range str {
		decoded = decoded*base + int64(strings.IndexRune(base62Alphabet, char))
	}

	return decoded
}

// generateRandomCode generates a random alphanumeric code of specified length
func generateRandomCode(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a pseudo-random approach if crypto/rand fails
		// (should never happen in practice)
		panic("crypto/rand.Read failed: " + err.Error())
	}
	
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	
	return string(b)
}
