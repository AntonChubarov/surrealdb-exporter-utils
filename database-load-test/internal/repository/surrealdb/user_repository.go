package surrealdb

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

const usersTable = "users"

type ManagedConnection interface {
	// DB returns the underlying SurrealDB connection
	DB() *surrealdb.DB

	// Close releases the connection back to its source.
	// For pooled connections: returns to pool
	// For singleton connections: no-op
	Close() error
}

// userRepository implements the UserRepository interface for SurrealDB
type userRepository struct {
	db *surrealdb.DB
}

// surrealUser is the SurrealDB representation of a Users with RecordID
type surrealUser struct {
	ID        *models.RecordID `json:"id,omitempty"`
	Email     string           `json:"email"`
	FirstName string           `json:"first_name"`
	LastName  string           `json:"last_name"`
	Age       int              `json:"age"`
	CreatedAt time.Time        `json:"created_at"` // SurrealDB datetime as string
	UpdatedAt time.Time        `json:"updated_at"` // SurrealDB datetime as string
}

// NewUserRepository creates a new SurrealDB user repository
func NewUserRepository(db *surrealdb.DB) *userRepository {
	return &userRepository{
		db: db,
	}
}

// Setup initializes the users table with schema definition
func (r *userRepository) Setup(ctx context.Context) error {
	// Define the users table with schema
	query := `
		DEFINE TABLE IF NOT EXISTS users SCHEMAFULL PERMISSIONS FULL;
		DEFINE FIELD IF NOT EXISTS email ON users TYPE string;
		DEFINE FIELD IF NOT EXISTS first_name ON users TYPE string;
		DEFINE FIELD IF NOT EXISTS last_name ON users TYPE string;
		DEFINE FIELD IF NOT EXISTS age ON users TYPE int;
		DEFINE FIELD IF NOT EXISTS created_at ON users TYPE datetime;
		DEFINE FIELD IF NOT EXISTS updated_at ON users TYPE datetime;
		DEFINE INDEX IF NOT EXISTS email_idx ON users FIELDS email UNIQUE;
	`

	_, err := surrealdb.Query[[]any](ctx, r.db, query, nil)
	if err != nil {
		return fmt.Errorf("failed to setup users table: %w", err)
	}

	return nil
}

// Create creates a new user in SurrealDB
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	// Convert domain user to SurrealDB user
	sUser := domainToSurreal(user)

	// Create the user with explicit ID
	_, err := surrealdb.Create[surrealUser](ctx, r.db, models.Table(usersTable), sUser)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	recordID := models.NewRecordID(usersTable, id)

	result, err := surrealdb.Select[surrealUser](ctx, r.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	// Check if user was found
	if result == nil {
		return nil, fmt.Errorf("user with ID %s not found", id)
	}

	return surrealToDomain(result), nil
}

// GetAll retrieves all users with pagination
func (r *userRepository) GetAll(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.User], error) {
	// Calculate offset
	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	// Query for paginated users
	query := `
		SELECT * FROM type::table($table)
		ORDER BY created_at DESC
		LIMIT $limit
		START $offset
	`

	results, err := surrealdb.Query[[]surrealUser](ctx, r.db, query, map[string]any{
		"table":  usersTable,
		"limit":  limit,
		"offset": offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}

	// Get total count
	totalCount, err := r.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user count: %w", err)
	}

	// Convert to domain users
	var domainUsers []domain.User
	if results != nil && len(*results) > 0 {
		surrealUsers := (*results)[0].Result
		domainUsers = make([]domain.User, 0, len(surrealUsers))
		for i := range surrealUsers {
			domainUsers = append(domainUsers, *surrealToDomain(&surrealUsers[i]))
		}
	} else {
		domainUsers = []domain.User{}
	}

	// Calculate total pages
	totalPages := int(totalCount) / params.PageSize
	if int(totalCount)%params.PageSize != 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.User]{
		Items:      domainUsers,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	recordID := models.NewRecordID(usersTable, user.ID)
	sUser := domainToSurreal(user)

	// Use Merge to update only provided fields
	_, err := surrealdb.Merge[surrealUser](ctx, r.db, recordID, sUser)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete deletes a user by their ID
func (r *userRepository) Delete(ctx context.Context, id string) error {
	recordID := models.NewRecordID(usersTable, id)

	_, err := surrealdb.Delete[surrealUser](ctx, r.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// Count returns the total number of users
func (r *userRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT count() FROM type::table($table) GROUP ALL`

	type countResult struct {
		Count int64 `json:"count"`
	}

	results, err := surrealdb.Query[[]countResult](ctx, r.db, query, map[string]any{
		"table": usersTable,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	if len(*results) == 0 || len((*results)[0].Result) == 0 {
		return 0, nil
	}

	return (*results)[0].Result[0].Count, nil
}

// GetRandomID returns a random user ID for load testing
func (r *userRepository) GetRandomID(ctx context.Context) (string, error) {
	// Get all user IDs
	query := `SELECT id FROM type::table($table)`

	type idResult struct {
		ID *models.RecordID `json:"id"`
	}

	results, err := surrealdb.Query[[]idResult](ctx, r.db, query, map[string]any{
		"table": usersTable,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get user IDs: %w", err)
	}

	if len(*results) == 0 || len((*results)[0].Result) == 0 {
		return "", fmt.Errorf("no users available")
	}

	// Get all IDs from the result
	allIDs := (*results)[0].Result

	// Pick a random user
	randomIdx := rand.Intn(len(allIDs))
	recordID := allIDs[randomIdx].ID

	// Extract the ID part from the RecordID
	if recordID != nil {
		return fmt.Sprintf("%v", recordID.ID), nil
	}

	return "", fmt.Errorf("invalid record ID")
}

// Helper functions to convert between domain and SurrealDB models

func domainToSurreal(user *domain.User) surrealUser {
	return surrealUser{
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Age:       user.Age,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func surrealToDomain(sUser *surrealUser) *domain.User {
	var id string
	if sUser.ID != nil {
		id = fmt.Sprintf("%v", sUser.ID.ID)
	}

	return &domain.User{
		ID:        id,
		Email:     sUser.Email,
		FirstName: sUser.FirstName,
		LastName:  sUser.LastName,
		Age:       sUser.Age,
		CreatedAt: sUser.CreatedAt,
		UpdatedAt: sUser.UpdatedAt,
	}
}
