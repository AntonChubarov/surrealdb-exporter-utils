# Database Load Test

A flexible and extensible database load testing framework written in Go, following Clean Architecture principles and SOLID design patterns.

## Project Structure

```
database-load-test/
├── cmd/
│   └── app/
│       └── main.go                 # Application entry point (minimal code)
├── internal/
│   ├── config/                     # Configuration management
│   │   ├── config.go              # Config implementation with YAML loading
│   │   └── interfaces.go          # Config interfaces (AppConfig, DatabaseConfig, etc.)
│   ├── domain/                    # Domain entities
│   │   └── user.go                # User entity and pagination types
│   ├── repository/                # Repository layer
│   │   ├── interfaces.go          # Repository interfaces (UserRepository)
│   │   └── repository.go          # Placeholder for DB implementations
│   ├── executables/               # Executable operations factory
│   │   ├── factory.go             # Factory for creating executables
│   │   ├── interfaces.go          # Executable interface
│   │   └── user_executables.go   # User CRUD executables
│   ├── executor/                  # Execution engine
│   │   └── executor.go            # Executes operations with timing variation
│   ├── metrics/                   # Metrics collection
│   │   ├── interfaces.go          # Metrics interfaces
│   │   └── metrics.go             # Metrics collector implementation
│   ├── metricsdisplay/           # Metrics display
│   │   └── display.go             # Console metrics display
│   └── runner/                    # Application orchestration
│       └── runner.go              # Composes and runs the application
├── configs/
│   └── config.yaml                # YAML configuration file
├── go.mod
└── README.md
```

## Architecture

The project follows Clean Architecture principles with clear separation of concerns:

### Layers

1. **Domain Layer** (`internal/domain`)
   - Contains business entities
   - No dependencies on other layers
   - Example: User entity

2. **Application Layer** (interfaces in `internal/repository`, `internal/config`, etc.)
   - Defines interfaces for repositories and configurations
   - Ensures dependency inversion (SOLID principle)

3. **Infrastructure Layer** (`internal/repository`, `internal/config`)
   - Implements interfaces defined in the application layer
   - Handles external concerns (database, file I/O)

4. **Presentation Layer** (`internal/metricsdisplay`)
   - Displays metrics to the user

### Dependencies

```
main -> runner
runner -> config, executables factory, executor, repository, metrics, metrics-display
executables factory -> domain, repository, metrics
executor -> executables
metrics-display -> metrics
```

## Features

- **Configurable Load Testing**: YAML-based configuration for all parameters
- **CRUD Operations**: Full user management operations (Create, Read, Update, Delete, GetAll)
- **Weighted Distribution**: Configure operation rates (e.g., 40% reads, 30% updates)
- **Timing Variation**: Normal distribution for realistic load patterns
- **Real-time Metrics**: Live performance statistics during test execution
- **Clean Architecture**: Easy to extend with new operations or databases
- **Repository Pattern**: Abstract database layer for multiple DB support

## Configuration

Edit `configs/config.yaml`:

```yaml
database:
  host: "localhost"
  port: 5432
  database: "loadtest"
  username: "postgres"
  password: "postgres"
  max_connections: 100

load_test:
  duration_seconds: 300          # Test duration
  time_step_ms: 100             # Base time between operations
  time_step_variation_ms: 20    # Variation using normal distribution

executables:
  user_crud:
    create_rate: 10   # 10% creates
    read_rate: 40     # 40% reads
    update_rate: 30   # 30% updates
    delete_rate: 10   # 10% deletes
    get_all_rate: 10  # 10% paginated queries
    page_size: 20     # Results per page

metrics:
  collection_interval_ms: 1000
  display_interval_ms: 5000
```

## Implementation Guide

### 1. Implement Repository

Create a database-specific repository implementation in `internal/repository/`:

```go
// Example: internal/repository/postgres/user_repository.go
package postgres

import (
    "context"
    "database/sql"
    
    "github.com/yourorg/database-load-test/internal/domain"
    "github.com/yourorg/database-load-test/internal/repository"
    "github.com/yourorg/database-load-test/internal/config"
)

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(cfg config.DatabaseConfig) (repository.UserRepository, error) {
    // Initialize database connection
    db, err := sql.Open("postgres", buildDSN(cfg))
    if err != nil {
        return nil, err
    }
    
    return &userRepository{db: db}, nil
}

func (r *userRepository) Setup(ctx context.Context) error {
    // Create tables, indexes, etc.
    return nil
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
    // Implement user creation
    return nil
}

// ... implement other methods
```

### 2. Update main.go

Replace the placeholder in `cmd/app/main.go`:

```go
// Initialize your repository
userRepo, err := postgres.NewUserRepository(cfg)
if err != nil {
    log.Fatalf("Failed to initialize repository: %v", err)
}
```

### 3. Build and Run

```bash
# Build the application
go build -o bin/loadtest ./cmd/app

# Run with default config
./bin/loadtest

# Run with custom config
./bin/loadtest -config /path/to/config.yaml
```

## Extending the Framework

### Adding New Operations

1. Create new executable in `internal/executables/`
2. Add factory method in `factory.go`
3. Update configuration interfaces and implementation
4. Add to runner if needed

### Adding New Entities

1. Define entity in `internal/domain/`
2. Create repository interface in `internal/repository/interfaces.go`
3. Implement executables
4. Add to factory

### Supporting New Databases

1. Create new package under `internal/repository/`
2. Implement `UserRepository` interface
3. Add database-specific connection logic
4. Update `main.go` to use new implementation

## SOLID Principles Applied

- **Single Responsibility**: Each package has one clear purpose
- **Open/Closed**: Easy to add new executables without modifying existing code
- **Liskov Substitution**: Any repository implementation can replace another
- **Interface Segregation**: Specific config interfaces for each component
- **Dependency Inversion**: High-level modules depend on abstractions (interfaces)

## Metrics

The framework tracks:
- Total operations executed
- Success/failure rates
- Operation counts by type
- Average, min, and max durations
- Real-time and final reports

## License

MIT
