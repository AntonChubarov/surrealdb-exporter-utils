package metrics

import (
	"sort"
	"sync"
	"time"
)

// Snapshot represents a point-in-time view of metrics
type Snapshot struct {
	Timestamp   time.Time
	Overall     map[string]Stats
	CurrentStep map[string]Stats
}

// Stats represents aggregated statistics for a time period
type Stats struct {
	Count        int64
	SuccessCount int64
	FailureCount int64
	MinDuration  time.Duration
	Average      time.Duration
	Median       time.Duration
	P90          time.Duration
	P95          time.Duration
	P99          time.Duration
	MaxDuration  time.Duration
}

// collector is the concrete implementation of metrics collector
type collector struct {
	mu               sync.RWMutex
	operationsByType map[string]*operationData
}

// operationData holds both overall stats and current raw data
type operationData struct {
	overall Stats

	// Current step raw data (cleared after each snapshot)
	currentDurations []time.Duration
	currentSuccesses []bool
}

// NewCollector creates a new metrics collector
func NewCollector() *collector {
	return &collector{
		operationsByType: make(map[string]*operationData),
	}
}

// RecordOperation records a completed operation
func (c *collector) RecordOperation(operation string, duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, exists := c.operationsByType[operation]
	if !exists {
		data = &operationData{
			currentDurations: make([]time.Duration, 0, 100),
			currentSuccesses: make([]bool, 0, 100),
		}
		c.operationsByType[operation] = data
	}

	data.currentDurations = append(data.currentDurations, duration)
	data.currentSuccesses = append(data.currentSuccesses, success)
}

// GetSnapshot returns a snapshot of current metrics and aggregates current step data
func (c *collector) GetSnapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := Snapshot{
		Timestamp:   time.Now(),
		Overall:     make(map[string]Stats),
		CurrentStep: make(map[string]Stats),
	}

	for opType, data := range c.operationsByType {
		// Calculate current step stats from raw data
		current := c.calculateStats(data.currentDurations, data.currentSuccesses)

		// Aggregate current into overall
		overall := c.aggregateStats(data.overall, current)

		snapshot.CurrentStep[opType] = current
		snapshot.Overall[opType] = overall

		// Update overall stats and clear current data to save memory
		data.overall = overall
		data.currentDurations = make([]time.Duration, 0, 100)
		data.currentSuccesses = make([]bool, 0, 100)
	}

	return snapshot
}

// calculateStats computes histogram statistics from raw data
func (c *collector) calculateStats(durations []time.Duration, successes []bool) Stats {
	stats := Stats{}

	count := len(durations)
	if count == 0 {
		return stats
	}

	stats.Count = int64(count)

	// Count successes/failures
	for _, success := range successes {
		if success {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
	}

	// Sort durations for percentile calculations
	sorted := make([]time.Duration, count)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate min, max, and sum
	stats.MinDuration = sorted[0]
	stats.MaxDuration = sorted[count-1]

	var sum time.Duration
	for _, d := range sorted {
		sum += d
	}
	stats.Average = sum / time.Duration(count)

	// Calculate percentiles
	stats.Median = c.percentile(sorted, 50)
	stats.P90 = c.percentile(sorted, 90)
	stats.P95 = c.percentile(sorted, 95)
	stats.P99 = c.percentile(sorted, 99)

	return stats
}

// percentile calculates the nth percentile from sorted data using linear interpolation
func (c *collector) percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Calculate the rank using the nearest-rank method with interpolation
	rank := float64(p) / 100.0 * float64(len(sorted)-1)
	lower := int(rank)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation between lower and upper values
	weight := rank - float64(lower)
	return time.Duration(float64(sorted[lower])*(1-weight) + float64(sorted[upper])*weight)
}

// aggregateStats combines two Stats into an overall aggregate
func (c *collector) aggregateStats(overall, current Stats) Stats {
	if current.Count == 0 {
		return overall
	}

	if overall.Count == 0 {
		return current
	}

	result := Stats{
		Count:        overall.Count + current.Count,
		SuccessCount: overall.SuccessCount + current.SuccessCount,
		FailureCount: overall.FailureCount + current.FailureCount,
	}

	// Min/Max
	result.MinDuration = min(overall.MinDuration, current.MinDuration)
	result.MaxDuration = max(overall.MaxDuration, current.MaxDuration)

	// Weighted average
	totalDuration := overall.Average*time.Duration(overall.Count) + current.Average*time.Duration(current.Count)
	result.Average = totalDuration / time.Duration(result.Count)

	// For percentiles, use weighted approximation
	// Note: This is an approximation since we don't retain all historical raw data
	weight1 := float64(overall.Count) / float64(result.Count)
	weight2 := float64(current.Count) / float64(result.Count)

	result.Median = c.weightedDuration(overall.Median, current.Median, weight1, weight2)
	result.P90 = c.weightedDuration(overall.P90, current.P90, weight1, weight2)
	result.P95 = c.weightedDuration(overall.P95, current.P95, weight1, weight2)
	result.P99 = c.weightedDuration(overall.P99, current.P99, weight1, weight2)

	return result
}

// weightedDuration calculates weighted average of two durations
func (c *collector) weightedDuration(d1, d2 time.Duration, w1, w2 float64) time.Duration {
	return time.Duration(float64(d1)*w1 + float64(d2)*w2)
}
