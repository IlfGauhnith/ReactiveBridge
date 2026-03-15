package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSAPI defines the interface for the SQS client (useful for Mockery later).
type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
}

// Producer defines the interface for publishing events.
type Producer interface {
	Publish(ctx context.Context, event models.EventEnvelope) error
}

// SQSProducer handles publishing to AWS SQS.
type SQSProducer struct {
	client   SQSAPI
	queueURL string
}

// NewSQSProducer initializes a new SQS producer and resolves the Queue URL.
func NewSQSProducer(ctx context.Context, queueName string) (*SQSProducer, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := sqs.NewFromConfig(cfg)

	// Dynamically resolve the Queue URL from the Queue Name
	urlResult, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL for %s: %w", queueName, err)
	}

	return &SQSProducer{
		client:   client,
		queueURL: *urlResult.QueueUrl,
	}, nil
}

// Publish sends an event to the SQS queue.
func (s *SQSProducer) Publish(ctx context.Context, event models.EventEnvelope) error {
	// SQS requires a string payload, so we marshal the envelope
	msgBody, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = s.client.SendMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(msgBody)),
		QueueUrl:    aws.String(s.queueURL),
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}