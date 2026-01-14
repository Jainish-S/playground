package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID       uuid.UUID `json:"id"`
	Auth0Sub string    `json:"auth0_sub"`
	Email    string    `json:"email"`
	Name     string    `json:"name,omitempty"`
}

// GetOrCreateUser gets an existing user by Auth0 sub or creates a new one
func (db *DB) GetOrCreateUser(ctx context.Context, auth0Sub, email, name string) (*User, error) {
	user := &User{}
	
	// Try to get existing user
	err := db.Pool.QueryRow(ctx, `
		SELECT id, auth0_sub, email, name
		FROM users
		WHERE auth0_sub = $1
	`, auth0Sub).Scan(&user.ID, &user.Auth0Sub, &user.Email, &user.Name)
	
	if err == nil {
		return user, nil
	}

	// Create new user
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO users (auth0_sub, email, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (auth0_sub) DO UPDATE SET
			email = EXCLUDED.email,
			name = COALESCE(EXCLUDED.name, users.name),
			updated_at = NOW()
		RETURNING id, auth0_sub, email, name
	`, auth0Sub, email, name).Scan(&user.ID, &user.Auth0Sub, &user.Email, &user.Name)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	return user, nil
}

// GetUserByAuth0Sub retrieves a user by their Auth0 subject
func (db *DB) GetUserByAuth0Sub(ctx context.Context, auth0Sub string) (*User, error) {
	user := &User{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, auth0_sub, email, name
		FROM users
		WHERE auth0_sub = $1
	`, auth0Sub).Scan(&user.ID, &user.Auth0Sub, &user.Email, &user.Name)
	
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by their internal ID
func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, auth0_sub, email, name
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Auth0Sub, &user.Email, &user.Name)
	
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}
