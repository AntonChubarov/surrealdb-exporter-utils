package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/domain"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/executables"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/executor"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/metrics"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/metricsdisplay"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/repository/surrealdb"
)

// AppConfig is the main application configuration interface
type AppConfig interface {
	LoadTestDuration() time.Duration

	MetricsDisplayInterval() time.Duration

	SurrealURL() string
	SurrealNamespace() string
	SurrealDatabase() string
	SurrealUsername() string
	SurrealPassword() string

	UsersCreateParams() domain.EventRate
	UsersReadParams() domain.EventRate
	UsersUpdateParams() domain.EventRate
	UsersDeleteParams() domain.EventRate
	UsersGetAllParams() domain.EventRate
	UsersPageSize() int
}

// Runner orchestrates the load test execution
type Runner struct {
	cfg              AppConfig
	userRepo         executables.UserRepository
	metricsCollector executables.Collector
	metricsDisplay   *metricsdisplay.Display
	userExecFactory  *executables.UserExecutablesFactory
	executor         *executor.Executor
}

// New creates a new runner instance
func New(ctx context.Context, cfg AppConfig) (*Runner, error) {
	surrealDBInstance, err := surrealdb.NewConnection(ctx, cfg)
	if err != nil {
		return nil, err
	}

	userRepo := surrealdb.NewUserRepository(surrealDBInstance)

	metricsCollector := metrics.NewCollector()

	metricsDisplay := metricsdisplay.New(metricsCollector, cfg)

	userExecFactory := executables.NewUserExecutablesFactory(userRepo, metricsCollector)

	return &Runner{
		cfg:              cfg,
		userRepo:         userRepo,
		metricsCollector: metricsCollector,
		metricsDisplay:   metricsDisplay,
		userExecFactory:  userExecFactory,
	}, nil
}

// Run executes the load test
func (r *Runner) Run(ctx context.Context) error {
	fmt.Println("Starting database load test...")
	fmt.Printf("Duration: %v\n", r.cfg.LoadTestDuration())
	fmt.Println()

	// Create a context with timeout for the load test duration
	loadTestCtx, cancel := context.WithTimeout(ctx, r.cfg.LoadTestDuration())
	defer cancel()

	// Start metrics display in a separate goroutine
	displayDone := make(chan struct{})
	go func() {
		r.metricsDisplay.Start(loadTestCtx)
		close(displayDone)
	}()

	loadTestStart := time.Now()

	// Phase 1: Setup
	if err := r.runSetup(loadTestCtx); err != nil {
		return fmt.Errorf("setup phase failed: %w", err)
	}

	// Phase 2: Load test execution
	err := r.runLoadTest(loadTestCtx)

	loadTestDuration := time.Since(loadTestStart)

	// Wait for metrics display to finish
	<-displayDone

	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("load test execution failed: %w", err)
	}

	fmt.Printf("\nLoad test completed in %v\n", loadTestDuration)

	return nil
}

// runSetup performs one-time setup operations
func (r *Runner) runSetup(ctx context.Context) error {
	fmt.Println("=== Setup Phase ===")

	setupExecutor := executor.New()

	setupExecutor.AddSingle(r.userExecFactory.SetupUsersExecutable(), 0)

	if err := setupExecutor.Run(ctx); err != nil {
		return fmt.Errorf("failed to execute setup: %w", err)
	}

	fmt.Println("Setup completed successfully")
	fmt.Println()

	return nil
}

// runLoadTest executes the main load test
func (r *Runner) runLoadTest(ctx context.Context) error {
	fmt.Println("=== Load Test Phase ===")

	loadTestExecutor := executor.New()

	loadTestExecutor.Add(r.userExecFactory.UserCreateExecutable(), r.cfg.UsersCreateParams())
	loadTestExecutor.Add(r.userExecFactory.UserReadExecutable(), r.cfg.UsersReadParams())
	loadTestExecutor.Add(r.userExecFactory.UserUpdateExecutable(), r.cfg.UsersUpdateParams())
	loadTestExecutor.Add(r.userExecFactory.UserDeleteExecutable(), r.cfg.UsersDeleteParams())
	loadTestExecutor.Add(r.userExecFactory.UserGetAllExecutable(r.cfg.UsersPageSize()), r.cfg.UsersGetAllParams())

	fmt.Printf("Load test executor set up")
	fmt.Println()

	return loadTestExecutor.Run(ctx)
}
