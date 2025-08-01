package llmtracer

import (
	"context"
	"time"
)

// SimpleTracker provides easy-to-use methods for tracking LLM requests
type SimpleTracker struct {
	tracker *Tracker
}

// NewSimpleTracker creates a simplified tracker interface
func NewSimpleTracker(storage StorageAdapter) *SimpleTracker {
	return &SimpleTracker{
		tracker: NewTracker(storage),
	}
}

// TrackCall is a one-liner to track any LLM API call
func (st *SimpleTracker) TrackCall(provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, err error) error {
	return st.TrackCallWithContext(context.Background(), provider, model, inputTokens, outputTokens, duration, err, nil)
}

// TrackCallWithContext tracks an LLM call with additional context
func (st *SimpleTracker) TrackCallWithContext(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, err error, metadata map[string]interface{}) error {
	cost := CalculateCost(provider, model, inputTokens, outputTokens)
	statusCode := 200
	if err != nil {
		statusCode = 500
	}

	opts := RequestOptions{
		TraceID:    GetTraceIDFromContext(ctx),
		Provider:   provider,
		Model:      model,
		Dimensions: metadata,
	}

	return st.tracker.TrackRequest(ctx, opts, inputTokens, outputTokens, cost, duration, statusCode, err)
}

// TrackOpenAI is a convenience method specifically for OpenAI calls
func (st *SimpleTracker) TrackOpenAI(model string, inputTokens, outputTokens int, duration time.Duration, err error) error {
	return st.TrackCall(ProviderOpenAI, model, inputTokens, outputTokens, duration, err)
}

// TrackAnthropic is a convenience method specifically for Anthropic calls
func (st *SimpleTracker) TrackAnthropic(model string, inputTokens, outputTokens int, duration time.Duration, err error) error {
	return st.TrackCall(ProviderAnthropic, model, inputTokens, outputTokens, duration, err)
}

// TrackWithDimensions tracks a call with custom dimensions (user_id, feature, etc.)
func (st *SimpleTracker) TrackWithDimensions(provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, dimensions map[string]interface{}, err error) error {
	return st.TrackCallWithContext(context.Background(), provider, model, inputTokens, outputTokens, duration, err, dimensions)
}

// GetUsageStats returns simple usage statistics
func (st *SimpleTracker) GetUsageStats(ctx context.Context, provider Provider, since *time.Time) (*UsageStats, error) {
	filter := &RequestFilter{
		Provider: provider,
	}
	if since != nil {
		filter.StartTime = since
	}

	aggregates, err := st.tracker.GetAggregates(ctx, []string{"provider", "model"}, filter)
	if err != nil {
		return nil, err
	}

	var stats UsageStats
	for _, agg := range aggregates {
		stats.TotalRequests += agg.TotalRequests
		stats.TotalTokens += agg.TotalTokens
		stats.TotalCost += agg.TotalCost
		if agg.AvgLatency > stats.MaxLatency {
			stats.MaxLatency = agg.AvgLatency
		}
		stats.ErrorCount += agg.ErrorCount
	}

	if stats.TotalRequests > 0 {
		stats.ErrorRate = float64(stats.ErrorCount) / float64(stats.TotalRequests) * 100
	}

	return &stats, nil
}

// Close closes the underlying tracker
func (st *SimpleTracker) Close() error {
	return st.tracker.Close()
}

// GetTracker returns the underlying tracker for advanced usage
func (st *SimpleTracker) GetTracker() *Tracker {
	return st.tracker
}

type UsageStats struct {
	TotalRequests int64         `json:"total_requests"`
	TotalTokens   int64         `json:"total_tokens"`
	TotalCost     float64       `json:"total_cost"`
	ErrorCount    int64         `json:"error_count"`
	ErrorRate     float64       `json:"error_rate"`
	MaxLatency    time.Duration `json:"max_latency"`
}

// CalculateCost estimates the cost for a given provider/model combination
func CalculateCost(provider Provider, model string, inputTokens, outputTokens int) float64 {
	switch provider {
	case ProviderOpenAI:
		return calculateOpenAICost(model, inputTokens, outputTokens)
	case ProviderAnthropic:
		return calculateAnthropicCost(model, inputTokens, outputTokens)
	case ProviderGoogle:
		return calculateGoogleCost(model, inputTokens, outputTokens)
	default:
		return 0
	}
}

func calculateOpenAICost(model string, inputTokens, outputTokens int) float64 {
	rates := map[string][2]float64{
		"gpt-4":              {0.00003, 0.00006},
		"gpt-4-turbo":        {0.00001, 0.00003},
		"gpt-3.5-turbo":      {0.0000015, 0.000002},
		"gpt-3.5-turbo-16k":  {0.000003, 0.000004},
		"text-davinci-003":   {0.00002, 0.00002},
		"text-curie-001":     {0.000002, 0.000002},
		"text-babbage-001":   {0.0000005, 0.0000005},
		"text-ada-001":       {0.0000004, 0.0000004},
	}

	if rate, exists := rates[model]; exists {
		return float64(inputTokens)*rate[0] + float64(outputTokens)*rate[1]
	}
	return 0
}

func calculateAnthropicCost(model string, inputTokens, outputTokens int) float64 {
	rates := map[string][2]float64{
		"claude-3-opus":   {0.000015, 0.000075},
		"claude-3-sonnet": {0.000003, 0.000015},
		"claude-3-haiku":  {0.00000025, 0.00000125},
		"claude-2":        {0.000008, 0.000024},
		"claude-instant":  {0.000001, 0.000003},
	}

	if rate, exists := rates[model]; exists {
		return float64(inputTokens)*rate[0] + float64(outputTokens)*rate[1]
	}
	return 0
}

func calculateGoogleCost(model string, inputTokens, outputTokens int) float64 {
	rates := map[string][2]float64{
		"gemini-pro":    {0.0000005, 0.0000015},
		"gemini-pro-1": {0.0000005, 0.0000015},
		"palm-2":        {0.0000005, 0.0000015},
	}

	if rate, exists := rates[model]; exists {
		return float64(inputTokens)*rate[0] + float64(outputTokens)*rate[1]
	}
	return 0
}