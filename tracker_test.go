package llmtracer

import (
	"context"
	"testing"
	"time"
)

// MockStorageAdapter for testing
type MockStorageAdapter struct {
	requests  []*Request
	saveError error
}

func (m *MockStorageAdapter) Save(ctx context.Context, request *Request) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.requests = append(m.requests, request)
	return nil
}

func (m *MockStorageAdapter) Get(ctx context.Context, id string) (*Request, error) {
	for _, r := range m.requests {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockStorageAdapter) GetByTraceID(ctx context.Context, traceID string) ([]*Request, error) {
	var results []*Request
	for _, r := range m.requests {
		if r.TraceID == traceID {
			results = append(results, r)
		}
	}
	return results, nil
}

func (m *MockStorageAdapter) Query(ctx context.Context, filter *RequestFilter) ([]*Request, error) {
	return m.requests, nil
}

func (m *MockStorageAdapter) Aggregate(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error) {
	return nil, nil
}

func (m *MockStorageAdapter) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockStorageAdapter) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func (m *MockStorageAdapter) Close() error {
	return nil
}

func TestTracker(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	tracker := NewTracker(mockStorage)
	ctx := context.Background()

	t.Run("TrackRequest", func(t *testing.T) {
		opts := RequestOptions{
			TraceID:  "test-trace",
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
			Dimensions: map[string]interface{}{
				"user_id": "test-user",
			},
		}

		err := tracker.TrackRequest(ctx, opts, 100, 150, 1*time.Second, 200, nil)
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
		if req.InputTokens != 100 {
			t.Errorf("Expected 100 input tokens, got %d", req.InputTokens)
		}
	})

	t.Run("StartRequest and Finish", func(t *testing.T) {
		tracked := tracker.StartRequest("tracked-trace", ProviderAnthropic, "claude-3-sonnet")
		time.Sleep(10 * time.Millisecond)

		err := tracked.Finish(ctx, 80, 120, 200, nil)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		requests, err := tracker.GetRequestsByTrace(ctx, "tracked-trace")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(requests))
		}
	})
}
