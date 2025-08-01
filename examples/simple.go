package main

import (
	"context"
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	llmtracer "github.com/yourusername/llm-request-tracer"
	"github.com/yourusername/llm-request-tracer/adapters"
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

	// Example 1: Call OpenAI - it's just one line!
	response, err := client.CallOpenAI(
		"gpt-3.5-turbo",
		"You are a helpful assistant.",
		"What is the capital of France?",
		map[string]interface{}{
			"user_id": "user-123",
			"feature": "geography-quiz",
		},
	)
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else {
		fmt.Printf("OpenAI Response: %s\n", response)
	}

	// Example 2: Another OpenAI call with different context
	response, err = client.CallOpenAI(
		"gpt-4",
		"You are a helpful assistant.",
		"What is 2+2?",
		map[string]interface{}{
			"user_id": "user-123",
			"feature": "math-help",
		},
	)
	if err != nil {
		log.Printf("OpenAI error: %v", err)
	} else {
		fmt.Printf("OpenAI Response: %s\n", response)
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
		fmt.Printf("  Total Tokens: %d\n", stat.TotalTokens)
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
	// Just call the appropriate method - tracking happens automatically!
	return s.client.CallOpenAI(
		"gpt-4",
		"You are a helpful assistant for our application.",
		message,
		map[string]interface{}{
			"user_id":  userID,
			"endpoint": "/api/chat",
			"version":  "v1",
		},
	)
}