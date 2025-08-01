# LLM Request Tracer

Simple Go library that wraps AI provider clients (OpenAI, Anthropic, Mistral, Google) and automatically tracks token usage in the background.

## 🎯 Quick Start

```go
// Setup once
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
storage, _ := adapters.NewGormAdapter(db)
client := llmtracer.NewClient(storage)

// Configure API keys
client.SetOpenAIKey("your-key")
client.SetAnthropicKey("your-key")

// Make calls - token tracking happens automatically!
response, _ := client.CallOpenAI("gpt-4", "You are helpful.", "Hello!", nil)
response, _ := client.CallAnthropic("claude-3-haiku", "You are helpful.", "Hi!", nil)

// Get token stats
stats, _ := client.GetTokenStats(context.Background(), nil)
```

## Features

- **Simple unified interface**: Just `CallOpenAI`, `CallAnthropic`, `CallMistral`, `CallGoogle`
- **Automatic token tracking**: No manual tracking needed
- **Built-in clients**: Uses official SDKs under the hood
- **Flexible storage**: SQLite, PostgreSQL, MySQL via GORM
- **Optional tracking context**: Add user IDs, features, or any metadata

## Installation

```bash
go get github.com/yourusername/llm-request-tracer
```

## Usage

### Basic Usage

```go
package main

import (
    "log"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    
    llmtracer "github.com/yourusername/llm-request-tracer"
    "github.com/yourusername/llm-request-tracer/adapters"
)

func main() {
    // Setup storage
    db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
    storage, _ := adapters.NewGormAdapter(db)
    
    // Create client
    client := llmtracer.NewClient(storage)
    defer client.Close()
    
    // Configure providers
    client.SetOpenAIKey("your-openai-key")
    client.SetAnthropicKey("your-anthropic-key")
    
    // Make calls
    response, err := client.CallOpenAI(
        "gpt-3.5-turbo",
        "You are a helpful assistant.",
        "What is the capital of France?",
        nil, // optional tracking context
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response)
}
```

### With Tracking Context

```go
// Add metadata for better tracking
response, err := client.CallOpenAI(
    "gpt-4",
    systemMessage,
    userMessage,
    map[string]interface{}{
        "user_id": "user-123",
        "feature": "chat",
        "session": "sess-456",
    },
)
```

### Integration with Existing Code

```go
type YourService struct {
    aiClient *llmtracer.Client
}

func (s *YourService) ProcessUserRequest(userID, message string) (string, error) {
    // Your existing logic...
    
    // Just replace your OpenAI call with this:
    return s.aiClient.CallOpenAI(
        "gpt-4",
        "You are a helpful assistant.",
        message,
        map[string]interface{}{"user_id": userID},
    )
}
```

## Supported Providers

- **OpenAI**: GPT-4, GPT-3.5-turbo, etc.
- **Anthropic**: Claude 3 Opus, Sonnet, Haiku
- **Mistral**: Mistral Large, Medium, Small
- **Google**: Gemini Pro, Gemini Pro Vision

## Token Statistics

```go
// Get usage stats
stats, err := client.GetTokenStats(context.Background(), nil)

// Get stats since a specific time
since := time.Now().Add(-24 * time.Hour)
stats, err := client.GetTokenStats(context.Background(), &since)

// Stats include:
// - Total requests per model
// - Input/output/total tokens
// - Error counts
```

## Storage Adapters

The library uses GORM for storage, supporting:

- SQLite (great for development/small apps)
- PostgreSQL (recommended for production)
- MySQL/MariaDB

```go
// SQLite
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})

// PostgreSQL
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// MySQL
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
```

## Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run specific tests
go test -v -run TestClient ./...
```

## License

MIT License