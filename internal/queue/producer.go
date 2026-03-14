package queue

import (
	"context"
	"fmt"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
)

// KinesisAPI defines the interface for the Kinesis client.
type KinesisAPI interface {
	PutRecord(ctx context.Context, params *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error)
}

// Producer defines the interface for publishing events.
type Producer interface {
	Publish(ctx context.Context, event models.EventEnvelope) error
}

// KinesisProducer handles publishing to AWS Kinesis.
type KinesisProducer struct {
	client     KinesisAPI
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
func (k *KinesisProducer) Publish(ctx context.Context, event models.EventEnvelope) error {
	_, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         event.Data,
		PartitionKey: aws.String(event.Source),
		StreamName:   aws.String(k.streamName),
	})

	if err != nil {
		return fmt.Errorf("failed to put record: %w", err)
	}

	return nil
}
