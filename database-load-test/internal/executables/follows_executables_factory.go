package executables

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"github.com/brianvoe/gofakeit/v7"
)

// FollowsRepository defines the interface for follow data operations
type FollowsRepository interface {
	Setup(ctx context.Context) error
	Create(ctx context.Context, follow *domain.Follow) error
	GetByID(ctx context.Context, id string) (*domain.Follow, error)
	GetUserFollows(ctx context.Context, userID string, params domain.PaginationParams) (*domain.PaginatedResult[domain.Follow], error)
	GetUserFollowers(ctx context.Context, userID string, params domain.PaginationParams) (*domain.PaginatedResult[domain.Follow], error)
	GetCommonFollows(ctx context.Context, userAID, userBID string) ([]string, error)
	GetCommonFollowers(ctx context.Context, userAID, userBID string) ([]string, error)
	Delete(ctx context.Context, id string) error
	DeleteByUsers(ctx context.Context, followerID, followeeID string) error
	Count(ctx context.Context) (int64, error)
	GetTwoRandomUserIDs(ctx context.Context) (string, string, error)
	GetRandomFollowID(ctx context.Context) (string, error)
}

// FollowsExecutablesFactory creates different types of follows executables
type FollowsExecutablesFactory struct {
	followsRepo  FollowsRepository
	collector    Collector
	followBuffer map[string]struct{}
	userBuffer   []string
	mu           sync.RWMutex
}

// NewFollowsExecutablesFactory creates a new follows executables factory
func NewFollowsExecutablesFactory(followsRepo FollowsRepository, collector Collector) *FollowsExecutablesFactory {
	return &FollowsExecutablesFactory{
		followsRepo:  followsRepo,
		collector:    collector,
		followBuffer: make(map[string]struct{}),
		userBuffer:   make([]string, 0),
		mu:           sync.RWMutex{},
	}
}

// SetupFollowsExecutable creates an executable for setting up the follows collection
func (f *FollowsExecutablesFactory) SetupFollowsExecutable() *setupFollowsExecutable {
	return &setupFollowsExecutable{
		factory: f,
	}
}

// FollowCreateExecutable creates an executable for creating follow relationships
func (f *FollowsExecutablesFactory) FollowCreateExecutable() *createFollowExecutable {
	return &createFollowExecutable{
		factory: f,
	}
}

// GetUserFollowsExecutable creates an executable for getting users a user follows
func (f *FollowsExecutablesFactory) GetUserFollowsExecutable(pageSize int) *getUserFollowsExecutable {
	return &getUserFollowsExecutable{
		factory:  f,
		pageSize: pageSize,
	}
}

// GetUserFollowersExecutable creates an executable for getting a user's followers
func (f *FollowsExecutablesFactory) GetUserFollowersExecutable(pageSize int) *getUserFollowersExecutable {
	return &getUserFollowersExecutable{
		factory:  f,
		pageSize: pageSize,
	}
}

// CommonFollowsExecutable creates an executable for finding common follows between users
func (f *FollowsExecutablesFactory) CommonFollowsExecutable() *commonFollowsExecutable {
	return &commonFollowsExecutable{
		factory: f,
	}
}

// CommonFollowersExecutable creates an executable for finding common followers between users
func (f *FollowsExecutablesFactory) CommonFollowersExecutable() *commonFollowersExecutable {
	return &commonFollowersExecutable{
		factory: f,
	}
}

// DeleteFollowExecutable creates an executable for deleting follow relationships
func (f *FollowsExecutablesFactory) DeleteFollowExecutable() *deleteFollowExecutable {
	return &deleteFollowExecutable{
		factory: f,
	}
}

func (f *FollowsExecutablesFactory) getRandomUserID() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(f.userBuffer) == 0 {
		return ""
	}

	return f.userBuffer[rand.Intn(len(f.userBuffer))]
}

func (f *FollowsExecutablesFactory) getTwoRandomUserIDs() (string, string) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(f.userBuffer) < 2 {
		return "", ""
	}

	idx1 := rand.Intn(len(f.userBuffer))
	idx2 := rand.Intn(len(f.userBuffer))

	// Ensure different users
	for idx2 == idx1 {
		idx2 = rand.Intn(len(f.userBuffer))
	}

	return f.userBuffer[idx1], f.userBuffer[idx2]
}

func (f *FollowsExecutablesFactory) updateUserBuffer(userIDs []string) {
	f.mu.Lock()
	f.userBuffer = userIDs
	f.mu.Unlock()
}

// setupFollowsExecutable handles initial setup of the follows collection
type setupFollowsExecutable struct {
	factory *FollowsExecutablesFactory
}

func (e *setupFollowsExecutable) Name() string {
	return "setup-follows"
}

func (e *setupFollowsExecutable) Execute(ctx context.Context) error {
	start := time.Now()
	err := e.factory.followsRepo.Setup(ctx)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// createFollowExecutable handles follow relationship creation
type createFollowExecutable struct {
	factory *FollowsExecutablesFactory
}

func (e *createFollowExecutable) Name() string {
	return "create-follow"
}

func (e *createFollowExecutable) Execute(ctx context.Context) error {
	// Get two random user IDs from the repository
	start := time.Now()

	followerID, followeeID, err := e.factory.followsRepo.GetTwoRandomUserIDs(ctx)
	e.factory.collector.RecordOperation("get-random-users", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	// Update user buffer for other operations
	go e.factory.updateUserBuffer([]string{followerID, followeeID})

	// Generate random follow with relationship type
	follow := generateRandomFollow(followerID, followeeID)

	start = time.Now()
	err = e.factory.followsRepo.Create(ctx, follow)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// getUserFollowsExecutable handles getting users that a user follows
type getUserFollowsExecutable struct {
	factory  *FollowsExecutablesFactory
	pageSize int
}

func (e *getUserFollowsExecutable) Name() string {
	return "get-user-follows"
}

func (e *getUserFollowsExecutable) Execute(ctx context.Context) error {
	// Get a random user ID
	start := time.Now()

	followerID, _, err := e.factory.followsRepo.GetTwoRandomUserIDs(ctx)
	e.factory.collector.RecordOperation("get-random-user", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	params := domain.PaginationParams{
		Page:     1,
		PageSize: e.pageSize,
	}

	start = time.Now()
	_, err = e.factory.followsRepo.GetUserFollows(ctx, followerID, params)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// getUserFollowersExecutable handles getting a user's followers
type getUserFollowersExecutable struct {
	factory  *FollowsExecutablesFactory
	pageSize int
}

func (e *getUserFollowersExecutable) Name() string {
	return "get-user-followers"
}

func (e *getUserFollowersExecutable) Execute(ctx context.Context) error {
	// Get a random user ID
	start := time.Now()

	_, followeeID, err := e.factory.followsRepo.GetTwoRandomUserIDs(ctx)
	e.factory.collector.RecordOperation("get-random-user", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	params := domain.PaginationParams{
		Page:     1,
		PageSize: e.pageSize,
	}

	start = time.Now()
	_, err = e.factory.followsRepo.GetUserFollowers(ctx, followeeID, params)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// commonFollowsExecutable handles finding common follows between two users
type commonFollowsExecutable struct {
	factory *FollowsExecutablesFactory
}

func (e *commonFollowsExecutable) Name() string {
	return "common-follows"
}

func (e *commonFollowsExecutable) Execute(ctx context.Context) error {
	// Get two random user IDs
	start := time.Now()

	userAID, userBID, err := e.factory.followsRepo.GetTwoRandomUserIDs(ctx)
	e.factory.collector.RecordOperation("get-random-users", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	start = time.Now()
	_, err = e.factory.followsRepo.GetCommonFollows(ctx, userAID, userBID)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// commonFollowersExecutable handles finding common followers between two users
type commonFollowersExecutable struct {
	factory *FollowsExecutablesFactory
}

func (e *commonFollowersExecutable) Name() string {
	return "common-followers"
}

func (e *commonFollowersExecutable) Execute(ctx context.Context) error {
	// Get two random user IDs
	start := time.Now()

	userAID, userBID, err := e.factory.followsRepo.GetTwoRandomUserIDs(ctx)
	e.factory.collector.RecordOperation("get-random-users", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	start = time.Now()
	_, err = e.factory.followsRepo.GetCommonFollowers(ctx, userAID, userBID)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// deleteFollowExecutable handles deletion of follow relationships
type deleteFollowExecutable struct {
	factory *FollowsExecutablesFactory
}

func (e *deleteFollowExecutable) Name() string {
	return "delete-follow"
}

func (e *deleteFollowExecutable) Execute(ctx context.Context) error {
	// Get a random follow ID
	start := time.Now()

	followID, err := e.factory.followsRepo.GetRandomFollowID(ctx)
	e.factory.collector.RecordOperation("get-random-follow", time.Since(start), err == nil)
	if err != nil {
		return err
	}

	start = time.Now()
	err = e.factory.followsRepo.Delete(ctx, followID)
	e.factory.collector.RecordOperation(e.Name(), time.Since(start), err == nil)
	return err
}

// generateRandomFollow creates a random follow relationship with metadata
func generateRandomFollow(followerID, followeeID string) *domain.Follow {
	now := time.Now()

	// Get random relationship type
	relationshipTypes := domain.AllRelationshipTypes()
	randomType := relationshipTypes[gofakeit.IntRange(0, len(relationshipTypes)-1)]

	return &domain.Follow{
		FollowerID:       followerID,
		FolloweeID:       followeeID,
		RelationshipType: randomType,
		Strength:         gofakeit.IntRange(1, 10),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
