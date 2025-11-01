package executor

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
)

// Executable defines the interface for operations that can be executed
type Executable interface {
	// Execute runs the operation
	Execute(ctx context.Context) error

	// Name returns the name of the executable for identification
	Name() string
}

// executableTask represents a recurring executable with its timing parameters
type executableTask struct {
	exec                      Executable
	delay                     time.Duration
	timeStep                  time.Duration
	timeStepStandardDeviation time.Duration
}

// singleTask represents a one-time executable with delay
type singleTask struct {
	exec  Executable
	delay time.Duration
	async bool
}

// Executor manages and executes operations with configurable timing
type Executor struct {
	recurringTasks []executableTask
	singleTasks    []singleTask
	mu             sync.Mutex
}

// New creates a new executor
func New() *Executor {
	return &Executor{
		recurringTasks: make([]executableTask, 0),
		singleTasks:    make([]singleTask, 0),
	}
}

// Add adds a recurring executable with individual timing parameters
func (e *Executor) Add(exec Executable, eventRate domain.EventRate) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.recurringTasks = append(e.recurringTasks, executableTask{
		exec:                      exec,
		delay:                     eventRate.StartDelay,
		timeStep:                  eventRate.TimeStep(),
		timeStepStandardDeviation: eventRate.TimeStepStandardDeviation(),
	})
}

// AddSingle adds a one-time executable that will be executed synchronously
func (e *Executor) AddSingle(exec Executable, delay time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.singleTasks = append(e.singleTasks, singleTask{
		exec:  exec,
		delay: delay,
		async: false,
	})
}

// AddSingleAsync adds a one-time executable that will be executed asynchronously
func (e *Executor) AddSingleAsync(exec Executable, delay time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.singleTasks = append(e.singleTasks, singleTask{
		exec:  exec,
		delay: delay,
		async: true,
	})
}

// Run starts execution of all added executables
func (e *Executor) Run(ctx context.Context) error {
	e.mu.Lock()
	recurringTasks := make([]executableTask, len(e.recurringTasks))
	copy(recurringTasks, e.recurringTasks)
	singleTasks := make([]singleTask, len(e.singleTasks))
	copy(singleTasks, e.singleTasks)
	e.mu.Unlock()

	if len(recurringTasks) == 0 && len(singleTasks) == 0 {
		return fmt.Errorf("no executables to run")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(recurringTasks)+len(singleTasks))

	// Start all recurring tasks
	for _, task := range recurringTasks {
		wg.Add(1)
		go func(t executableTask) {
			defer wg.Done()
			if err := e.runRecurringTask(ctx, t); err != nil {
				errChan <- fmt.Errorf("%s (recurring) failed: %w", t.exec.Name(), err)
			}
		}(task)
	}

	// Execute single tasks
	for _, task := range singleTasks {
		if task.async {
			wg.Add(1)
			go func(t singleTask) {
				defer wg.Done()
				if err := e.runSingleTask(ctx, t); err != nil {
					errChan <- fmt.Errorf("%s (single async) failed: %w", t.exec.Name(), err)
				}
			}(task)
		} else {
			// Execute synchronous single tasks inline
			if err := e.runSingleTask(ctx, task); err != nil {
				errChan <- fmt.Errorf("%s (single sync) failed: %w", task.exec.Name(), err)
			}
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("executor completed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// runRecurringTask executes a recurring task with initial delay and periodic intervals
func (e *Executor) runRecurringTask(ctx context.Context, task executableTask) error {
	// Initial delay
	if task.delay > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(task.delay):
		}
	}

	// Recurring execution loop
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Execute the operation
			_ = task.exec.Execute(ctx)

			// Calculate sleep duration for this specific task
			sleepDuration := calculateSleepDuration(task.timeStep, task.timeStepStandardDeviation)

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(sleepDuration):
				// Continue to next iteration
			}
		}
	}
}

// runSingleTask executes a single task once after the specified delay
func (e *Executor) runSingleTask(ctx context.Context, task singleTask) error {
	// Wait for delay
	if task.delay > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(task.delay):
		}
	}

	// Execute once
	return task.exec.Execute(ctx)
}

// calculateSleepDuration calculates the next sleep duration using normal distribution
func calculateSleepDuration(timeStep, timeStepStandardDeviation time.Duration) time.Duration {
	if timeStepStandardDeviation == 0 {
		return timeStep
	}

	// Generate a random value from normal distribution
	// Mean = timeStep, StdDev = timeStepStandardDeviation
	variation := normalDistribution(0, float64(timeStepStandardDeviation))
	duration := float64(timeStep) + variation

	// Ensure duration is positive
	if duration < 0 {
		duration = float64(timeStep)
	}

	return time.Duration(duration)
}

// normalDistribution generates a random value from a normal distribution
// using the Box-Muller transform
func normalDistribution(mean, stdDev float64) float64 {
	// Box-Muller transform
	u1 := rand.Float64()
	u2 := rand.Float64()

	// Ensure u1 is not zero to avoid log(0)
	for u1 == 0 {
		u1 = rand.Float64()
	}

	z0 := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
	return mean + stdDev*z0
}
