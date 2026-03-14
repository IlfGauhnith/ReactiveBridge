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

	"github.com/IlfGauhnith/ReactiveBridge/internal/middleware"
	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
	"github.com/IlfGauhnith/ReactiveBridge/internal/queue"
)

type Server struct {
	producer queue.Producer
}

func (s *Server) ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event models.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Basic Validation
	if event.UserID == "" || event.Source == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Asynchronously publish to Kinesis (or synchronously if strict durability is required)
	// Using context from request to handle cancellations
	if err := s.producer.Publish(r.Context(), event); err != nil {
		log.Printf("Failed to publish event: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"received"}`))
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Kinesis Producer
	// In a real scenario, "ReactiveBridgeStream" would come from an environment variable
	kp, err := queue.NewKinesisProducer(ctx, "ReactiveBridgeStream")
	if err != nil {
		// Fallback for local dev if AWS credentials aren't set, or panic.
		// For now, we log and exit as this is critical infrastructure.
		log.Fatalf("Failed to initialize Kinesis producer: %v", err)
	}

	srv := &Server{producer: kp}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", srv.ingestHandler)

	// Wrap mux with middleware
	handler := middleware.Logger(mux)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Server start in goroutine
	go func() {
		log.Println("Starting server on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exiting")
}