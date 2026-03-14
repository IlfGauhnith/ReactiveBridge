package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
)

// Producer defines the interface for publishing events.
type Producer interface {
	Publish(ctx context.Context, event models.Event) error
}

// KinesisProducer handles publishing to AWS Kinesis.
type KinesisProducer struct {
	client     *kinesis.Client
	streamName string
}

// NewKinesisProducer initializes a new Kinesis producer.
func NewKinesisProducer(ctx context.Context, streamName string) (*KinesisProducer, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &KinesisProducer{
		client:     kinesis.NewFromConfig(cfg),
		streamName: streamName,
	}, nil
}

// Publish sends an event to the Kinesis stream.
func (k *KinesisProducer) Publish(ctx context.Context, event models.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		PartitionKey: aws.String(event.UserID), // Use UserID for ordering guarantees
		StreamName:   aws.String(k.streamName),
	})

	return err
}
