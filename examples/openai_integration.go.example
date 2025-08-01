package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	llmtracer "github.com/yourusername/llm-request-tracer"
	"github.com/yourusername/llm-request-tracer/adapters"
)

func simulateOpenAICall(tracker *llmtracer.Tracker, traceID string) error {
	tracked := tracker.StartRequest(traceID, llmtracer.ProviderOpenAI, "gpt-4")
	
	time.Sleep(850 * time.Millisecond)
	
	inputTokens := 120
	outputTokens := 180
	cost := calculateOpenAICost(inputTokens, outputTokens, "gpt-4")
	
	dimensions := map[string]interface{}{
		"user_id":    "user-12345",
		"endpoint":   "/v1/chat/completions",
		"feature":    "chat-completion",
		"region":     "us-east-1",
		"usage_type": "production",
	}
	
	return tracked.FinishWithDimensions(context.Background(), inputTokens, outputTokens, cost, 200, dimensions, nil)
}

func simulateAnthropicCall(tracker *llmtracer.Tracker, traceID string) error {
	tracked := tracker.StartRequest(traceID, llmtracer.ProviderAnthropic, "claude-3-sonnet")
	
	time.Sleep(650 * time.Millisecond)
	
	inputTokens := 95
	outputTokens := 140
	cost := calculateAnthropicCost(inputTokens, outputTokens, "claude-3-sonnet")
	
	dimensions := map[string]interface{}{
		"user_id":    "user-67890",
		"endpoint":   "/v1/messages",
		"feature":    "document-analysis",
		"region":     "us-west-2",
		"usage_type": "development",
	}
	
	return tracked.FinishWithDimensions(context.Background(), inputTokens, outputTokens, cost, 200, dimensions, nil)
}

func calculateOpenAICost(inputTokens, outputTokens int, model string) float64 {
	switch model {
	case "gpt-4":
		return float64(inputTokens)*0.00003 + float64(outputTokens)*0.00006
	case "gpt-3.5-turbo":
		return float64(inputTokens)*0.0000015 + float64(outputTokens)*0.000002
	default:
		return 0
	}
}

func calculateAnthropicCost(inputTokens, outputTokens int, model string) float64 {
	switch model {
	case "claude-3-sonnet":
		return float64(inputTokens)*0.000003 + float64(outputTokens)*0.000015
	case "claude-3-haiku":
		return float64(inputTokens)*0.00000025 + float64(outputTokens)*0.00000125
	default:
		return 0
	}
}

func main() {
	db, err := gorm.Open(sqlite.Open("integration_example.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	storage, err := adapters.NewGormAdapter(db)
	if err != nil {
		log.Fatal("Failed to create storage adapter:", err)
	}

	tracker := llmtracer.NewTracker(storage)
	defer tracker.Close()

	fmt.Println("=== Simulating LLM API Calls ===")

	for i := 0; i < 5; i++ {
		traceID := fmt.Sprintf("conversation-%d", i+1)
		
		if err := simulateOpenAICall(tracker, traceID); err != nil {
			log.Printf("OpenAI call failed: %v", err)
		} else {
			fmt.Printf("✓ Tracked OpenAI call for %s\n", traceID)
		}
		
		if err := simulateAnthropicCall(tracker, traceID); err != nil {
			log.Printf("Anthropic call failed: %v", err)
		} else {
			fmt.Printf("✓ Tracked Anthropic call for %s\n", traceID)
		}
	}

	fmt.Println("\n=== Usage Analytics ===")

	ctx := context.Background()
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	filter := &llmtracer.RequestFilter{
		StartTime: &yesterday,
		EndTime:   &now,
		Limit:     100,
	}

	aggregates, err := tracker.GetAggregates(ctx, []string{"provider", "model"}, filter)
	if err != nil {
		log.Printf("Error getting aggregates: %v", err)
		return
	}

	fmt.Printf("Provider/Model Usage Summary:\n")
	var totalCost float64
	var totalRequests int64
	var totalTokens int64

	for _, agg := range aggregates {
		fmt.Printf("  %s/%s:\n", agg.Provider, agg.Model)
		fmt.Printf("    Requests: %d\n", agg.TotalRequests)
		fmt.Printf("    Tokens: %d\n", agg.TotalTokens)
		fmt.Printf("    Cost: $%.6f\n", agg.TotalCost)
		fmt.Printf("    Avg Latency: %.2fms\n", float64(agg.AvgLatency.Nanoseconds())/1e6)
		fmt.Printf("    Error Rate: %.2f%%\n", float64(agg.ErrorCount)/float64(agg.TotalRequests)*100)
		fmt.Println()

		totalCost += agg.TotalCost
		totalRequests += agg.TotalRequests
		totalTokens += agg.TotalTokens
	}

	fmt.Printf("Overall Summary:\n")
	fmt.Printf("  Total Requests: %d\n", totalRequests)
	fmt.Printf("  Total Tokens: %d\n", totalTokens)
	fmt.Printf("  Total Cost: $%.6f\n", totalCost)

	fmt.Println("\n=== Top Users by Usage ===")
	
	userFilter := &llmtracer.RequestFilter{
		StartTime: &yesterday,
		EndTime:   &now,
		Limit:     50,
		OrderBy:   "total_tokens",
		OrderDesc: true,
	}

	requests, err := tracker.QueryRequests(ctx, userFilter)
	if err != nil {
		log.Printf("Error querying requests: %v", err)
		return
	}

	userUsage := make(map[string]struct {
		requests int
		tokens   int
		cost     float64
	})

	for _, req := range requests {
		if userID, ok := req.Dimensions["user_id"].(string); ok {
			usage := userUsage[userID]
			usage.requests++
			usage.tokens += req.TotalTokens
			usage.cost += req.Cost
			userUsage[userID] = usage
		}
	}

	for userID, usage := range userUsage {
		fmt.Printf("  %s: %d requests, %d tokens, $%.6f\n", 
			userID, usage.requests, usage.tokens, usage.cost)
	}
}