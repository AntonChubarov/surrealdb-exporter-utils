package domain

import "time"

type User struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RelationshipType represents the type of follow relationship between users
type RelationshipType string

const (
	RelationshipTypeFriend       RelationshipType = "friend"
	RelationshipTypeColleague    RelationshipType = "colleague"
	RelationshipTypeAcquaintance RelationshipType = "acquaintance"
	RelationshipTypeMentor       RelationshipType = "mentor"
	RelationshipTypeFamily       RelationshipType = "family"
	RelationshipTypeProfessional RelationshipType = "professional"
	RelationshipTypeCasual       RelationshipType = "casual"
)

// AllRelationshipTypes returns all available relationship types
func AllRelationshipTypes() []RelationshipType {
	return []RelationshipType{
		RelationshipTypeFriend,
		RelationshipTypeColleague,
		RelationshipTypeAcquaintance,
		RelationshipTypeMentor,
		RelationshipTypeFamily,
		RelationshipTypeProfessional,
		RelationshipTypeCasual,
	}
}

// Follow represents a graph edge between two users with relationship metadata
type Follow struct {
	ID               string
	FollowerID       string // The user who is following (in)
	FolloweeID       string // The user being followed (out)
	RelationshipType RelationshipType
	Strength         int // Relationship strength 1-10
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FollowWithUsers represents a follow relationship with expanded user data
type FollowWithUsers struct {
	Follow
	Follower *User
	Followee *User
}

// CommonFollowResult represents the result of common follows/followers queries
type CommonFollowResult struct {
	UserID string
	User   *User
	Count  int
}

type EventRate struct {
	StartDelay       time.Duration
	EventsPerMinute  float64
	VariancePercents float64
}

func (er *EventRate) TimeStep() time.Duration {
	if er.EventsPerMinute <= 0 {
		return 0
	}

	// Convert events per minute to time between events
	secondsBetweenEvents := 60.0 / er.EventsPerMinute
	return time.Duration(secondsBetweenEvents * float64(time.Second))
}

func (er *EventRate) TimeStepStandardDeviation() time.Duration {
	meanTimeStep := er.TimeStep()
	if meanTimeStep == 0 {
		return 0
	}

	// VariancePercents represents the coefficient of variation (CV)
	// Standard deviation = mean * CV / 100 / 3 (3 sigma)
	stdDev := float64(meanTimeStep) * er.VariancePercents / 100.0 / 3
	return time.Duration(stdDev)
}

type PaginationParams struct {
	Page     int
	PageSize int
}

type PaginatedResult[T any] struct {
	Items      []T
	TotalCount int64
	Page       int
	PageSize   int
	TotalPages int
}
