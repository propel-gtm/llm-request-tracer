package llmtracer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Tracker struct {
	storage StorageAdapter
}

func NewTracker(storage StorageAdapter) *Tracker {
	return &Tracker{
		storage: storage,
	}
}

type RequestOptions struct {
	TraceID    string
	Provider   Provider
	Model      string
	Dimensions map[string]interface{}
}

func (t *Tracker) TrackRequest(ctx context.Context, opts RequestOptions, inputTokens, outputTokens int, latency time.Duration, statusCode int, err error) error {
	// Convert map dimensions to DimensionTag slice
	var dimensions []DimensionTag
	for key, value := range opts.Dimensions {
		dimensions = append(dimensions, DimensionTag{
			Key:   key,
			Value: fmt.Sprintf("%v", value),
		})
	}
	
	request := &Request{
		ID:           uuid.New().String(),
		TraceID:      opts.TraceID,
		Provider:     opts.Provider,
		Model:        opts.Model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Latency:      latency,
		StatusCode:   statusCode,
		Dimensions:   dimensions,
		RequestedAt:  time.Now().Add(-latency),
		RespondedAt:  time.Now(),
	}

	if err != nil {
		request.Error = err.Error()
	}

	return t.storage.Save(ctx, request)
}

func (t *Tracker) GetRequest(ctx context.Context, id string) (*Request, error) {
	return t.storage.Get(ctx, id)
}

func (t *Tracker) GetRequestsByTrace(ctx context.Context, traceID string) ([]*Request, error) {
	return t.storage.GetByTraceID(ctx, traceID)
}

func (t *Tracker) QueryRequests(ctx context.Context, filter *RequestFilter) ([]*Request, error) {
	return t.storage.Query(ctx, filter)
}

func (t *Tracker) GetAggregates(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error) {
	return t.storage.Aggregate(ctx, groupBy, filter)
}

func (t *Tracker) DeleteRequest(ctx context.Context, id string) error {
	return t.storage.Delete(ctx, id)
}

func (t *Tracker) CleanupOldRequests(ctx context.Context, before time.Time) (int64, error) {
	return t.storage.DeleteOlderThan(ctx, before)
}

func (t *Tracker) Close() error {
	return t.storage.Close()
}

type TrackedRequest struct {
	tracker   *Tracker
	traceID   string
	provider  Provider
	model     string
	startTime time.Time
}

func (t *Tracker) StartRequest(traceID string, provider Provider, model string) *TrackedRequest {
	return &TrackedRequest{
		tracker:   t,
		traceID:   traceID,
		provider:  provider,
		model:     model,
		startTime: time.Now(),
	}
}

func (tr *TrackedRequest) Finish(ctx context.Context, inputTokens, outputTokens int, statusCode int, err error) error {
	latency := time.Since(tr.startTime)

	opts := RequestOptions{
		TraceID:  tr.traceID,
		Provider: tr.provider,
		Model:    tr.model,
	}

	return tr.tracker.TrackRequest(ctx, opts, inputTokens, outputTokens, latency, statusCode, err)
}

func (tr *TrackedRequest) FinishWithDimensions(ctx context.Context, inputTokens, outputTokens int, statusCode int, dimensions map[string]interface{}, err error) error {
	latency := time.Since(tr.startTime)

	opts := RequestOptions{
		TraceID:    tr.traceID,
		Provider:   tr.provider,
		Model:      tr.model,
		Dimensions: dimensions,
	}

	return tr.tracker.TrackRequest(ctx, opts, inputTokens, outputTokens, latency, statusCode, err)
}
