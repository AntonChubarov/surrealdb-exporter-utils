package executables

import (
	"time"
)

// Collector defines the interface for collecting metrics
type Collector interface {
	// RecordOperation records a completed operation
	RecordOperation(operation string, duration time.Duration, success bool)
}
