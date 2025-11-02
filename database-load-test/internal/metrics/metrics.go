package metrics

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Snapshot represents a point-in-time view of metrics
// (public API preserved)
type Snapshot struct {
	Timestamp   time.Time
	Overall     map[string]Stats
	CurrentStep map[string]Stats
}

// Stats represents aggregated statistics for a time period
// (public API preserved)
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
// It uses:
//   - Welford's algorithm for precise running mean (no precision loss)
//   - Fixed-size reservoir sampling for accurate percentiles without storing all history
//   - Exact stats for the current step from raw buffered samples
//   - Hazen-style percentile interpolation to avoid repeated values on small N
type collector struct {
	mu               sync.RWMutex
	operationsByType map[string]*operationData
	rnd              *rand.Rand
}

const (
	// size of per-operation overall reservoir (tunable). Large enough for stable percentiles
	defaultReservoirCap = 8192
	defaultStepBufCap   = 256
)

// operationData holds both overall accumulators and current-step buffers
// Overall uses streaming updates + reservoir sampling to compute quantiles from a bounded sample that approximates the full distribution.
type operationData struct {
	// overall accumulators
	overallCount   int64
	overallSuccess int64
	overallFail    int64
	minDur         time.Duration
	maxDur         time.Duration

	// Welford mean/variance (we only expose mean, but keep m2 if needed later)
	mean float64
	m2   float64

	// bounded reservoir for quantiles of overall distribution
	reservoir    []time.Duration
	reservoirCap int
	seen         int64 // total seen samples (for reservoir sampling)

	// current step raw data (cleared after each snapshot)
	curDurations []time.Duration
	curSuccesses []bool
}

// NewCollector creates a new metrics collector
func NewCollector() *collector {
	return &collector{
		operationsByType: make(map[string]*operationData),
		rnd:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RecordOperation records a completed operation (signature preserved)
func (c *collector) RecordOperation(operation string, duration time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, ok := c.operationsByType[operation]
	if !ok {
		data = &operationData{
			minDur:       time.Duration(math.MaxInt64),
			maxDur:       0,
			reservoirCap: defaultReservoirCap,
			reservoir:    make([]time.Duration, 0, defaultReservoirCap),
			curDurations: make([]time.Duration, 0, defaultStepBufCap),
			curSuccesses: make([]bool, 0, defaultStepBufCap),
		}
		c.operationsByType[operation] = data
	}

	// --- Update current step buffers ---
	data.curDurations = append(data.curDurations, duration)
	data.curSuccesses = append(data.curSuccesses, success)

	// --- Streaming overall counters ---
	data.overallCount++
	if success {
		data.overallSuccess++
	} else {
		data.overallFail++
	}
	if duration < data.minDur {
		data.minDur = duration
	}
	if duration > data.maxDur {
		data.maxDur = duration
	}

	// Welford: precise running mean
	// newMean = mean + (x - mean)/n ; m2 += (x-mean)*(x-newMean)
	n := float64(data.overallCount)
	x := float64(duration)
	delta := x - data.mean
	data.mean += delta / n
	data.m2 += delta * (x - data.mean)

	// --- Reservoir sampling (Vitter's Algorithm R) ---
	data.seen++
	if len(data.reservoir) < data.reservoirCap {
		data.reservoir = append(data.reservoir, duration)
	} else {
		// pick random index in [0, seen-1]; if within cap, replace
		j := c.rnd.Int63n(data.seen)
		if j < int64(data.reservoirCap) {
			data.reservoir[j] = duration
		}
	}
}

// GetSnapshot returns a snapshot of current metrics and aggregates current step data (signature preserved)
func (c *collector) GetSnapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := Snapshot{
		Timestamp:   time.Now(),
		Overall:     make(map[string]Stats),
		CurrentStep: make(map[string]Stats),
	}

	for opType, data := range c.operationsByType {
		// Current step precise stats from raw buffers
		current := calculateFromSamples(data.curDurations, data.curSuccesses)

		// Overall stats built from streaming accumulators + reservoir for quantiles
		overall := Stats{}
		overall.Count = data.overallCount
		overall.SuccessCount = data.overallSuccess
		overall.FailureCount = data.overallFail
		if data.overallCount > 0 {
			overall.MinDuration = data.minDur
			overall.MaxDuration = data.maxDur
			overall.Average = time.Duration(data.mean)

			// percentiles from reservoir (bounded, but representative)
			if len(data.reservoir) > 0 {
				// copy+sort to avoid mutating the live reservoir
				arr := make([]time.Duration, len(data.reservoir))
				copy(arr, data.reservoir)
				sort.Slice(arr, func(i, j int) bool { return arr[i] < arr[j] })

				overall.Median = percentileHazen(arr, 50)
				overall.P90 = percentileHazen(arr, 90)
				overall.P95 = percentileHazen(arr, 95)
				overall.P99 = percentileHazen(arr, 99)
			}
		}

		snapshot.CurrentStep[opType] = current
		snapshot.Overall[opType] = overall

		// Clear step buffers to bound memory & mark a new step
		data.curDurations = data.curDurations[:0]
		data.curSuccesses = data.curSuccesses[:0]
	}

	return snapshot
}

// calculateFromSamples computes exact stats for a batch of raw samples (current step)
func calculateFromSamples(durations []time.Duration, successes []bool) Stats {
	var out Stats
	n := len(durations)
	if n == 0 {
		return out
	}

	// counts
	out.Count = int64(n)
	for _, s := range successes {
		if s {
			out.SuccessCount++
		} else {
			out.FailureCount++
		}
	}

	// sort for quantiles
	arr := make([]time.Duration, n)
	copy(arr, durations)
	sort.Slice(arr, func(i, j int) bool { return arr[i] < arr[j] })

	out.MinDuration = arr[0]
	out.MaxDuration = arr[n-1]

	// average (use float then cast to avoid overflow on big n)
	var sum float64
	for _, d := range arr {
		sum += float64(d)
	}
	out.Average = time.Duration(sum / float64(n))

	// percentiles with Hazen interpolation to reduce ties on small samples
	out.Median = percentileHazen(arr, 50)
	out.P90 = percentileHazen(arr, 90)
	out.P95 = percentileHazen(arr, 95)
	out.P99 = percentileHazen(arr, 99)

	return out
}

// percentileHazen computes the p-th percentile using Hazen's definition:
// rank r = 1/2 + p/100*(n)
// then linear interpolation between floor(r) and ceil(r)
func percentileHazen(sorted []time.Duration, p int) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}

	// Clamp p into [0,100]
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}

	r := 0.5 + (float64(p)/100.0)*float64(n)
	// Convert to 0-based indices
	lo := int(math.Floor(r)) - 1
	hi := int(math.Ceil(r)) - 1
	if lo < 0 {
		lo = 0
	}
	if hi < 0 {
		hi = 0
	}
	if lo >= n {
		lo = n - 1
	}
	if hi >= n {
		hi = n - 1
	}

	if lo == hi {
		return sorted[lo]
	}

	w := r - math.Floor(r)
	return lerpDuration(sorted[lo], sorted[hi], w)
}

func lerpDuration(a, b time.Duration, w float64) time.Duration {
	return time.Duration(float64(a)*(1.0-w) + float64(b)*w)
}
