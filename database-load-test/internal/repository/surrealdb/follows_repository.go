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

const followsTable = "follows"

// followsRepository implements the FollowsRepository interface for SurrealDB
type followsRepository struct {
	db *surrealdb.DB
}

// surrealFollow is the SurrealDB representation of a Follow edge with RecordID
type surrealFollow struct {
	ID               *models.RecordID `json:"id,omitempty"`
	In               *models.RecordID `json:"in,omitempty"`  // Follower (user who follows)
	Out              *models.RecordID `json:"out,omitempty"` // Followee (user being followed)
	RelationshipType string           `json:"relationship_type"`
	Strength         int              `json:"strength"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// surrealFollowWithUsers is used for queries that expand user data
type surrealFollowWithUsers struct {
	ID               *models.RecordID `json:"id,omitempty"`
	In               *surrealUser     `json:"in,omitempty"`
	Out              *surrealUser     `json:"out,omitempty"`
	RelationshipType string           `json:"relationship_type"`
	Strength         int              `json:"strength"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// NewFollowsRepository creates a new SurrealDB follows repository
func NewFollowsRepository(db *surrealdb.DB) *followsRepository {
	return &followsRepository{
		db: db,
	}
}

// Setup initializes the follows edge table with schema definition
func (r *followsRepository) Setup(ctx context.Context) error {
	// Define the follows table as a relation (edge table)
	query := `
		DEFINE TABLE IF NOT EXISTS follows TYPE RELATION IN users OUT users SCHEMAFULL PERMISSIONS FULL;
		DEFINE FIELD IF NOT EXISTS relationship_type ON follows TYPE string;
		DEFINE FIELD IF NOT EXISTS strength ON follows TYPE int;
		DEFINE FIELD IF NOT EXISTS created_at ON follows TYPE datetime;
		DEFINE FIELD IF NOT EXISTS updated_at ON follows TYPE datetime;
		DEFINE INDEX IF NOT EXISTS follows_in_out_idx ON follows FIELDS in, out UNIQUE;
	`

	_, err := surrealdb.Query[[]any](ctx, r.db, query, nil)
	if err != nil {
		return fmt.Errorf("failed to setup follows table: %w", err)
	}

	return nil
}

// Create creates a new follow relationship between two users using RELATE
func (r *followsRepository) Create(ctx context.Context, follow *domain.Follow) error {
	// Use RELATE statement to create graph edge
	query := `
		RELATE type::thing("users", $follower_id)->follows->type::thing("users", $followee_id) SET
			relationship_type = $relationship_type,
			strength = $strength,
			created_at = $created_at,
			updated_at = $updated_at
	`

	resp, err := surrealdb.Query[[]surrealFollow](ctx, r.db, query, map[string]any{
		"follower_id":       follow.FollowerID,
		"followee_id":       follow.FolloweeID,
		"relationship_type": string(follow.RelationshipType),
		"strength":          follow.Strength,
		"created_at":        follow.CreatedAt,
		"updated_at":        follow.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to create follow relationship: %w", err)
	}

	_ = resp

	return nil
}

// GetByID retrieves a follow relationship by its ID
func (r *followsRepository) GetByID(ctx context.Context, id string) (*domain.Follow, error) {
	recordID := models.NewRecordID(followsTable, id)

	result, err := surrealdb.Select[surrealFollow](ctx, r.db, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get follow by ID: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("follow with ID %s not found", id)
	}

	return surrealFollowToDomain(result), nil
}

// GetUserFollows retrieves all users that a specific user is following (outgoing edges)
func (r *followsRepository) GetUserFollows(ctx context.Context, userID string, params domain.PaginationParams) (*domain.PaginatedResult[domain.Follow], error) {
	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	// Query for users this user follows using graph traversal
	query := `
		SELECT * FROM follows 
		WHERE in = type::thing("users", $user_id)
		ORDER BY created_at DESC
		LIMIT $limit
		START $offset
	`

	results, err := surrealdb.Query[[]surrealFollow](ctx, r.db, query, map[string]any{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user follows: %w", err)
	}

	// Get total count
	countQuery := `SELECT count() FROM follows WHERE in = type::thing("users", $user_id) GROUP ALL`
	type countResult struct {
		Count int64 `json:"count"`
	}
	countResults, err := surrealdb.Query[[]countResult](ctx, r.db, countQuery, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count user follows: %w", err)
	}

	var totalCount int64
	if countResults != nil && len(*countResults) > 0 && len((*countResults)[0].Result) > 0 {
		totalCount = (*countResults)[0].Result[0].Count
	}

	// Convert to domain follows
	var domainFollows []domain.Follow
	if results != nil && len(*results) > 0 {
		surrealFollows := (*results)[0].Result
		domainFollows = make([]domain.Follow, 0, len(surrealFollows))
		for i := range surrealFollows {
			domainFollows = append(domainFollows, *surrealFollowToDomain(&surrealFollows[i]))
		}
	} else {
		domainFollows = []domain.Follow{}
	}

	totalPages := int(totalCount) / params.PageSize
	if int(totalCount)%params.PageSize != 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Follow]{
		Items:      domainFollows,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserFollowers retrieves all users that follow a specific user (incoming edges)
func (r *followsRepository) GetUserFollowers(ctx context.Context, userID string, params domain.PaginationParams) (*domain.PaginatedResult[domain.Follow], error) {
	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize

	// Query for users who follow this user using graph traversal
	query := `
		SELECT * FROM follows 
		WHERE out = type::thing("users", $user_id)
		ORDER BY created_at DESC
		LIMIT $limit
		START $offset
	`

	results, err := surrealdb.Query[[]surrealFollow](ctx, r.db, query, map[string]any{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user followers: %w", err)
	}

	// Get total count
	countQuery := `SELECT count() FROM follows WHERE out = type::thing("users", $user_id) GROUP ALL`
	type countResult struct {
		Count int64 `json:"count"`
	}
	countResults, err := surrealdb.Query[[]countResult](ctx, r.db, countQuery, map[string]any{
		"user_id": userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count user followers: %w", err)
	}

	var totalCount int64
	if countResults != nil && len(*countResults) > 0 && len((*countResults)[0].Result) > 0 {
		totalCount = (*countResults)[0].Result[0].Count
	}

	// Convert to domain follows
	var domainFollows []domain.Follow
	if results != nil && len(*results) > 0 {
		surrealFollows := (*results)[0].Result
		domainFollows = make([]domain.Follow, 0, len(surrealFollows))
		for i := range surrealFollows {
			domainFollows = append(domainFollows, *surrealFollowToDomain(&surrealFollows[i]))
		}
	} else {
		domainFollows = []domain.Follow{}
	}

	totalPages := int(totalCount) / params.PageSize
	if int(totalCount)%params.PageSize != 0 {
		totalPages++
	}

	return &domain.PaginatedResult[domain.Follow]{
		Items:      domainFollows,
		TotalCount: totalCount,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetCommonFollows finds users that both userA and userB follow
func (r *followsRepository) GetCommonFollows(ctx context.Context, userAID, userBID string) ([]string, error) {
	// Use graph traversal to find common follows
	query := `
		LET $follows_a = (SELECT VALUE out.id FROM follows WHERE in = type::thing("users", $user_a));
		LET $follows_b = (SELECT VALUE out.id FROM follows WHERE in = type::thing("users", $user_b));
		RETURN array::intersect($follows_a, $follows_b);
	`

	results, err := surrealdb.Query[[]any](ctx, r.db, query, map[string]any{
		"user_a": userAID,
		"user_b": userBID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get common follows: %w", err)
	}

	// Extract IDs from result
	var commonIDs []string
	if results != nil && len(*results) > 0 {
		// The result is nested in the query response
		for _, queryResult := range *results {
			for _, item := range queryResult.Result {
				if idStr, ok := item.(string); ok {
					commonIDs = append(commonIDs, idStr)
				}
			}
		}
	}

	return commonIDs, nil
}

// GetCommonFollowers finds users that follow both userA and userB
func (r *followsRepository) GetCommonFollowers(ctx context.Context, userAID, userBID string) ([]string, error) {
	// Use graph traversal to find common followers
	query := `
		LET $followers_a = (SELECT VALUE in.id FROM follows WHERE out = type::thing("users", $user_a));
		LET $followers_b = (SELECT VALUE in.id FROM follows WHERE out = type::thing("users", $user_b));
		RETURN array::intersect($followers_a, $followers_b);
	`

	results, err := surrealdb.Query[[]any](ctx, r.db, query, map[string]any{
		"user_a": userAID,
		"user_b": userBID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get common followers: %w", err)
	}

	// Extract IDs from result
	var commonIDs []string
	if results != nil && len(*results) > 0 {
		// The result is nested in the query response
		for _, queryResult := range *results {
			for _, item := range queryResult.Result {
				if idStr, ok := item.(string); ok {
					commonIDs = append(commonIDs, idStr)
				}
			}
		}
	}

	return commonIDs, nil
}

// Delete removes a follow relationship
func (r *followsRepository) Delete(ctx context.Context, id string) error {
	recordID := models.NewRecordID(followsTable, id)

	_, err := surrealdb.Delete[surrealFollow](ctx, r.db, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete follow: %w", err)
	}

	return nil
}

// DeleteByUsers removes a follow relationship between two specific users
func (r *followsRepository) DeleteByUsers(ctx context.Context, followerID, followeeID string) error {
	query := `
		DELETE FROM follows 
		WHERE in = type::thing("users", $follower_id) 
		AND out = type::thing("users", $followee_id)
	`

	_, err := surrealdb.Query[[]any](ctx, r.db, query, map[string]any{
		"follower_id": followerID,
		"followee_id": followeeID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete follow by users: %w", err)
	}

	return nil
}

// Count returns the total number of follow relationships
func (r *followsRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT count() FROM follows GROUP ALL`

	type countResult struct {
		Count int64 `json:"count"`
	}

	results, err := surrealdb.Query[[]countResult](ctx, r.db, query, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to count follows: %w", err)
	}

	if results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return 0, nil
	}

	return (*results)[0].Result[0].Count, nil
}

// GetTwoRandomUserIDs returns two different random user IDs for creating follows
func (r *followsRepository) GetTwoRandomUserIDs(ctx context.Context) (string, string, error) {
	// Get all user IDs
	query := `SELECT id FROM users`

	type idResult struct {
		ID *models.RecordID `json:"id"`
	}

	results, err := surrealdb.Query[[]idResult](ctx, r.db, query, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user IDs: %w", err)
	}

	if results == nil || len(*results) == 0 || len((*results)[0].Result) < 2 {
		return "", "", fmt.Errorf("need at least 2 users to create follow relationship")
	}

	allIDs := (*results)[0].Result

	// Pick two different random users
	idx1 := rand.Intn(len(allIDs))
	idx2 := rand.Intn(len(allIDs))

	// Ensure we get different users
	for idx2 == idx1 && len(allIDs) > 1 {
		idx2 = rand.Intn(len(allIDs))
	}

	var id1, id2 string
	if allIDs[idx1].ID != nil {
		id1 = fmt.Sprintf("%v", allIDs[idx1].ID.ID)
	}
	if allIDs[idx2].ID != nil {
		id2 = fmt.Sprintf("%v", allIDs[idx2].ID.ID)
	}

	if id1 == "" || id2 == "" {
		return "", "", fmt.Errorf("invalid record IDs")
	}

	return id1, id2, nil
}

// GetRandomFollowID returns a random follow relationship ID
func (r *followsRepository) GetRandomFollowID(ctx context.Context) (string, error) {
	query := `SELECT id FROM follows`

	type idResult struct {
		ID *models.RecordID `json:"id"`
	}

	results, err := surrealdb.Query[[]idResult](ctx, r.db, query, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get follow IDs: %w", err)
	}

	if results == nil || len(*results) == 0 || len((*results)[0].Result) == 0 {
		return "", fmt.Errorf("no follows available")
	}

	allIDs := (*results)[0].Result
	randomIdx := rand.Intn(len(allIDs))
	recordID := allIDs[randomIdx].ID

	if recordID != nil {
		return fmt.Sprintf("%v", recordID.ID), nil
	}

	return "", fmt.Errorf("invalid record ID")
}

// Helper function to convert SurrealDB follow to domain follow
func surrealFollowToDomain(sf *surrealFollow) *domain.Follow {
	var id, followerID, followeeID string

	if sf.ID != nil {
		id = fmt.Sprintf("%v", sf.ID.ID)
	}
	if sf.In != nil {
		followerID = fmt.Sprintf("%v", sf.In.ID)
	}
	if sf.Out != nil {
		followeeID = fmt.Sprintf("%v", sf.Out.ID)
	}

	return &domain.Follow{
		ID:               id,
		FollowerID:       followerID,
		FolloweeID:       followeeID,
		RelationshipType: domain.RelationshipType(sf.RelationshipType),
		Strength:         sf.Strength,
		CreatedAt:        sf.CreatedAt,
		UpdatedAt:        sf.UpdatedAt,
	}
}
