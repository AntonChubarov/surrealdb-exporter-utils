package executables

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"github.com/brianvoe/gofakeit/v7"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Setup(ctx context.Context) error
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetAll(ctx context.Context, params domain.PaginationParams) (*domain.PaginatedResult[domain.User], error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

// UserExecutablesFactory creates different types of user executables
type UserExecutablesFactory struct {
	userRepo   UserRepository
	collector  Collector
	userBuffer map[string]struct{}
	mu         sync.RWMutex
}

// NewUserExecutablesFactory creates a new executables factory
func NewUserExecutablesFactory(userRepo UserRepository, collector Collector) *UserExecutablesFactory {
	return &UserExecutablesFactory{
		userRepo:   userRepo,
		collector:  collector,
		userBuffer: make(map[string]struct{}),
		mu:         sync.RWMutex{},
	}
}

// SetupUsersExecutable creates an executable for setting up the users collection
func (f *UserExecutablesFactory) SetupUsersExecutable() *setupUsersExecutable {
	return &setupUsersExecutable{
		factory: f,
	}
}

// UserCreateExecutable creates an executable for creating users
func (f *UserExecutablesFactory) UserCreateExecutable() *createUserExecutable {
	return &createUserExecutable{
		factory: f,
	}
}

// UserReadExecutable creates an executable for reading users
func (f *UserExecutablesFactory) UserReadExecutable() *readUserExecutable {
	return &readUserExecutable{
		factory: f,
	}
}

// UserUpdateExecutable creates an executable for updating users
func (f *UserExecutablesFactory) UserUpdateExecutable() *updateUserExecutable {
	return &updateUserExecutable{
		factory: f,
	}
}

// UserDeleteExecutable creates an executable for deleting users
func (f *UserExecutablesFactory) UserDeleteExecutable() *deleteUserExecutable {
	return &deleteUserExecutable{
		factory: f,
	}
}

// UserGetAllExecutable creates an executable for getting all users with pagination
func (f *UserExecutablesFactory) UserGetAllExecutable(pageSize int) *getAllUsersExecutable {
	return &getAllUsersExecutable{
		factory:  f,
		pageSize: pageSize,
	}
}

func (f *UserExecutablesFactory) getRandomUserID() string {
	f.mu.RLock()

	for id := range f.userBuffer {
		f.mu.RUnlock()
		return id
	}

	f.mu.RUnlock()
	return "empty"
}

func (f *UserExecutablesFactory) updateUserBuffer(users []domain.User) {
	userIDMap := make(map[string]struct{}, len(users))
	for i := range users {
		userIDMap[users[i].ID] = struct{}{}
	}

	f.mu.Lock()
	f.userBuffer = userIDMap
	f.mu.Unlock()
}

// setupUsersExecutable handles initial setup of the users collection
type setupUsersExecutable struct {
	factory *UserExecutablesFactory
}

func (e *setupUsersExecutable) Name() string {
	return "setup-users"
}

func (e *setupUsersExecutable) Execute(ctx context.Context) error {
	start := time.Now()
	err := e.factory.userRepo.Setup(ctx)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// createUserExecutable handles user creation operations
type createUserExecutable struct {
	factory *UserExecutablesFactory
}

func (e *createUserExecutable) Name() string {
	return "create-user"
}

func (e *createUserExecutable) Execute(ctx context.Context) error {
	user := generateRandomUser()
	start := time.Now()
	err := e.factory.userRepo.Create(ctx, user)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// readUserExecutable handles user read operations
type readUserExecutable struct {
	factory *UserExecutablesFactory
}

func (e *readUserExecutable) Name() string {
	return "read-user"
}

func (e *readUserExecutable) Execute(ctx context.Context) error {
	id := e.factory.getRandomUserID()

	start := time.Now()

	_, err := e.factory.userRepo.GetByID(ctx, id)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// updateUserExecutable handles user update operations
type updateUserExecutable struct {
	factory *UserExecutablesFactory
}

func (e *updateUserExecutable) Name() string {
	return "update-user"
}

func (e *updateUserExecutable) Execute(ctx context.Context) error {
	id := e.factory.getRandomUserID()

	start := time.Now()

	user, err := e.factory.userRepo.GetByID(ctx, id)
	e.factory.collector.RecordOperation("read-user", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	// Randomly pick one field to update
	switch gofakeit.IntRange(1, 3) {
	case 1:
		user.FirstName = gofakeit.FirstName()
	case 2:
		user.LastName = gofakeit.LastName()
	case 3:
		user.Age = gofakeit.IntRange(18, 98)
	}

	user.UpdatedAt = time.Now()

	start = time.Now()

	err = e.factory.userRepo.Update(ctx, user)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// deleteUserExecutable handles user deletion operations
type deleteUserExecutable struct {
	factory *UserExecutablesFactory
}

func (e *deleteUserExecutable) Name() string {
	return "delete-user"
}

func (e *deleteUserExecutable) Execute(ctx context.Context) error {
	id := e.factory.getRandomUserID()

	start := time.Now()

	err := e.factory.userRepo.Delete(ctx, id)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// getAllUsersExecutable handles paginated user retrieval
type getAllUsersExecutable struct {
	factory  *UserExecutablesFactory
	pageSize int
}

func (e *getAllUsersExecutable) Name() string {
	return "get-all-users"
}

func (e *getAllUsersExecutable) Execute(ctx context.Context) error {
	start := time.Now()

	count, err := e.factory.userRepo.Count(ctx)
	e.factory.collector.RecordOperation("users-count", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	pages := int(count) / e.pageSize
	if int(count)%e.pageSize > 0 {
		pages++
	}

	params := domain.PaginationParams{
		Page:     gofakeit.IntRange(1, pages),
		PageSize: e.pageSize,
	}

	start = time.Now()

	result, err := e.factory.userRepo.GetAll(ctx, params)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	if err != nil {
		return err
	}

	go e.factory.updateUserBuffer(result.Items)

	return nil
}

// generateRandomUser creates a random user for testing using gofakeit
func generateRandomUser() *domain.User {
	now := time.Now()

	firstName := gofakeit.FirstName()
	lastName := gofakeit.LastName()

	email := fmt.Sprintf("%s_%s@%s", strings.ToLower(firstName), strings.ToLower(lastName), gofakeit.DomainName())

	return &domain.User{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Age:       gofakeit.IntRange(18, 98),
		CreatedAt: now,
		UpdatedAt: now,
	}
}
