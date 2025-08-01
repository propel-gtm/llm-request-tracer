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
go test -v -run TestClient ./...
go test -v -run TestGormAdapter ./adapters/...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Development
```bash
# Format all code
go fmt ./...

# Download and tidy dependencies
go mod tidy

# Build the library
go build ./...

# Run example
go run examples/simple.go
```

## Architecture Overview

This library provides a simple wrapper around AI provider clients with automatic token tracking. The design prioritizes simplicity and ease of integration.

### Core Design

1. **Unified Client** (`client.go`)
   - Single `Client` type with simple methods: `CallOpenAI`, `CallAnthropic`, `CallMistral`, `CallGoogle`
   - Each method takes: model, system message, user message, optional tracking context
   - Automatic token tracking happens transparently in the background
   - Uses official SDKs internally (anthropic-sdk-go, go-openai, mistral-go, generative-ai-go)

2. **Storage Layer** (`storage.go`, `adapters/`)
   - `StorageAdapter` interface for pluggable backends
   - GORM implementation supporting SQLite, PostgreSQL, MySQL
   - Stores token usage with provider, model, timestamps, and custom dimensions

3. **Data Model** (`types.go`)
   - `Request` struct stores all tracking data
   - No cost calculation - pure token counting
   - Flexible dimensions map for custom metadata

### Key Design Principles

- **Hidden complexity**: Tracking logic is completely hidden from users
- **Simple interface**: Just call methods like `CallOpenAI()` - no manual tracking
- **Zero configuration**: Works out of the box, just set API keys
- **Flexible metadata**: Optional tracking context for user IDs, features, etc.

### Usage Pattern

```go
// Setup once
client := llmtracer.NewClient(storage)
client.SetOpenAIKey("key")

// Use anywhere - tracking is automatic
response, err := client.CallOpenAI("gpt-4", system, user, context)
```

### Important Notes

- The user wants a simple interface that hides all tracking complexity
- No cost calculation needed - just token counting
- Methods should mirror calling the underlying AI clients directly
- Tracking context is optional but useful for analytics