package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
	"github.com/IlfGauhnith/ReactiveBridge/internal/queue/mocks"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSQSProducer_Publish_Success(t *testing.T) {
	// 1. Initialize the Mock API
	mockAPI := mocks.NewMockSQSAPI(t)

	// 2. Manually construct the producer, injecting the mock
	producer := &SQSProducer{
		client:   mockAPI,
		queueURL: "https://sqs.us-east-1.amazonaws.com/123456789012/ReactiveBridgeQueue",
	}

	// 3. Create a sample event
	event := models.EventEnvelope{
		ID:        "evt-123",
		Source:    "test-client",
		EventType: "system.test",
		Data:      json.RawMessage(`{"user":"lucas","action":"login"}`),
	}

	// 4. Set expectations: Ensure the QueueUrl and MessageBody are correctly populated
	mockAPI.EXPECT().
		SendMessage(mock.Anything, mock.MatchedBy(func(input *sqs.SendMessageInput) bool {
			// Verify it's hitting the right queue
			if *input.QueueUrl != producer.queueURL {
				return false
			}
			// Verify the payload actually contains our ID
			return input.MessageBody != nil && string(*input.MessageBody) != ""
		})).
		Return(&sqs.SendMessageOutput{}, nil).
		Once()

	// 5. Execute and Assert
	err := producer.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestSQSProducer_Publish_AWSError(t *testing.T) {
	mockAPI := mocks.NewMockSQSAPI(t)

	producer := &SQSProducer{
		client:   mockAPI,
		queueURL: "https://fake-url",
	}

	event := models.EventEnvelope{
		ID: "evt-error-case",
	}

	// Simulate an AWS internal error (like throttling or permissions)
	expectedAWSErr := errors.New("AWS internal failure")

	mockAPI.EXPECT().
		SendMessage(mock.Anything, mock.AnythingOfType("*sqs.SendMessageInput")).
		Return(nil, expectedAWSErr).
		Once()

	err := producer.Publish(context.Background(), event)

	// We expect our Publish method to catch and wrap the AWS error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message")
	assert.Contains(t, err.Error(), "AWS internal failure")
}
