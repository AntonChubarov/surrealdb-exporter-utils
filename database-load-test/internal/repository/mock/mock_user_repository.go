package mock

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/config"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/executables"
)

// mockUserRepository is an example in-memory implementation for testing
// Replace this with your actual database implementation
type mockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
	ids   []string
	cfg   config.DatabaseConfig
}

// NewUserRepository creates a new mock user repository
// This is just an example - implement with your actual database
func NewUserRepository(cfg config.DatabaseConfig) executables.UserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
		ids:   make([]string, 0),
		cfg:   cfg,
	}
}

func (r *mockUserRepository) Setup(ctx context.Context) error {
	// In a real implementation, this would create tables, indexes, etc.
	fmt.Println("Mock repository: Setup completed")
	return nil
}

func (r *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; exists {
		return fmt.Errorf("user with ID %s already exists", user.ID)
	}

	r.users[user.ID] = user
	r.ids = append(r.ids, user.ID)
	return nil
}

func (r *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user with ID %s not found", id)
	}

	// Return a copy to avoid race conditions
	userCopy := *user
	return &userCopy, nil
}

func (r *mockUserRepository) GetAll(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.User], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalCount := int64(len(r.users))
	totalPages := (int(totalCount) + params.PageSize - 1) / params.PageSize

	// Calculate pagination
	startIdx := (params.Page - 1) * params.PageSize
	endIdx := startIdx + params.PageSize

	if startIdx >= len(r.ids) {
		return &domain.PaginatedResult[domain.User]{
			Items:      []domain.User{},
			TotalCount: totalCount,
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalPages: totalPages,
		}, nil
	}

	if endIdx > len(r.ids) {
		endIdx = len(r.ids)
	}

	items := make([]domain.User, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		if user, exists := r.users[r.ids[i]]; exists {
			items = append(items, *user)
		}
	}

	return &domain.PaginatedResult[domain.User]{
		Items:      items,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return fmt.Errorf("user with ID %s not found", user.ID)
	}

	r.users[user.ID] = user
	return nil
}

func (r *mockUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[id]; !exists {
		return fmt.Errorf("user with ID %s not found", id)
	}

	delete(r.users, id)

	// Remove from IDs slice
	for i, uid := range r.ids {
		if uid == id {
			r.ids = append(r.ids[:i], r.ids[i+1:]...)
			break
		}
	}

	return nil
}

func (r *mockUserRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.users)), nil
}

func (r *mockUserRepository) GetRandomID(ctx context.Context) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ids) == 0 {
		return "", fmt.Errorf("no users available")
	}

	randomIdx := rand.Intn(len(r.ids))
	return r.ids[randomIdx], nil
}
