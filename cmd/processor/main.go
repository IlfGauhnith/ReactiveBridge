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
	"github.com/IlfGauhnith/ReactiveBridge/internal/validator"
)

var dynamoClient *dynamodb.Client
var tableName string

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	tableName = os.Getenv("TABLE_NAME")
	if tableName == "" {
		slog.Error("TABLE_NAME environment variable is not set")
		os.Exit(1)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		slog.Error("unable to load SDK config", "error", err)
		os.Exit(1)
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		rawBody := record.Body

		// 1. PEEK: Identify the schema type without full unmarshal
		// We use an anonymous struct to efficiently grab just the field we need.
		var metadata struct {
			SchemaType string `json:"schema_type"`
		}
		// It only looks for schema_type
		if err := json.Unmarshal([]byte(rawBody), &metadata); err != nil || metadata.SchemaType == "" {
			slog.Error("Invalid event format: missing or empty schema_type",
				"message_id", record.MessageId,
			)
			continue // Skip "Poison Pill"
		}

		// 2. VALIDATE: Run the dynamic schema validation
		// This uses our internal/validator map-cached schemas.
		if err := validator.ValidateEvent(metadata.SchemaType, rawBody); err != nil {
			slog.Error("Schema Validation Failed",
				"schema_type", metadata.SchemaType,
				"message_id", record.MessageId,
				"error", err.Error(),
			)
			// In Phase 5, we will send these to the "Bad Rows" queue instead of just 'continue'
			continue
		}

		// 3. UNMARSHAL: Now we know the data is safe and valid
		var payload models.EventEnvelope
		_ = json.Unmarshal([]byte(rawBody), &payload)

		slog.Info("Event Validated and Processing",
			"event_id", payload.ID,
			"schema_type", payload.SchemaType,
		)

		// 4. PERSIST: Save to DynamoDB
		_, err := dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: &tableName,
			Item: map[string]types.AttributeValue{
				"ID":      &types.AttributeValueMemberS{Value: payload.ID},
				"Payload": &types.AttributeValueMemberS{Value: rawBody},
			},
		})

		if err != nil {
			slog.Error("Failed to write to DynamoDB",
				"event_id", payload.ID,
				"error", err,
			)
			return fmt.Errorf("failed to write to DynamoDB for ID %s: %w", payload.ID, err)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
