package llmtracer

import (
	"context"
	"testing"
)

func TestContextHelpers(t *testing.T) {
	t.Run("WithTraceID", func(t *testing.T) {
		ctx := context.Background()
		traceID := "test-trace-123"

		ctx = WithTraceID(ctx, traceID)
		retrieved := GetTraceIDFromContext(ctx)

		if retrieved != traceID {
			t.Errorf("Expected trace ID %s, got %s", traceID, retrieved)
		}
	})

	t.Run("WithNewTraceID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithNewTraceID(ctx)

		traceID := GetTraceIDFromContext(ctx)
		if traceID == "" {
			t.Error("Expected non-empty trace ID")
		}
	})

	t.Run("GetTraceIDFromContext with nil context", func(t *testing.T) {
		traceID := GetTraceIDFromContext(nil)
		if traceID == "" {
			t.Error("Expected generated trace ID for nil context")
		}
	})

	t.Run("WithUserID", func(t *testing.T) {
		ctx := context.Background()
		userID := "user-456"

		ctx = WithUserID(ctx, userID)
		retrieved := GetUserIDFromContext(ctx)

		if retrieved != userID {
			t.Errorf("Expected user ID %s, got %s", userID, retrieved)
		}
	})

	t.Run("GetDimensionsFromContext", func(t *testing.T) {
		ctx := context.Background()

		// Add various context values
		ctx = WithUserID(ctx, "user-789")
		ctx = WithWorkflow(ctx, "test-workflow")
		ctx = WithFeature(ctx, "test-feature")
		ctx = WithDimensions(ctx, map[string]interface{}{
			"custom_key": "custom_value",
			"number":     42,
		})

		dimensions := GetDimensionsFromContext(ctx)

		// Check all values are present
		if dimensions["user_id"] != "user-789" {
			t.Errorf("Expected user_id user-789, got %v", dimensions["user_id"])
		}
		if dimensions["workflow"] != "test-workflow" {
			t.Errorf("Expected workflow test-workflow, got %v", dimensions["workflow"])
		}
		if dimensions["feature"] != "test-feature" {
			t.Errorf("Expected feature test-feature, got %v", dimensions["feature"])
		}
		if dimensions["custom_key"] != "custom_value" {
			t.Errorf("Expected custom_key custom_value, got %v", dimensions["custom_key"])
		}
		if dimensions["number"] != 42 {
			t.Errorf("Expected number 42, got %v", dimensions["number"])
		}
	})

	t.Run("GetDimensionsFromContext with nil", func(t *testing.T) {
		dimensions := GetDimensionsFromContext(nil)
		if dimensions == nil {
			t.Error("Expected empty map, got nil")
		}
		if len(dimensions) != 0 {
			t.Errorf("Expected empty dimensions, got %d items", len(dimensions))
		}
	})
}
