package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/IlfGauhnith/ReactiveBridge/internal/models"
)

// Global variables remain in memory across multiple Warm Starts.
// This is how we cache our database connection.
var dynamoClient *dynamodb.Client
var tableName string

// init() is a special Go function that runs automatically before main().
// In AWS Lambda, this runs exactly once during the container's "Cold Start".
// We put slow operations here (reading env vars, network handshakes) 
// so they aren't repeated for every single SQS message.
func init() {
	// 1. Configure Structured Logging
	// Standard logs are just raw text strings. By wrapping os.Stdout in a JSONHandler,
	// every log we emit becomes a structured JSON object. CloudWatch automatically 
	// indexes JSON, allowing us to query logs via SQL-like syntax later.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Load Configuration
	tableName = os.Getenv("TABLE_NAME")
	if tableName == "" {
		// If we are missing critical config, we fail during the Cold Start.
		// os.Exit(1) kills the container immediately before it tries to process events.
		slog.Error("TABLE_NAME environment variable is not set")
		os.Exit(1) 
	}

	// 3. Initialize the AWS Client
	// This performs a network handshake with AWS to verify credentials. 
	// Doing this here saves ~100ms per invocation during Warm Starts.
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		slog.Error("unable to load SDK config", "error", err)
		os.Exit(1)
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
}

// handler is the actual function executed for every batch of SQS messages.
// This function needs to be as fast as possible.
func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	
	// SQS sends messages in batches (we configured BatchSize: 10 in template.yaml).
	// We must loop through them.
	for _, record := range sqsEvent.Records {
		
		// SQS payloads arrive as standard strings in the Body field.
		sqsData := []byte(record.Body)

		var payload models.EventEnvelope
		if err := json.Unmarshal(sqsData, &payload); err != nil {
			// BAD PAYLOAD HANDLING (Poison Pill):
			// If we return an error here, SQS assumes the Lambda failed and will 
			// retry sending this bad message forever. Instead, we log it with the 
			// message_id for debugging, and 'continue' to skip it, effectively deleting it.
			slog.Error("Error unmarshaling data",
				"error", err,
				"raw_data", string(sqsData),
				"message_id", record.MessageId, // Vital for tracing the exact failed SQS message
			)
			continue
		}

		// GOOD PAYLOAD LOGGING:
		// We emit the event ID and type as separate JSON fields. 
		// In CloudWatch, you can now search: { $.event_id = "integration-test-001" }
		slog.Info("Processing Event",
			"event_id", payload.ID,
			"source", payload.Source,
			"event_type", payload.EventType,
		)

		// Persist the data to DynamoDB using our pre-warmed client.
		_, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &tableName,
			Item: map[string]types.AttributeValue{
				"ID":      &types.AttributeValueMemberS{Value: payload.ID},
				"Payload": &types.AttributeValueMemberS{Value: string(sqsData)},
			},
		})

		if err != nil {
			// DATABASE FAILURE HANDLING:
			// If DynamoDB is down or throttles us, we DO want to return the error.
			// This tells SQS: "The payload was fine, but our DB failed. Keep this 
			// message in the queue and try giving it to me again later."
			slog.Error("Failed to write to DynamoDB",
				"event_id", payload.ID,
				"error", err,
			)
			return fmt.Errorf("failed to write to DynamoDB for ID %s: %w", payload.ID, err)
		}
	}

	// Returning nil tells SQS that the entire batch was processed successfully, 
	// and SQS will permanently delete the messages from the queue.
	return nil
}

// main is the entry point, but in AWS Lambda, it just tells the AWS runtime
// which function to use as the handler.
func main() {
	lambda.Start(handler)
}