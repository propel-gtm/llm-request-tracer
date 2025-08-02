package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	llmtracer "github.com/propel-gtm/llm-request-tracer"
	"github.com/propel-gtm/llm-request-tracer/adapters"
)

func main() {
	// Setup token tracking storage (one time)
	db, err := gorm.Open(sqlite.Open("token_usage.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	storage, err := adapters.NewGormAdapter(db)
	if err != nil {
		log.Fatal("Failed to create storage adapter:", err)
	}

	// Create the unified AI client
	client := llmtracer.NewClient(storage)
	defer client.Close()

	// Configure your API keys
	client.SetOpenAIKey("your-openai-key")
	// Note: Other providers will be implemented later

	// Example 1: Call OpenAI with automatic tracking
	ctx := context.Background()
	ctx = llmtracer.WithUserID(ctx, "user-123")
	ctx = llmtracer.WithFeature(ctx, "geography-quiz")

	response, err := client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			{Role: openai.ChatMessageRoleUser, Content: "What is the capital of France?"},
		},
	})
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else if len(response.Choices) > 0 {
		fmt.Printf("OpenAI Response: %s\n", response.Choices[0].Message.Content)
	}

	// Example 2: Another OpenAI call with different context
	ctx = llmtracer.WithUserID(context.Background(), "user-123")
	ctx = llmtracer.WithFeature(ctx, "math-help")

	response, err = client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
			{Role: openai.ChatMessageRoleUser, Content: "What is 2+2?"},
		},
	})
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else if len(response.Choices) > 0 {
		fmt.Printf("OpenAI Response: %s\n", response.Choices[0].Message.Content)
	}

	// Get token usage statistics
	stats, err := client.GetTokenStats(context.Background(), nil)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Println("\n=== Token Usage Stats ===")
	for model, stat := range stats {
		fmt.Printf("%s:\n", model)
		fmt.Printf("  Total Requests: %d\n", stat.TotalRequests)
		fmt.Printf("  Input Tokens: %d\n", stat.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", stat.OutputTokens)
		if stat.ErrorCount > 0 {
			fmt.Printf("  Errors: %d\n", stat.ErrorCount)
		}
		fmt.Println()
	}
}

// Example of how to integrate with your existing code
type YourAIService struct {
	client *llmtracer.Client
}

func (s *YourAIService) ProcessRequest(userID, message string) (string, error) {
	// Create context with tracking dimensions
	ctx := context.Background()
	ctx = llmtracer.WithUserID(ctx, userID)
	ctx = llmtracer.WithDimensions(ctx, map[string]interface{}{
		"endpoint": "/api/chat",
		"version":  "v1",
	})

	// Just call the wrapped method - tracking happens automatically!
	response, err := s.client.TraceOpenAIRequest(ctx, openai.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant for our application."},
			{Role: openai.ChatMessageRoleUser, Content: message},
		},
	})

	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return response.Choices[0].Message.Content, nil
}
