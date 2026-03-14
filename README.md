# ReactiveBridge

ReactiveBridge is a robust backend infrastructure project designed to bridge the gap between high-velocity client data and scalable AWS cloud storage. Built with Golang and the Gin Gonic framework, it demonstrates a production-ready approach to event-driven architecture.

## System Architecture

**The project implements a decoupled, three-tier data pipeline:**

 1. Ingestion Tier (Producer): A containerized Gin API written in Go. It receives RESTful payloads, validates data integrity, and asynchronously dispatches events to the cloud.

2. Streaming Tier (Buffer): Amazon Kinesis Data Streams acts as the high-throughput backbone, ensuring data persistence and ordered delivery via intelligent Partition Key strategies.

3. Processing Tier (Consumer): An AWS Lambda (Go runtime) triggered by Kinesis shards. It performs:

   - Data transformation and enrichment.

   - PII masking for LGPD/GDPR compliance.

   - Final persistence to Amazon DynamoDB.
  
## Tech Stack & AWS Domains
- Language: Go (utilizing `goroutines`, `context`, and `AWS SDK v2`)
- Web Framework: `net/http`
- Infrastructure: Kinesis, Lambda, DynamoDB, SQS (DLQ), X-Ray, Secrets Manager.
- Deployment: AWS SAM (Serverless Application Model) / Docker.

## Performance & Scaling
- Shard Optimization: ReactiveBridge utilizes source-based partition keys to prevent "hot shards" while maintaining event ordering for specific users/services.
- Graceful Shutdown: The Gin server implements signal handling to ensure all pending Kinesis `PutRecord` calls complete before the process exits.
- Backpressure: The architecture is designed to handle spikes by leveraging Kinesis as a buffer, preventing the downstream Lambda from being overwhelmed.
