.PHONY: build run clean test fmt vet

# Build the application
build:
	@echo "Building application..."
	@go build -o bin/loadtest ./cmd/app

# Run the application
run: build
	@echo "Running load test..."
	@./bin/loadtest

# Run with custom config
run-config: build
	@echo "Running load test with custom config..."
	@./bin/loadtest -config $(CONFIG)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Build and run the application"
	@echo "  run-config  - Run with custom config (CONFIG=path/to/config.yaml)"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  deps        - Download dependencies"
	@echo "  tidy        - Tidy dependencies"
	@echo "  lint        - Run linter"
