package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	// Create mock Discord API server
	server := NewMockDiscordServer()

	// Create Gateway server
	gateway := NewGatewayServer()

	// Create Voice server
	voiceServer := NewVoiceServer()

	// Setup HTTP router
	router := mux.NewRouter()
	server.SetupRoutes(router)

	// Add Gateway WebSocket endpoint
	router.HandleFunc("/gateway", gateway.HandleWebSocket)

	// Add voice server endpoints
	router.HandleFunc("/voice/connections", func(w http.ResponseWriter, r *http.Request) {
		connections := voiceServer.GetActiveConnections()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(connections)
	}).Methods("GET")

	// Configure HTTP server
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gateway.Start(ctx)

	// Start voice server on port 8081
	if err := voiceServer.Start(ctx, "8081"); err != nil {
		log.Fatalf("Failed to start voice server: %v", err)
	}
	defer voiceServer.Stop()

	// Start HTTP server in goroutine
	go func() {
		log.Println("Mock Discord API server starting on :8080")
		log.Println("Gateway WebSocket available at ws://localhost:8080/gateway")
		log.Println("Voice server available on :8081")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel Gateway context
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
