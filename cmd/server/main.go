ipackage main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/eGGnogSC/qbserver/config"
	"github.com/eGGnogSC/qbserver/infrastructure"
	"github.com/eGGnogSC/qbserver/routes"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create dependency container
	container, err := infrastructure.NewContainer(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer container.Shutdown()
	
	// Create router
	router := mux.NewRouter()
	
	// Set up routes
	routes.SetupRoutes(
		router,
		container.AuthHandler,
		container.AuthService,
		container.InvoiceHandler,
		container.CustomerHandler,
		container.ItemHandler,
		container.PaymentHandler,
		container.AgentHandler,
	)
	
	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.Timeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.Timeout) * time.Second,
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
