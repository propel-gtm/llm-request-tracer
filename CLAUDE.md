# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detector
go test -race ./...

# Run specific test
go test -v -run TestContextHelpers ./...
go test -v -run TestGormAdapter ./adapters/...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Development
```bash
# Format all code
go fmt ./...

# Check for common issues
go vet ./...

# Download and tidy dependencies
go mod tidy

# Build the library
go build ./...

# Run example
go run examples/simple.go
```

**Important**: Always run `go fmt ./...` after making any changes to Go code. Also run `go vet ./...` to check for common issues and `go mod tidy` if dependencies change.

## Architecture Overview

This library provides a simple wrapper around AI provider clients with automatic token tracking. The design prioritizes simplicity and ease of integration.

### Core Design

1. **Unified Client** (`client.go`)
   - Single `Client` type with trace methods: `TraceOpenAIRequest`, `TraceAnthropicRequest`, `TraceMistralRequest`, `TraceGoogleRequest`
   - Each method wraps your existing AI client calls with automatic token tracking
   - Uses dependency injection pattern - you pass your client's method to the tracer
   - Automatic token tracking happens transparently in the background

2. **Storage Layer** (`storage.go`, `adapters/`)
   - `StorageAdapter` interface for pluggable backends
   - GORM implementation supporting SQLite, PostgreSQL, MySQL
   - Stores token usage with provider, model, timestamps, and custom dimensions

3. **Data Model** (`types.go`)
   - `Request` struct stores all tracking data
   - No cost calculation - pure token counting
   - Flexible dimensions via `DimensionTag` for custom metadata
   - Supports providers: OpenAI, Anthropic, Google, Mistral

### Key Design Principles

- **Dependency Injection**: Pass your AI client methods directly to the tracer
- **Transparent Tracking**: Token usage is automatically captured from responses
- **Context-based Metadata**: Use context to add user IDs, features, and custom dimensions
- **Flexible Storage**: Pluggable storage backend via `StorageAdapter` interface

### Usage Pattern

```go
// Setup once
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
storage, _ := adapters.NewGormAdapter(db)
client := llmtracer.NewClient(storage)

// Create your AI clients
openaiClient := openai.NewClient("your-key")

// Use anywhere - wrap your client calls with the tracer
ctx := llmtracer.WithUserID(context.Background(), "user-123")
response, err := client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
    Model: "gpt-4",
    Messages: []openai.ChatCompletionMessage{
        {Role: openai.ChatMessageRoleUser, Content: "Hello"},
    },
}, openaiClient.CreateChatCompletion)
```

### Important Notes

- Tracking is transparent - just wrap your existing AI client calls
- No cost calculation - pure token counting only
- Context helpers available for adding metadata: `WithUserID`, `WithFeature`, `WithWorkflow`, `WithDimensions`
- Tracking context is optional but useful for analytics
- All providers use the same pattern: pass your request and client method to the tracer

### Recent Improvements

1. **Circuit Breaker**: Protects AI requests from storage failures
   - Automatically opens after configurable failures
   - AI requests continue working even if storage is down
   - Auto-recovers when storage comes back online

2. **Error Categorization**: Automatically categorizes errors
   - Rate limit, authentication, timeout, network, invalid request, server errors
   - Stored in `error_type` field for better analytics
   - Can filter requests by error type

3. **100% Test Coverage**: Comprehensive unit tests
   - All methods fully tested including edge cases
   - Mock implementations for testing
   - Timing tests for async behavior