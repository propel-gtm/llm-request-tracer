# Simple Usage Guide

## 🎯 Super Simple Token Tracking (No Cost Calculation)

If you just want to track tokens without cost calculation, use the `TokenTracker`:

```go
// Setup once
db, _ := gorm.Open(sqlite.Open("tokens.db"), &gorm.Config{})
storage, _ := adapters.NewGormAdapter(db)
tracker := llmtracer.NewTokenTracker(storage)

// Track tokens with one line
tracker.Track(llmtracer.ProviderOpenAI, "gpt-4", inputTokens, outputTokens)

// Get token stats
stats, _ := tracker.GetTokenStats(context.Background(), nil)
for model, s := range stats {
    fmt.Printf("%s: %d requests, %d total tokens\n", model, s.TotalRequests, s.TotalTokens)
}
```

## 🚀 Super Easy Integration

### Option 1: Automatic Tracking (Recommended)

**Just change your OpenAI client type and you're done!**

```go
// Before (your current code)
type AIConfigImpl struct {
    OpenAIModel  string
    OpenAIClient *openai.Client  // ← Change this line
}

// After (with automatic tracking)  
type AIConfigImpl struct {
    OpenAIModel  string
    OpenAIClient *wrappers.TrackedOpenAIClient  // ← To this
}
```

**Setup once at startup:**

```go
import (
    llmtracer "github.com/yourusername/llm-request-tracer"
    "github.com/yourusername/llm-request-tracer/adapters"
    "github.com/yourusername/llm-request-tracer/wrappers"
)

// Setup once at app startup
func setupTracking() *wrappers.TrackedOpenAIClient {
    // Database setup
    db, _ := gorm.Open(sqlite.Open("requests.db"), &gorm.Config{})
    storage, _ := adapters.NewGormAdapter(db)
    tracker := llmtracer.NewSimpleTracker(storage)
    
    // Return tracked client
    return wrappers.NewTrackedOpenAIClient("your-api-key", tracker)
}

// Your existing code works exactly the same!
func (c *AIConfigImpl) CallOpenAI(contextMessage, userMessage string) (string, error) {
    // Add context for better tracking (optional)
    ctx := llmtracer.WithUserID(context.Background(), "user-123")
    
    // Same API call - tracking happens automatically!
    return c.OpenAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: c.OpenAIModel,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleSystem, Content: contextMessage},
            {Role: openai.ChatMessageRoleUser, Content: userMessage},
        },
        Temperature: 0,
        TopP:        1,
    })
}
```

**That's it! 🎉 All your OpenAI calls are now automatically tracked.**

---

### Option 2: Manual Tracking (More Control)

**Add just one line after your API call:**

```go
func (c *AIConfigImpl) CallOpenAI(contextMessage, userMessage string) (string, error) {
    startTime := time.Now()
    
    // Your existing OpenAI call...
    message, err := c.OpenAIClient.CreateChatCompletion(context.Background(), req)
    
    duration := time.Since(startTime)
    
    // Add this one line:
    c.tracker.TrackOpenAI(c.OpenAIModel, inputTokens, outputTokens, duration, err)
    
    return message.Choices[0].Message.Content, nil
}
```

---

### Option 3: One-Liner Tracking

**Super minimal - just track what you need:**

```go
// After any API call, just do:
tracker.TrackCall(llmtracer.ProviderOpenAI, "gpt-3.5-turbo", 100, 150, duration, err)
```

---

## 📊 Getting Usage Stats

**One line to get usage statistics:**

```go
stats, _ := tracker.GetUsageStats(ctx, llmtracer.ProviderOpenAI, nil)
fmt.Printf("Requests: %d, Tokens: %d, Cost: $%.4f\n", 
    stats.TotalRequests, stats.TotalTokens, stats.TotalCost)
```

---

## 🏷️ Adding Context (Optional)

**Make your tracking more useful by adding context:**

```go
ctx := context.Background()
ctx = llmtracer.WithUserID(ctx, "user-123")        // Track by user
ctx = llmtracer.WithWorkflow(ctx, "code-review")   // Track by feature
ctx = llmtracer.WithFeature(ctx, "chat")           // Track by component

// Use this context in your API calls
response, err := client.CreateChatCompletion(ctx, req)
```

---

## 🔧 Setup Once, Use Everywhere

**Create a global tracker in your main package:**

```go
package main

import (
    "gorm.io/driver/postgres" // or sqlite, mysql
    "gorm.io/gorm"
    
    llmtracer "github.com/yourusername/llm-request-tracer"
    "github.com/yourusername/llm-request-tracer/adapters"
)

var globalTracker *llmtracer.SimpleTracker

func init() {
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    storage, _ := adapters.NewGormAdapter(db)
    globalTracker = llmtracer.NewSimpleTracker(storage)
}

// Use anywhere in your app
func someFunction() {
    globalTracker.TrackOpenAI("gpt-4", 100, 200, duration, nil)
}
```

---

## ⚡ Key Benefits

- **Zero Code Changes**: Use the wrapped client - your existing code works exactly the same
- **Automatic Cost Calculation**: Built-in pricing for OpenAI, Anthropic, Google models
- **Flexible Storage**: SQLite, PostgreSQL, MySQL - your choice
- **Rich Analytics**: Query by user, model, time range, custom dimensions
- **Production Ready**: Proper error handling, no impact on API performance

---

## 🎯 Perfect for Your Use Case

Based on your code, I recommend **Option 1 (Automatic Tracking)**:

1. Replace `*openai.Client` with `*wrappers.TrackedOpenAIClient`
2. Setup the tracker once at startup
3. Keep all your existing code exactly the same
4. Get automatic tracking, cost calculation, and analytics

**Your current logging and error handling stays exactly the same - the library just adds tracking on top!**