package llmtracer

import (
	"context"
	"time"
)

// TokenTracker provides simplified token tracking without cost calculation
type TokenTracker struct {
	tracker *Tracker
}

// NewTokenTracker creates a simplified token tracker
func NewTokenTracker(storage StorageAdapter) *TokenTracker {
	return &TokenTracker{
		tracker: NewTracker(storage),
	}
}

// Track records token usage for a model
func (tt *TokenTracker) Track(provider Provider, model string, inputTokens, outputTokens int) error {
	return tt.TrackWithContext(context.Background(), provider, model, inputTokens, outputTokens, 0, nil)
}

// TrackWithContext records token usage with context and timing
func (tt *TokenTracker) TrackWithContext(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, err error) error {
	statusCode := 200
	if err != nil {
		statusCode = 500
	}

	opts := RequestOptions{
		TraceID:    GetTraceIDFromContext(ctx),
		Provider:   provider,
		Model:      model,
		Dimensions: GetDimensionsFromContext(ctx),
	}

	// Cost is always 0 since you don't need it
	return tt.tracker.TrackRequest(ctx, opts, inputTokens, outputTokens, 0, duration, statusCode, err)
}

// GetTokenStats returns token usage statistics by model
func (tt *TokenTracker) GetTokenStats(ctx context.Context, since *time.Time) (map[string]*TokenStats, error) {
	filter := &RequestFilter{}
	if since != nil {
		filter.StartTime = since
	}

	requests, err := tt.tracker.QueryRequests(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*TokenStats)
	
	for _, req := range requests {
		key := string(req.Provider) + "/" + req.Model
		if _, exists := stats[key]; !exists {
			stats[key] = &TokenStats{
				Provider: req.Provider,
				Model:    req.Model,
			}
		}
		
		s := stats[key]
		s.TotalRequests++
		s.InputTokens += int64(req.InputTokens)
		s.OutputTokens += int64(req.OutputTokens)
		s.TotalTokens += int64(req.TotalTokens)
		
		if req.Error != "" {
			s.ErrorCount++
		}
	}

	return stats, nil
}

// TokenStats holds token usage statistics for a model
type TokenStats struct {
	Provider      Provider `json:"provider"`
	Model         string   `json:"model"`
	TotalRequests int64    `json:"total_requests"`
	InputTokens   int64    `json:"input_tokens"`
	OutputTokens  int64    `json:"output_tokens"`
	TotalTokens   int64    `json:"total_tokens"`
	ErrorCount    int64    `json:"error_count"`
}

// QuickTrack is a one-liner for the most common use case
func QuickTrack(tracker *TokenTracker, provider Provider, model string, inputTokens, outputTokens int) {
	_ = tracker.Track(provider, model, inputTokens, outputTokens)
}