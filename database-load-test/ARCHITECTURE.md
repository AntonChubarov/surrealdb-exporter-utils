# Database Load Test - Architecture Design Document

## Overview

This project implements a flexible database load testing framework using Clean Architecture principles and SOLID design patterns. The framework is designed to be database-agnostic and easily extensible.

## Core Design Principles

### 1. Clean Architecture

The project follows Uncle Bob's Clean Architecture with clear layer separation:

```
┌─────────────────────────────────────────┐
│         Presentation Layer              │
│        (Metrics Display)                │
├─────────────────────────────────────────┤
│         Application Layer               │
│    (Runner, Executor, Executables)      │
├─────────────────────────────────────────┤
│         Domain Layer                    │
│      (Entities, Interfaces)             │
├─────────────────────────────────────────┤
│       Infrastructure Layer              │
│   (Repository Impl, Config Reader)      │
└─────────────────────────────────────────┘
```

**Benefits:**
- Independent of frameworks and UI
- Testable business logic
- Independent of database
- Independent of external agencies

### 2. SOLID Principles Applied

#### Single Responsibility Principle (SRP)
- Each package has one reason to change
- `config`: Handles configuration loading
- `executor`: Handles execution timing
- `metrics`: Handles metrics collection
- `metricsdisplay`: Handles metrics presentation
- `executables`: Defines operations
- `runner`: Orchestrates components

#### Open/Closed Principle (OCP)
- Open for extension, closed for modification
- New executables can be added without changing existing code
- New databases can be supported by implementing repository interface
- Factory pattern enables easy addition of new operations

#### Liskov Substitution Principle (LSP)
- Any repository implementation can substitute another
- All executables follow the same interface
- Config implementations are interchangeable

#### Interface Segregation Principle (ISP)
- Clients only depend on interfaces they use
- `DatabaseConfig` separate from `UserCRUDConfig`
- `AppConfig` separate from metrics configuration
- Each component defines its own config interface

#### Dependency Inversion Principle (DIP)
- High-level modules don't depend on low-level modules
- Both depend on abstractions (interfaces)
- Repository interfaces defined in application layer
- Config interfaces defined by consumers

## Component Details

### 1. Main Package (`cmd/app`)

**Purpose**: Application entry point with minimal logic

**Responsibilities**:
- Parse command-line arguments
- Load configuration
- Initialize dependencies
- Create and start runner
- Handle graceful shutdown

**Design Decision**: Keep main as thin as possible, delegating to runner.

### 2. Configuration (`internal/config`)

**Purpose**: Centralized configuration management

**Structure**:
```
config/
├── interfaces.go  # Config interfaces
└── config.go      # YAML implementation
```

**Key Design Decisions**:
- Interface-based configuration (DIP)
- YAML format for human readability
- Getters for type safety
- Specific interfaces for different concerns (ISP)

**Extension Point**: Add new config interfaces as needed.

### 3. Domain Layer (`internal/domain`)

**Purpose**: Core business entities

**Current Entities**:
- `User`: User entity with CRUD fields
- `PaginationParams`: Reusable pagination
- `PaginatedResult[T]`: Generic paginated response

**Design Decisions**:
- No external dependencies
- Pure data structures
- Generic pagination support

**Extension Point**: Add new domain entities here.

### 4. Repository Layer (`internal/repository`)

**Purpose**: Data access abstraction

**Structure**:
```
repository/
├── interfaces.go          # Repository interfaces (application layer)
├── repository.go          # Placeholder
└── mock/                  # Mock implementation
    └── mock_user_repository.go
```

**Key Interface**:
```go
type UserRepository interface {
    Setup(ctx) error
    Create(ctx, *User) error
    GetByID(ctx, id) (*User, error)
    GetAll(ctx, params) (*PaginatedResult[User], error)
    Update(ctx, *User) error
    Delete(ctx, id) error
    Count(ctx) (int64, error)
    GetRandomID(ctx) (string, error)
}
```

**Design Decisions**:
- Interfaces defined in application layer (DIP)
- Context for cancellation support
- Repository pattern for data access
- Database-agnostic interface

**Extension Points**:
- Add new repository implementations (PostgreSQL, MongoDB, etc.)
- Add new repository interfaces for other entities

### 5. Executables (`internal/executables`)

**Purpose**: Define testable operations

**Structure**:
```
executables/
├── interfaces.go          # Executable interface
├── factory.go             # Factory for creating executables
└── user_executables.go    # User CRUD executables
```

**Key Interface**:
```go
type Executable interface {
    Execute(ctx) error         // Run continuously
    ExecuteOnce(ctx) error     // Run once (for setup)
    Name() string              // Identification
}
```

**Implementations**:
- `setupUsersExecutable`: One-time setup
- `createUserExecutable`: Create operations
- `readUserExecutable`: Read operations
- `updateUserExecutable`: Update operations
- `deleteUserExecutable`: Delete operations
- `getAllUsersExecutable`: Paginated queries

**Factory Pattern**:
- Centralizes creation logic
- Supports weighted distribution
- Easy to add new executable types

**Design Decisions**:
- Separate Execute and ExecuteOnce for setup operations
- Each executable measures its own performance
- Random data generation for realistic testing
- Weighted factory method for configurable ratios

**Extension Points**:
- Add new executable types
- Add executables for other entities
- Customize data generation logic

### 6. Executor (`internal/executor`)

**Purpose**: Execute operations with controlled timing

**Key Features**:
- Normal distribution for timing variation
- Concurrent execution of multiple operations
- Context-aware cancellation
- Configurable time step and variation

**Algorithm**:
```
For each executable:
    While context not cancelled:
        1. Execute operation
        2. Calculate sleep duration (normal distribution)
        3. Sleep
        4. Repeat
```

**Normal Distribution Implementation**:
- Uses Box-Muller transform
- Mean = configured time step
- Standard deviation = configured variation
- Ensures realistic load patterns

**Design Decisions**:
- Goroutine per executable for concurrency
- Non-blocking execution
- Error handling without stopping others
- Precise timing control

**Extension Points**:
- Add different timing distributions
- Add burst patterns
- Add ramp-up/ramp-down logic

### 7. Metrics (`internal/metrics`)

**Purpose**: Collect and aggregate performance metrics

**Structure**:
```
metrics/
├── interfaces.go  # Collector interface
└── metrics.go     # Implementation
```

**Collected Metrics**:
- Total operations count
- Success/failure counts
- Operation duration (min, max, avg)
- Per-operation-type statistics

**Design Decisions**:
- Thread-safe implementation (mutex)
- Snapshot pattern for reading
- Zero-allocation for recording (mostly)
- Real-time aggregation

**Extension Points**:
- Add percentile calculations
- Add histogram support
- Add custom metrics

### 8. Metrics Display (`internal/metricsdisplay`)

**Purpose**: Present metrics to users

**Features**:
- Real-time snapshot display
- Final report generation
- Configurable display interval
- Clear, readable format

**Design Decisions**:
- Separate from collection (SRP)
- Console-based for simplicity
- Non-blocking display
- Comprehensive final report

**Extension Points**:
- Add JSON output
- Add Prometheus exporter
- Add web dashboard
- Add chart generation

### 9. Runner (`internal/runner`)

**Purpose**: Orchestrate the entire load test

**Phases**:
1. **Setup Phase**: Run one-time setup operations
2. **Load Test Phase**: Execute CRUD operations with metrics

**Responsibilities**:
- Initialize all components
- Coordinate phases
- Manage context lifecycle
- Handle errors gracefully

**Design Decisions**:
- Single entry point for execution
- Clear phase separation
- Proper cleanup
- Comprehensive error reporting

## Dependency Flow

```
main.go
  ├─→ config.LoadConfig()
  ├─→ repository.NewUserRepository()
  └─→ runner.New()
        ├─→ metrics.NewCollector()
        ├─→ metricsdisplay.New()
        ├─→ executables.NewFactory()
        ├─→ executor.New()
        └─→ runner.Run()
              ├─→ Setup Phase
              │     └─→ executor.ExecuteOnce()
              └─→ Load Test Phase
                    ├─→ factory.CreateUserCRUDExecutables()
                    ├─→ executor.Execute()
                    └─→ metricsDisplay.Start()
```

## Extensibility Points

### Adding New Operations

1. Create new executable type in `executables/`
2. Implement `Executable` interface
3. Add factory method
4. Update configuration if needed

Example:
```go
type customExecutable struct {
    repo repository.CustomRepository
    collector metrics.Collector
}

func (f *Factory) CreateCustom() Executable {
    return &customExecutable{...}
}
```

### Adding New Databases

1. Create new package under `repository/`
2. Implement `UserRepository` interface
3. Add connection setup logic
4. Update `main.go`

Example structure:
```
repository/
├── postgres/
│   └── user_repository.go
├── mongodb/
│   └── user_repository.go
└── mysql/
    └── user_repository.go
```

### Adding New Entities

1. Define entity in `domain/`
2. Create repository interface
3. Create executables
4. Add to factory
5. Update runner if needed

### Adding New Metrics

1. Extend `Collector` interface
2. Update `collector` implementation
3. Update display logic

## Configuration Strategy

Configuration uses interface segregation:

```
AppConfig
├─→ LoadTest configuration
├─→ Timing configuration
└─→ Duration configuration

DatabaseConfig
├─→ Connection parameters
└─→ Pool configuration

UserCRUDConfig
├─→ Operation rates
└─→ Pagination settings

MetricsConfig
├─→ Collection interval
└─→ Display interval
```

Each component receives only the configuration it needs.

## Error Handling Strategy

1. **Graceful Degradation**: Individual operation failures don't stop the test
2. **Context Propagation**: Cancellation flows through all components
3. **Error Recording**: Failures recorded in metrics
4. **Proper Cleanup**: Deferred cleanup ensures resource release

## Testing Strategy

The architecture enables easy testing:

- **Unit Tests**: Test each component in isolation
- **Mock Repository**: Included for integration testing
- **Interface Mocking**: Easy to mock dependencies
- **Deterministic Tests**: Metrics and timing can be controlled

## Performance Considerations

1. **Concurrency**: Goroutine per operation type
2. **Lock Contention**: Minimal mutex usage
3. **Memory**: Preallocated structures where possible
4. **GC Pressure**: Value types and object reuse
5. **Metrics**: Lock-free recording path

## Future Enhancements

1. **Distribution**: Multiple worker nodes
2. **Scenarios**: Complex operation sequences
3. **Data Validation**: Verify data integrity
4. **Chaos Testing**: Inject failures
5. **Adaptive Load**: Adjust based on response times
6. **Reporting**: Generate detailed HTML reports
7. **Monitoring**: Integrate with observability platforms

## Conclusion

This architecture provides:
- ✅ Clean separation of concerns
- ✅ Easy extensibility
- ✅ Database independence
- ✅ SOLID principles compliance
- ✅ Testability
- ✅ Production-ready patterns
- ✅ Clear documentation
- ✅ Type safety

The framework is ready for extension with real database implementations and additional testing scenarios.
