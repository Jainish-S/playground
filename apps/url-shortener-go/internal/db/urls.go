package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// URL represents a shortened URL
type URL struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	ShortCode      string                 `json:"short_code"`
	DestinationURL string                 `json:"destination_url"`
	Notes          *string                `json:"notes,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	IsActive       bool                   `json:"is_active"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// CreateURL inserts a new URL into the database
func (db *DB) CreateURL(ctx context.Context, userID uuid.UUID, shortCode, destinationURL string, expiresIn *int, notes *string) (*URL, error) {
	var expiresAt *time.Time
	if expiresIn != nil && *expiresIn > 0 {
		expiry := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		expiresAt = &expiry
	}

	url := &URL{}
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO urls (user_id, short_code, destination_url, notes, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, short_code, destination_url, notes, metadata, is_active, expires_at, created_at, updated_at
	`, userID, shortCode, destinationURL, notes, expiresAt).Scan(
		&url.ID, &url.UserID, &url.ShortCode, &url.DestinationURL,
		&url.Notes, &url.Metadata, &url.IsActive, &url.ExpiresAt,
		&url.CreatedAt, &url.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	return url, nil
}

// GetURLByShortCode retrieves an active URL by its short code
func (db *DB) GetURLByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	url := &URL{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, short_code, destination_url, notes, metadata, is_active, expires_at, created_at, updated_at
		FROM urls
		WHERE short_code = $1
			AND is_active = true
			AND (expires_at IS NULL OR expires_at > NOW())
	`, shortCode).Scan(
		&url.ID, &url.UserID, &url.ShortCode, &url.DestinationURL,
		&url.Notes, &url.Metadata, &url.IsActive, &url.ExpiresAt,
		&url.CreatedAt, &url.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("URL not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// GetURLByID retrieves a URL by its ID
func (db *DB) GetURLByID(ctx context.Context, id uuid.UUID) (*URL, error) {
	url := &URL{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, short_code, destination_url, notes, metadata, is_active, expires_at, created_at, updated_at
		FROM urls
		WHERE id = $1
	`, id).Scan(
		&url.ID, &url.UserID, &url.ShortCode, &url.DestinationURL,
		&url.Notes, &url.Metadata, &url.IsActive, &url.ExpiresAt,
		&url.CreatedAt, &url.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("URL not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	return url, nil
}

// URLFilters represents filter options for listing URLs
type URLFilters struct {
	IsActive     *bool
	CreatedAfter *time.Time
	CreatedBefore *time.Time
	SortOrder    string // "asc" or "desc"
}

// ListUserURLs retrieves all URLs for a user with pagination and filters
func (db *DB) ListUserURLs(ctx context.Context, userID uuid.UUID, limit, offset int, filters *URLFilters) ([]*URL, error) {
	query := `
		SELECT id, user_id, short_code, destination_url, notes, metadata, is_active, expires_at, created_at, updated_at
		FROM urls
		WHERE user_id = $1`
	
	args := []interface{}{userID}
	argIndex := 2

	// Build dynamic WHERE clause based on filters
	if filters != nil {
		if filters.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *filters.IsActive)
			argIndex++
		}
		if filters.CreatedAfter != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *filters.CreatedAfter)
			argIndex++
		}
		if filters.CreatedBefore != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *filters.CreatedBefore)
			argIndex++
		}
	}

	// Add ordering
	sortOrder := "DESC"
	if filters != nil && filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY created_at %s", sortOrder)

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list URLs: %w", err)
	}
	defer rows.Close()

	urls := []*URL{}
	for rows.Next() {
		url := &URL{}
		err := rows.Scan(
			&url.ID, &url.UserID, &url.ShortCode, &url.DestinationURL,
			&url.Notes, &url.Metadata, &url.IsActive, &url.ExpiresAt,
			&url.CreatedAt, &url.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan URL: %w", err)
		}
		urls = append(urls, url)
	}

	return urls, nil
}

// UpdateURL updates a URL's destination, notes, or expiry
func (db *DB) UpdateURL(ctx context.Context, id uuid.UUID, destinationURL *string, notes *string, expiresIn *int, isActive *bool) error {
	var expiresAt *time.Time
	if expiresIn != nil && *expiresIn > 0 {
		expiry := time.Now().Add(time.Duration(*expiresIn) * time.Second)
		expiresAt = &expiry
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE urls
		SET
			destination_url = COALESCE($2, destination_url),
			notes = COALESCE($3, notes),
			expires_at = COALESCE($4, expires_at),
			is_active = COALESCE($5, is_active),
			updated_at = NOW()
		WHERE id = $1
	`, id, destinationURL, notes, expiresAt, isActive)
	if err != nil {
		return fmt.Errorf("failed to update URL: %w", err)
	}

	return nil
}

// DeactivateURL soft deletes a URL by setting is_active to false
func (db *DB) DeactivateURL(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE urls
		SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate URL: %w", err)
	}

	return nil
}

// CountUserURLs counts total URLs for a user with filters
func (db *DB) CountUserURLs(ctx context.Context, userID uuid.UUID, filters *URLFilters) (int, error) {
	query := `SELECT COUNT(*) FROM urls WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	// Build dynamic WHERE clause based on filters
	if filters != nil {
		if filters.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *filters.IsActive)
			argIndex++
		}
		if filters.CreatedAfter != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
			args = append(args, *filters.CreatedAfter)
			argIndex++
		}
		if filters.CreatedBefore != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
			args = append(args, *filters.CreatedBefore)
			argIndex++
		}
	}

	var count int
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count URLs: %w", err)
	}
	return count, nil
}
