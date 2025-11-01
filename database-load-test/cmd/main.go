package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/config"
	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/runner"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Using mock repository (in-memory)")
	fmt.Println("For production use, implement and use a real database repository")
	fmt.Println()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Create runner
	r, err := runner.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Run the load test
	if err = r.Run(ctx); err != nil {
		log.Fatalf("Load testing failed: %v", err)
	}

	fmt.Println("Load test completed successfully")
}
