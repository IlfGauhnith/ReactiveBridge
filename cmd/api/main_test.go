package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
	"github.com/IlfGauhnith/ReactiveBridge/internal/queue/mocks"
	"github.com/stretchr/testify/mock"
)

func TestIngestHandler_MethodNotAllowed(t *testing.T) {
	// 1. Create a GET request (our handler should only accept POST)
	req, err := http.NewRequest("GET", "/events", nil)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a server with a mock producer
	mockProducer := mocks.NewMockProducer(t)
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
	mockProducer := mocks.NewMockProducer(t)
	server := &Server{producer: mockProducer}

	handler := http.HandlerFunc(server.ingestHandler)

	handler.ServeHTTP(rr, req)

	// 2. Assert the handler catches the bad payload
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestIngestHandler_Success(t *testing.T) {
	// 1. Instantiate the mock producer
	mockProducer := mocks.NewMockProducer(t)

	// 2. Set up the expectation
	mockProducer.EXPECT().Publish(mock.Anything, mock.AnythingOfType("models.EventEnvelope")).Return(nil).Once()

	// 3. Create a server with the mock producer
	server := &Server{producer: mockProducer}

	// 4. Create a valid request
	event := models.EventEnvelope{
		ID:        "test-id",
		Source:    "test-source",
		EventType: "test-event",
		Timestamp: time.Now().Unix(),
		Data:      json.RawMessage(`{"key":"value"}`),
	}
	body, _ := json.Marshal(event)
	req, err := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 5. Send the request and assert the response
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.ingestHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusAccepted)
	}
}
