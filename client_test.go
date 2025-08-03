package llmtracer

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	client := NewClient(mockStorage)

	t.Run("TrackingContext", func(t *testing.T) {
		// This test verifies that tracking context is properly stored
		ctx := map[string]interface{}{
			"user_id": "test-user",
			"feature": "test-feature",
			"custom":  "value",
		}

		// Simulate a request being tracked
		err := client.trackRequest(
			context.Background(),
			ProviderOpenAI,
			"gpt-4",
			100,
			50,
			500*time.Millisecond,
			nil,
			ctx,
		)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(mockStorage.requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(mockStorage.requests))
		}

		req := mockStorage.requests[0]
		if req.Provider != ProviderOpenAI {
			t.Errorf("Expected provider %s, got %s", ProviderOpenAI, req.Provider)
		}
		if req.Model != "gpt-4" {
			t.Errorf("Expected model gpt-4, got %s", req.Model)
		}
		if req.InputTokens != 100 {
			t.Errorf("Expected 100 input tokens, got %d", req.InputTokens)
		}
		if req.OutputTokens != 50 {
			t.Errorf("Expected 50 output tokens, got %d", req.OutputTokens)
		}

		// Check tracking context - need to check DimensionTag slice
		hasDimension := func(dims []DimensionTag, key, value string) bool {
			for _, dim := range dims {
				if dim.Key == key && dim.Value == value {
					return true
				}
			}
			return false
		}

		if !hasDimension(req.Dimensions, "user_id", "test-user") {
			t.Errorf("Expected dimension user_id=test-user not found")
		}
		if !hasDimension(req.Dimensions, "feature", "test-feature") {
			t.Errorf("Expected dimension feature=test-feature not found")
		}
		if !hasDimension(req.Dimensions, "custom", "value") {
			t.Errorf("Expected dimension custom=value not found")
		}
	})

	t.Run("GetTokenStats", func(t *testing.T) {
		// Add more test data
		client.trackRequest(context.Background(), ProviderOpenAI, "gpt-4", 150, 200, 600*time.Millisecond, nil, nil)
		client.trackRequest(context.Background(), ProviderOpenAI, "gpt-3.5-turbo", 50, 100, 300*time.Millisecond, nil, nil)
		client.trackRequest(context.Background(), ProviderAnthropic, "claude-3-haiku", 200, 300, 700*time.Millisecond, nil, nil)

		stats, err := client.GetTokenStats(context.Background(), nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check OpenAI GPT-4 stats
		gpt4Stats := stats["openai/gpt-4"]
		if gpt4Stats == nil {
			t.Error("Expected stats for openai/gpt-4")
		} else {
			if gpt4Stats.TotalRequests != 2 {
				t.Errorf("Expected 2 requests for gpt-4, got %d", gpt4Stats.TotalRequests)
			}
			if gpt4Stats.InputTokens != 250 { // 100 + 150
				t.Errorf("Expected 250 input tokens for gpt-4, got %d", gpt4Stats.InputTokens)
			}
			if gpt4Stats.OutputTokens != 250 { // 50 + 200
				t.Errorf("Expected 250 output tokens for gpt-4, got %d", gpt4Stats.OutputTokens)
			}
		}

		// Check we have all models
		if len(stats) != 3 {
			t.Errorf("Expected 3 different models in stats, got %d", len(stats))
		}
	})

	t.Run("ErrorTracking", func(t *testing.T) {
		// Track a failed request
		err := client.trackRequest(
			context.Background(),
			ProviderGoogle,
			"gemini-pro",
			0,
			0,
			100*time.Millisecond,
			fmt.Errorf("API rate limit exceeded"),
			nil,
		)

		if err != nil {
			t.Errorf("Expected no error tracking failed request, got %v", err)
		}

		lastReq := mockStorage.requests[len(mockStorage.requests)-1]
		if lastReq.StatusCode != 500 {
			t.Errorf("Expected status code 500 for error, got %d", lastReq.StatusCode)
		}
		if lastReq.Error != "API rate limit exceeded" {
			t.Errorf("Expected error message, got %s", lastReq.Error)
		}
	})
}
