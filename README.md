# LLM Request Tracer

Go library that wraps your existing AI provider client calls (OpenAI, Anthropic, Mistral, Google) with automatic token usage tracking.

## 🎯 Quick Start

```go
// Setup storage once
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
storage, _ := adapters.NewGormAdapter(db)
tracer := llmtracer.NewClient(storage)

// Create your AI clients as usual
openaiClient := openai.NewClient("your-key")

// Wrap your existing calls with the tracer - that's it!
ctx := llmtracer.WithUserID(context.Background(), "user-123")
response, err := tracer.TraceOpenAIRequest(ctx, 
    openai.ChatCompletionRequest{
        Model: "gpt-4",
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleUser, Content: "Hello!"},
        },
    }, 
    openaiClient.CreateChatCompletion,
)

// Get token usage statistics
stats, _ := tracer.GetTokenStats(context.Background(), nil)
```

## Features

- **Transparent tracking**: Wrap your existing AI client calls - no code rewrite needed
- **Dependency injection**: Pass your client methods directly to the tracer
- **Automatic token capture**: Token usage is extracted from provider responses
- **Flexible storage**: SQLite, PostgreSQL, MySQL via GORM adapter
- **Rich metadata**: Track user IDs, features, workflows, and custom dimensions via context
- **Structured logging**: Built-in logging with uber/zap for tracking errors and debugging
- **Async tracking option**: Optional asynchronous tracking to minimize latency impact

## Installation

```bash
go get github.com/propel-gtm/llm-request-tracer
```

## Usage

### Basic Setup

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/sashabaranov/go-openai"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "go.uber.org/zap"
    
    llmtracer "github.com/propel-gtm/llm-request-tracer"
    "github.com/propel-gtm/llm-request-tracer/adapters"
)

func main() {
    // 1. Setup storage
    db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
    storage, _ := adapters.NewGormAdapter(db)
    
    // 2. Create tracer with options
    logger, _ := zap.NewProduction()
    tracer := llmtracer.NewClient(storage, 
        llmtracer.WithLogger(logger),
        llmtracer.WithAsyncTracking(true), // Reduce latency impact
    )
    defer tracer.Close()
    
    // 3. Create your AI client as usual
    openaiClient := openai.NewClient("your-openai-key")
    
    // 4. Use the tracer to wrap your calls
    ctx := context.Background()
    response, err := tracer.TraceOpenAIRequest(ctx,
        openai.ChatCompletionRequest{
            Model: "gpt-3.5-turbo",
            Messages: []openai.ChatCompletionMessage{
                {Role: openai.ChatMessageRoleSystem, Content: "You are helpful."},
                {Role: openai.ChatMessageRoleUser, Content: "What is the capital of France?"},
            },
        },
        openaiClient.CreateChatCompletion, // Pass your client's method
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    // Use response as normal
    if len(response.Choices) > 0 {
        fmt.Println(response.Choices[0].Message.Content)
    }
}
```

### Adding Tracking Metadata

Use context helpers to add metadata for better analytics:

```go
// Add user context
ctx := llmtracer.WithUserID(context.Background(), "user-123")
ctx = llmtracer.WithFeature(ctx, "chat-support")
ctx = llmtracer.WithWorkflow(ctx, "customer-service")

// Add custom dimensions
ctx = llmtracer.WithDimensions(ctx, map[string]interface{}{
    "team": "support",
    "tier": "premium",
    "session_id": "sess-456",
})

// Make the tracked call
response, _ := tracer.TraceOpenAIRequest(ctx, request, client.CreateChatCompletion)
```

### Anthropic Example

```go
import "github.com/anthropics/anthropic-sdk-go"

// Create Anthropic client
anthropicClient := anthropic.NewClient()

// Wrap calls with tracer
ctx := llmtracer.WithUserID(context.Background(), "user-123")
response, err := tracer.TraceAnthropicRequest(ctx,
    anthropic.MessageNewParams{
        Model: anthropic.ModelClaude3_5SonnetLatest,
        MaxTokens: 1000,
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(
                anthropic.NewTextBlock("Write a haiku about coding"),
            ),
        },
    },
    anthropicClient.Messages.New, // Pass the client method
)
```

### Mistral Example

```go
import mistral "github.com/gage-technologies/mistral-go"

// Create Mistral client
mistralClient := mistral.NewMistralClientDefault("your-key")

// Wrap calls with tracer
response, err := tracer.TraceMistralRequest(ctx,
    mistral.ModelMistralLargeLatest,
    []mistral.ChatMessage{
        {Role: mistral.RoleUser, Content: "Hello!"},
    },
    &mistral.ChatRequestParams{MaxTokens: 1000},
    mistralClient.Chat, // Pass the client method
)
```

### Google Generative AI Example

```go
import "github.com/google/generative-ai-go/genai"

// Create Google client
googleClient, _ := genai.NewClient(context.Background())
googleModel := googleClient.GenerativeModel("gemini-1.5-flash")

// Wrap calls with tracer - model name is now a parameter
response, err := tracer.TraceGoogleRequest(context.Background(),
    "gemini-1.5-flash", // Model name as parameter
    []genai.Part{genai.Text("Write a poem about AI")},
    googleModel.GenerateContent, // Pass the model method
)
```

## Token Statistics

Get aggregated token usage statistics:

```go
import "time"

// Get all-time stats
stats, err := tracer.GetTokenStats(context.Background(), nil)

// Get stats since a specific time  
since := time.Now().Add(-24 * time.Hour)
stats, err := tracer.GetTokenStats(context.Background(), &since)

// Stats include per model:
// - Total requests
// - Input tokens
// - Output tokens  
// - Total tokens
// - Error count
for model, stat := range stats {
    fmt.Printf("%s: %d requests, %d total tokens\n", 
        model, stat.TotalRequests, stat.InputTokens + stat.OutputTokens)
}
```

## Storage Adapters

The library uses GORM for flexible storage options:

```go
// SQLite (great for development)
import "gorm.io/driver/sqlite"
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})

// PostgreSQL (recommended for production)
import "gorm.io/driver/postgres"
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// MySQL
import "gorm.io/driver/mysql"
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})

// Create adapter
storage, _ := adapters.NewGormAdapter(db)
```

## Logging Configuration

The library uses structured logging with uber/zap. Configure logging to capture tracking errors:

```go
// Development logger (pretty output)
logger, _ := zap.NewDevelopment()
tracer := llmtracer.NewClient(storage, llmtracer.WithLogger(logger))

// Production logger (JSON output)
logger, _ := zap.NewProduction()
tracer := llmtracer.NewClient(storage, llmtracer.WithLogger(logger))

// Custom logger configuration
config := zap.NewProductionConfig()
config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
logger, _ := config.Build()
tracer := llmtracer.NewClient(storage, llmtracer.WithLogger(logger))

// No logging (default)
tracer := llmtracer.NewClient(storage) // Uses no-op logger
```

Tracking errors are logged but don't stop your AI requests from completing.

## Async Tracking

Enable asynchronous tracking to minimize latency impact on your AI requests:

```go
// Synchronous tracking (default) - waits for database writes
tracer := llmtracer.NewClient(storage)

// Asynchronous tracking - returns immediately, tracks in background
tracer := llmtracer.NewClient(storage, llmtracer.WithAsyncTracking(true))
```

With async tracking:
- AI requests return immediately without waiting for database writes
- Token usage is tracked in background goroutines
- Tracking errors are still logged (if logger is configured)
- Minimal impact on response latency

**When to use async tracking:**
- High-throughput applications where latency is critical
- When database writes are slow or unreliable
- Production environments where response time matters

**When to use sync tracking:**
- Development/testing where you want to ensure tracking completes
- When you need immediate feedback on tracking errors
- Low-traffic applications where latency isn't critical

## Integration with Existing Code

The library is designed to wrap your existing AI client calls with minimal changes:

```go
// BEFORE: Direct OpenAI call
response, err := openaiClient.CreateChatCompletion(ctx, request)

// AFTER: Wrapped with tracking
response, err := tracer.TraceOpenAIRequest(ctx, request, openaiClient.CreateChatCompletion)
```

That's it! Your existing error handling, response processing, and business logic remain unchanged.

## Supported Providers

- **OpenAI**: All chat completion models (GPT-4, GPT-3.5-turbo, etc.)
- **Anthropic**: Claude 3 models (Opus, Sonnet, Haiku)
- **Mistral**: All Mistral models (Large, Medium, Small)
- **Google**: Gemini models (Pro, Flash, etc.)

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Design Philosophy

This library follows a simple principle: **wrap, don't replace**. You keep using your existing AI client libraries and simply wrap the calls with our tracer. This means:

- No vendor lock-in
- Easy to add or remove
- Your existing code patterns remain unchanged
- Full access to provider-specific features

## License

MIT License