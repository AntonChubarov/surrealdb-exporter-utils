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
