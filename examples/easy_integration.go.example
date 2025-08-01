package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	llmtracer "github.com/yourusername/llm-request-tracer"
	"github.com/yourusername/llm-request-tracer/adapters"
	"github.com/yourusername/llm-request-tracer/wrappers"
)

// Your existing AIConfig interface
type AIConfig interface {
	CallOpenAI(contextMessage, userMessage string) (string, error)
}

// Your existing implementation, but now with easy tracking
type AIConfigImpl struct {
	OpenAIModel  string
	OpenAIClient *wrappers.TrackedOpenAIClient // Just change this type!
}

// Your exact same function, with minimal changes
func (c *AIConfigImpl) CallOpenAI(contextMessage, userMessage string) (string, error) {
	startTime := time.Now()
	requestID := "" // Your existing code

	// Create context with tracking info (optional but recommended)
	ctx := context.Background()
	ctx = llmtracer.WithNewTraceID(ctx)
	ctx = llmtracer.WithUserID(ctx, "user-123") // Add user ID if available
	ctx = llmtracer.WithWorkflow(ctx, "chat-completion")

	// Your existing logging code...
	inputTokens := estimateTokenCount(contextMessage) + estimateTokenCount(userMessage)
	fmt.Printf("Calling OpenAI API, model: %s, estimated tokens: %d\n", c.OpenAIModel, inputTokens)

	// Same OpenAI call - tracking happens automatically!
	message, err := c.OpenAIClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.OpenAIModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: contextMessage},
			{Role: openai.ChatMessageRoleUser, Content: userMessage},
		},
		Temperature: 0,
		TopP:        1,
	})

	callDuration := time.Since(startTime)

	if err != nil {
		fmt.Printf("OpenAI API call failed: %v, duration: %v\n", err, callDuration)
		return "", err
	}

	// Your existing token counting logic...
	var actualInputTokens, actualOutputTokens int
	if message.Usage.PromptTokens > 0 {
		actualInputTokens = message.Usage.PromptTokens
	} else {
		actualInputTokens = inputTokens
	}

	if message.Usage.CompletionTokens > 0 {
		actualOutputTokens = message.Usage.CompletionTokens
	} else {
		actualOutputTokens = estimateTokenCount(message.Choices[0].Message.Content)
	}

	fmt.Printf("OpenAI call completed - model: %s, input: %d, output: %d, duration: %v\n",
		c.OpenAIModel, actualInputTokens, actualOutputTokens, callDuration)

	// Your existing structured logging...
	logModelCall("CHAT", 0, requestID, "openai", c.OpenAIModel, actualInputTokens, actualOutputTokens, callDuration)

	return message.Choices[0].Message.Content, nil
}

// Setup function - call this once at startup
func setupTracker() *llmtracer.SimpleTracker {
	// Setup your database (SQLite for example)
	db, err := gorm.Open(sqlite.Open("requests.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create storage adapter
	storage, err := adapters.NewGormAdapter(db)
	if err != nil {
		log.Fatal("Failed to create storage adapter:", err)
	}

	// Create simple tracker
	return llmtracer.NewSimpleTracker(storage)
}

func main() {
	// Setup tracking (do this once at app startup)
	tracker := setupTracker()
	defer tracker.Close()

	// Create your AI config with tracked client
	aiConfig := &AIConfigImpl{
		OpenAIModel:  "gpt-3.5-turbo",
		OpenAIClient: wrappers.NewTrackedOpenAIClient("your-api-key", tracker),
	}

	// Use it exactly like before - tracking happens automatically!
	response, err := aiConfig.CallOpenAI("You are a helpful assistant.", "What is the capital of France?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n", response)

	// Bonus: Get usage stats with one line
	ctx := context.Background()
	stats, err := tracker.GetUsageStats(ctx, llmtracer.ProviderOpenAI, nil)
	if err == nil {
		fmt.Printf("Usage Stats - Requests: %d, Tokens: %d, Cost: $%.6f\n",
			stats.TotalRequests, stats.TotalTokens, stats.TotalCost)
	}
}

// Alternative: Manual tracking if you prefer more control
func manualTrackingExample() {
	tracker := setupTracker()
	defer tracker.Close()

	// Your existing function mostly unchanged
	startTime := time.Now()
	
	// ... make your API call here ...
	
	duration := time.Since(startTime)
	
	// Just add this one line at the end:
	err := tracker.TrackOpenAI("gpt-3.5-turbo", 100, 150, duration, nil)
	if err != nil {
		log.Printf("Failed to track request: %v", err)
	}
}

// Even simpler: One-liner tracking
func oneLineTrackingExample() {
	tracker := setupTracker()
	defer tracker.Close()

	// After your API call, just do:
	tracker.TrackCall(llmtracer.ProviderOpenAI, "gpt-3.5-turbo", 100, 150, 1200*time.Millisecond, nil)
}

// Dummy functions to match your code
func estimateTokenCount(text string) int {
	return len(text) / 4 // Simple estimation
}

func logModelCall(workflow string, prNumber int, requestID, provider, model string, inputTokens, outputTokens int, duration time.Duration) {
	fmt.Printf("LOG: %s - %s/%s - %d/%d tokens - %v\n", workflow, provider, model, inputTokens, outputTokens, duration)
}