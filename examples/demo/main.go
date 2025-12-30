package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"glib/demo/generated"
)

func main() {
	ctx := context.Background()
	
	// Load configuration
	cfg := LoadConfig()
	
	// Bootstrap application
	handler, err := generated.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	
	// Create server
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("🛑 Shutting down server...")
	
	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	
	log.Println("✅ Server stopped")
}
