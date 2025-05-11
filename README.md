# Go Message Scheduler

A message scheduling service built in Go that allows for managing and sending messages via webhooks.

## Overview

Go Message Scheduler is an API service that manages message scheduling and delivery. It provides endpoints for retrieving sent messages and controlling a worker pool that processes the message queue.

## Features

- Message scheduling and delivery via webhooks
- Worker pool with pause/resume capabilities
- MongoDB storage for message persistence
- Redis caching for improved performance
- Rate limiting for API requests
- Swagger documentation

## Project Structure

```
go-message-scheduler/
├── client/             # External service clients (webhook)
├── docs/               # Auto-generated Swagger documentation
├── drafts/             # Drafts for API documentation
├── sample/             # Sample data for testing
├── handler.go          # HTTP handlers for message operations
├── main.go             # Application entry point and server setup
├── worker_handler.go   # HTTP handlers for worker pool control
├── worker.go           # Worker implementation for message processing
├── workerpool.go       # Worker pool implementation
├── service.go          # Business logic layer
├── repository.go       # Data access layer
├── cache.go            # Redis cache implementation
├── config.go           # Configuration management
├── ratelimiter.go      # API rate limiting implementation
└── docker-compose.yml  # Docker Compose configuration
```

## API Endpoints

### Messages API

- `GET /sent-messages` - Retrieve all sent messages

### Worker Pool API

- `PUT /worker-pool/state` - Control worker pool state (start/pause)

### API Documentation

- `GET /swagger/*` - Swagger UI for API documentation

## Installation

### Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for development)
- Make

### Running with Docker Compose

The easiest way to run the application is using the provided Makefile:

```
make run
```

This command builds and starts the application along with MongoDB and Redis services using Docker Compose.

### Development

To generate mock files for testing:

```
make generate-mocks
```

To run tests:

```
make tests
```

## Environment Variables

The application uses the following environment variables (configured in docker-compose.yml):

```
PORT=:3000
MONGODB_URI=mongodb://admin:password@mongo:27017
MESSAGES_DB_NAME=messages
MESSAGES_COLLECTION_NAME=messages
REDIS_URI=redis:6379
REDIS_PASSWORD=
REDIS_DB=0
CONFIG_PATH=.config
CONFIG_ENV=dev
```

## Webhook Integration

The service sends messages to a configurable webhook endpoint. The webhook configuration is handled by the webhook client in the `client` package. Messages are delivered to the endpoint with their content and recipient information.

Webhook URL: `https://webhook.site/a4d12c37-21b5-4470-92ad-357329f2b48c`

## Documentation

API documentation is available through Swagger UI at `http://localhost:3000/swagger/` when the server is running. The Swagger documentation is auto-generated and can be found in the `docs` directory.

## Future Improvements

### Handling Stuck Processing Data

- Implement processing_starting time to message data and calculate the time difference between the current time and the processing_starting time.
- If the time difference exceeds a certain threshold, mark the message as "stuck".
- Stuck messages can be sent or not. We cannot duplicated sending messages, so we need to check if the message is already sent before sending it again.
- If messages are not sent, we can mark them as `unsent` and retry sending them later.

### Other Improvements

- Enhanced observability with distributed tracing
- Horizontal scaling of worker nodes
- Message priority queues
- Enhanced security features like OAuth2 integration
- Performance optimization for high-throughput scenarios

## License

This project is licensed under the MIT License - see the LICENSE file for details.