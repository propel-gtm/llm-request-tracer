package adapters

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	llmtracer "github.com/propel-gtm/llm-request-tracer"
)

func TestGormAdapter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	adapter, err := NewGormAdapter(db)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()

	t.Run("Save and Get", func(t *testing.T) {
		request := &llmtracer.Request{
			ID:           "test-id",
			TraceID:      "test-trace",
			Provider:     llmtracer.ProviderOpenAI,
			Model:        "gpt-4",
			InputTokens:  100,
			OutputTokens: 150,
			Latency:      1000 * time.Millisecond,
			StatusCode:   200,
			Dimensions: []llmtracer.DimensionTag{
				{Key: "user_id", Value: "test-user"},
			},
			RequestedAt: time.Now().Add(-1000 * time.Millisecond),
			RespondedAt: time.Now(),
		}

		err := adapter.Save(ctx, request)
		if err != nil {
			t.Errorf("Failed to save request: %v", err)
		}

		retrieved, err := adapter.Get(ctx, "test-id")
		if err != nil {
			t.Errorf("Failed to get request: %v", err)
		}

		if retrieved.Provider != llmtracer.ProviderOpenAI {
			t.Errorf("Expected provider %s, got %s", llmtracer.ProviderOpenAI, retrieved.Provider)
		}
	})
}
