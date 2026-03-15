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

	var event models.EventEnvelope
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Basic Validation
	if event.ID == "" || event.Source == "" || event.EventType == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

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

	// Initialize SQS Producer instead of Kinesis
	qp, err := queue.NewSQSProducer(ctx, "ReactiveBridgeQueue")
	if err != nil {
		log.Fatalf("Failed to initialize SQS producer: %v", err)
	}

	srv := &Server{producer: qp}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", srv.ingestHandler)

	handler := middleware.Logger(mux)

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		log.Println("Starting server on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

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
