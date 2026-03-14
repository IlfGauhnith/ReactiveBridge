package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
)

// MockProducer is a mock implementation of the Producer interface for testing.
type MockProducer struct {
	PublishFunc func(ctx context.Context, event models.EventEnvelope) error
}

// Publish delegates to the mock function.
func (m *MockProducer) Publish(ctx context.Context, event models.EventEnvelope) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event)
	}
	return nil // Default behavior: do nothing and return no error
}

func TestIngestHandler_MethodNotAllowed(t *testing.T) {
	// 1. Create a GET request (our handler should only accept POST)
	req, err := http.NewRequest("GET", "/events", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	
	// Create a server with a mock producer
	mockProducer := &MockProducer{}
	server := &Server{producer: mockProducer}

	handler := http.HandlerFunc(server.ingestHandler)

	// 3. Serve the HTTP request to our recorder
	handler.ServeHTTP(rr, req)

	// 4. Assert the status code
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestIngestHandler_BadRequest(t *testing.T) {
	// 1. Create a POST request with malformed JSON
	badJSON := []byte(`{"id": "123", "source": "test-app", "event_type": "user.created", "data": "missing-quotes}`)
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(badJSON))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	
	// Create a server with a mock producer
	mockProducer := &MockProducer{}
	server := &Server{producer: mockProducer}

	handler := http.HandlerFunc(server.ingestHandler)

	handler.ServeHTTP(rr, req)

	// 2. Assert the handler catches the bad payload
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}
