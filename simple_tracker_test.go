package llmtracer

import (
	"context"
	"testing"
	"time"
)

func TestTokenTracker(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	tracker := NewTokenTracker(mockStorage)

	t.Run("Track", func(t *testing.T) {
		err := tracker.Track(ProviderOpenAI, "gpt-4", 100, 200)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(mockStorage.requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(mockStorage.requests))
		}

		req := mockStorage.requests[0]
		if req.InputTokens != 100 {
			t.Errorf("Expected 100 input tokens, got %d", req.InputTokens)
		}
		if req.OutputTokens != 200 {
			t.Errorf("Expected 200 output tokens, got %d", req.OutputTokens)
		}
		if req.Cost != 0 {
			t.Errorf("Expected 0 cost, got %f", req.Cost)
		}
	})

	t.Run("GetTokenStats", func(t *testing.T) {
		// Add more test data
		tracker.Track(ProviderOpenAI, "gpt-4", 150, 250)
		tracker.Track(ProviderOpenAI, "gpt-3.5-turbo", 50, 100)
		tracker.Track(ProviderAnthropic, "claude-3-sonnet", 200, 300)

		stats, err := tracker.GetTokenStats(context.Background(), nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check GPT-4 stats
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
			if gpt4Stats.TotalTokens != 700 { // 250 + 450
				t.Errorf("Expected 700 total tokens for gpt-4, got %d", gpt4Stats.TotalTokens)
			}
		}
	})

	t.Run("TrackWithContext", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithUserID(ctx, "test-user")
		ctx = WithWorkflow(ctx, "test-workflow")

		err := tracker.TrackWithContext(ctx, ProviderGoogle, "gemini-pro", 80, 120, 500*time.Millisecond, nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		lastReq := mockStorage.requests[len(mockStorage.requests)-1]
		if lastReq.Dimensions["user_id"] != "test-user" {
			t.Errorf("Expected user_id test-user, got %v", lastReq.Dimensions["user_id"])
		}
		if lastReq.Dimensions["workflow"] != "test-workflow" {
			t.Errorf("Expected workflow test-workflow, got %v", lastReq.Dimensions["workflow"])
		}
	})
}

func TestQuickTrack(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	tracker := NewTokenTracker(mockStorage)

	// Test the QuickTrack helper
	QuickTrack(tracker, ProviderOpenAI, "gpt-4", 100, 200)

	if len(mockStorage.requests) != 1 {
		t.Errorf("Expected 1 request, got %d", len(mockStorage.requests))
	}
}