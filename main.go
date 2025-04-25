// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eGGnogSC/qbserver/infrastructure"
	"github.com/eGGnogSC/qbserver/config"
)

func main() {
	// Load configuration from environment/files
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Create dependency container
	container, err := infrastructure.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer container.Shutdown()
	
	// Create application context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start health checking and replication
	fallbackStore, ok := container.TokenStore.(*auth.FallbackTokenStore)
	if ok {
		fallbackStore.StartReplicationRoutine(ctx)
	}
	
	// Set up and start HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      setupRoutes(container),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	// Shutdown gracefully
	log.Println("Shutting down server...")
	
	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	
	log.Println("Server gracefully stopped")
}
