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

func main() {
	db, err := gorm.Open(sqlite.Open("llm_requests.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	gormAdapter, err := adapters.NewGormAdapter(db)
	if err != nil {
		log.Fatal("Failed to create GORM adapter:", err)
	}

	tracker := llmtracer.NewTracker(gormAdapter)
	defer tracker.Close()

	ctx := context.Background()

	fmt.Println("=== Example 1: Basic request tracking ===")
	err = tracker.TrackRequest(ctx, llmtracer.RequestOptions{
		TraceID:  "trace-123",
		Provider: llmtracer.ProviderOpenAI,
		Model:    "gpt-4",
		Dimensions: map[string]interface{}{
			"user_id":    "user-456",
			"endpoint":   "/chat/completions",
			"usage_type": "production",
		},
	}, 100, 150, 0.002, 1200*time.Millisecond, 200, nil)

	if err != nil {
		log.Printf("Error tracking request: %v", err)
	} else {
		fmt.Println("✓ Successfully tracked OpenAI request")
	}

	fmt.Println("\n=== Example 2: Using TrackedRequest for timing ===")
	trackedReq := tracker.StartRequest("trace-456", llmtracer.ProviderAnthropic, "claude-3-sonnet")
	
	time.Sleep(800 * time.Millisecond)
	
	err = trackedReq.FinishWithDimensions(ctx, 80, 120, 0.0015, 200, map[string]interface{}{
		"user_id":    "user-789",
		"usage_type": "development",
	}, nil)

	if err != nil {
		log.Printf("Error finishing tracked request: %v", err)
	} else {
		fmt.Println("✓ Successfully tracked Anthropic request with timing")
	}

	fmt.Println("\n=== Example 3: Tracking an error ===")
	err = tracker.TrackRequest(ctx, llmtracer.RequestOptions{
		TraceID:  "trace-error",
		Provider: llmtracer.ProviderGoogle,
		Model:    "gemini-pro",
		Dimensions: map[string]interface{}{
			"user_id": "user-error",
		},
	}, 50, 0, 0, 500*time.Millisecond, 429, fmt.Errorf("rate limit exceeded"))

	if err != nil {
		log.Printf("Error tracking error request: %v", err)
	} else {
		fmt.Println("✓ Successfully tracked error request")
	}

	fmt.Println("\n=== Example 4: Querying requests ===")
	filter := &llmtracer.RequestFilter{
		Provider: llmtracer.ProviderOpenAI,
		Limit:    10,
	}

	requests, err := tracker.QueryRequests(ctx, filter)
	if err != nil {
		log.Printf("Error querying requests: %v", err)
	} else {
		fmt.Printf("✓ Found %d OpenAI requests\n", len(requests))
		for _, req := range requests {
			fmt.Printf("  - Request %s: %s/%s (%d tokens, $%.4f)\n", 
				req.ID[:8], req.Provider, req.Model, req.TotalTokens, req.Cost)
		}
	}

	fmt.Println("\n=== Example 5: Getting aggregates ===")
	aggregates, err := tracker.GetAggregates(ctx, []string{"provider", "model"}, nil)
	if err != nil {
		log.Printf("Error getting aggregates: %v", err)
	} else {
		fmt.Printf("✓ Generated %d aggregate results\n", len(aggregates))
		for _, agg := range aggregates {
			fmt.Printf("  - %s/%s: %d requests, %d tokens, $%.4f total cost, %.2fms avg latency\n",
				agg.Provider, agg.Model, agg.TotalRequests, agg.TotalTokens, 
				agg.TotalCost, float64(agg.AvgLatency.Nanoseconds())/1e6)
		}
	}

	fmt.Println("\n=== Example 6: Getting requests by trace ID ===")
	traceRequests, err := tracker.GetRequestsByTrace(ctx, "trace-123")
	if err != nil {
		log.Printf("Error getting requests by trace: %v", err)
	} else {
		fmt.Printf("✓ Found %d requests for trace-123\n", len(traceRequests))
	}
}