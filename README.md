# LLM Request Tracer

A Go library for tracking requests to AI providers like OpenAI, Anthropic, Google, and others. Track token usage, costs, latency, and custom dimensions with pluggable storage adapters.

## 🎯 Quick Start - Simple Token Tracking

Just want to track tokens? Here's the simplest way:

```go
// Setup once
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
storage, _ := adapters.NewGormAdapter(db)
tracker := llmtracer.NewTokenTracker(storage)

// Track tokens with one line after any LLM call
tracker.Track(llmtracer.ProviderOpenAI, "gpt-4", inputTokens, outputTokens)

// Get stats
stats, _ := tracker.GetTokenStats(context.Background(), nil)
```

## Features

- **Multi-provider support**: OpenAI, Anthropic, Google, AWS, and custom providers
- **Comprehensive tracking**: Input/output tokens, costs, latency, status codes, errors
- **Custom dimensions**: Add your own metadata for filtering and aggregation
- **Trace correlation**: Group related requests with trace IDs
- **Pluggable storage**: Interface-based storage with GORM adapter included
- **Aggregation queries**: Get usage statistics by provider, model, time period, etc.
- **Cleanup utilities**: Remove old requests to manage storage

## Installation

```bash
go get github.com/yourusername/llm-request-tracer
```

## Quick Start

```go
package main

import (
    "context"
    "time"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    
    llmtracer "github.com/yourusername/llm-request-tracer"
    "github.com/yourusername/llm-request-tracer/adapters"
)

func main() {
    // Setup database and storage adapter
    db, _ := gorm.Open(sqlite.Open("requests.db"), &gorm.Config{})
    storage, _ := adapters.NewGormAdapter(db)
    tracker := llmtracer.NewTracker(storage)
    defer tracker.Close()
    
    ctx := context.Background()
    
    // Track a request
    err := tracker.TrackRequest(ctx, llmtracer.RequestOptions{
        TraceID:  "trace-123",
        Provider: llmtracer.ProviderOpenAI,
        Model:    "gpt-4",
        Dimensions: map[string]interface{}{
            "user_id": "user-456",
            "endpoint": "/chat/completions",
        },
    }, 100, 150, 0.002, 1200*time.Millisecond, 200, nil)
}
```

## Usage Patterns

### 1. Manual Request Tracking

```go
// Track completed request with all details
err := tracker.TrackRequest(ctx, llmtracer.RequestOptions{
    TraceID:  "trace-abc",
    Provider: llmtracer.ProviderAnthropic,
    Model:    "claude-3-sonnet",
    Dimensions: map[string]interface{}{
        "user_id": "user-123",
        "feature": "chat",
    },
}, inputTokens, outputTokens, cost, latency, statusCode, err)
```

### 2. Timed Request Tracking

```go
// Start timing a request
tracked := tracker.StartRequest("trace-xyz", llmtracer.ProviderOpenAI, "gpt-4")

// ... make your API call ...

// Finish with results (latency calculated automatically)
err := tracked.FinishWithDimensions(ctx, inputTokens, outputTokens, cost, statusCode, 
    map[string]interface{}{
        "user_id": "user-456",
    }, apiErr)
```

### 3. Querying Requests

```go
// Query with filters
filter := &llmtracer.RequestFilter{
    Provider:  llmtracer.ProviderOpenAI,
    StartTime: &startTime,
    EndTime:   &endTime,
    Dimensions: map[string]interface{}{
        "user_id": "user-123",
    },
    Limit: 100,
}

requests, err := tracker.QueryRequests(ctx, filter)
```

### 4. Getting Aggregates

```go
// Get usage stats by provider and model
aggregates, err := tracker.GetAggregates(ctx, []string{"provider", "model"}, filter)

for _, agg := range aggregates {
    fmt.Printf("%s/%s: %d requests, %d tokens, $%.4f cost\n",
        agg.Provider, agg.Model, agg.TotalRequests, agg.TotalTokens, agg.TotalCost)
}
```

## Storage Adapters

### GORM Adapter

Supports SQLite, PostgreSQL, MySQL, and other GORM-compatible databases:

```go
// SQLite
db, _ := gorm.Open(sqlite.Open("requests.db"), &gorm.Config{})

// PostgreSQL  
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// MySQL
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

storage, _ := adapters.NewGormAdapter(db)
```

### Custom Adapters

Implement the `StorageAdapter` interface:

```go
type StorageAdapter interface {
    Save(ctx context.Context, request *Request) error
    Get(ctx context.Context, id string) (*Request, error)
    GetByTraceID(ctx context.Context, traceID string) ([]*Request, error)
    Query(ctx context.Context, filter *RequestFilter) ([]*Request, error)
    Aggregate(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error)
    Delete(ctx context.Context, id string) error
    DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
    Close() error
}
```

## Data Model

### Request

```go
type Request struct {
    ID             string                 // Unique request ID
    TraceID        string                 // Correlation ID for related requests  
    Provider       Provider               // AI provider (openai, anthropic, etc.)
    Model          string                 // Model name (gpt-4, claude-3-sonnet, etc.)
    InputTokens    int                    // Input token count
    OutputTokens   int                    // Output token count  
    TotalTokens    int                    // Total token count
    Cost           float64                // Request cost in USD
    Latency        time.Duration          // Request duration
    StatusCode     int                    // HTTP status code
    Error          string                 // Error message if failed
    Dimensions     map[string]interface{} // Custom metadata
    Metadata       map[string]interface{} // Additional metadata
    RequestedAt    time.Time              // Request start time
    RespondedAt    time.Time              // Request end time
    CreatedAt      time.Time              // Record creation time
    UpdatedAt      time.Time              // Record update time
}
```

### Providers

```go
const (
    ProviderOpenAI    Provider = "openai"
    ProviderAnthropic Provider = "anthropic"  
    ProviderGoogle    Provider = "google"
    ProviderAWS       Provider = "aws"
    ProviderCustom    Provider = "custom"
)
```

## Testing

### Run all tests
```bash
go test ./...
```

### Run tests with verbose output
```bash
go test -v ./...
```

### Run tests with coverage
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Run tests with race detector
```bash
go test -race ./...
```

### Run specific package tests
```bash
# Test adapters only
go test -v ./adapters/...

# Test core functionality
go test -v .
```

### Format code
```bash
go fmt ./...
```

### Check and tidy dependencies
```bash
go mod tidy
```

## License

MIT License