package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/marcotuna/adaptive-metrics/cmd/server"
	"github.com/marcotuna/adaptive-metrics/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start the server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	fmt.Println("Adaptive Metrics server started. Press Ctrl+C to stop.")

	// Wait for termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down server...")
	if err := srv.Stop(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
}