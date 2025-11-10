package surrealdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/surrealdb/surrealdb.go"
)

// Config provides SurrealDB-specific configuration
type Config interface {
	SurrealURL() string
	SurrealNamespace() string
	SurrealDatabase() string
	SurrealUsername() string
	SurrealPassword() string
}

// NewConnection creates and configures a new SurrealDB connection
func NewConnection(ctx context.Context, cfg Config) (*surrealdb.DB, error) {
	// Create new SurrealDB connection
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.SurrealURL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SurrealDB: %w", err)
	}

	// Sign in with credentials
	authData := &surrealdb.Auth{
		Username: cfg.SurrealUsername(),
		Password: cfg.SurrealPassword(),
	}

	token, err := db.SignIn(ctx, authData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in to SurrealDB: %w", err)
	}

	// Authenticate using the token
	if err = db.Authenticate(ctx, token); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Select namespace and database
	if err = db.Use(ctx, cfg.SurrealNamespace(), cfg.SurrealDatabase()); err != nil {
		return nil, fmt.Errorf("failed to use namespace/database: %w", err)
	}

	sessionInfo, err := surrealdb.Query[[]interface{}](ctx, db, "SELECT * FROM $session", nil)
	if err != nil {
		slog.Error("failed to query session info", "error", err)
	}
	slog.Info("APP A: Session info", "session", sessionInfo)

	dbInfo, _ := surrealdb.Query[map[string]any](ctx, db, "INFO FOR DB", nil)
	slog.Info("APP A: Database info", "db", dbInfo)

	return db, nil
}
