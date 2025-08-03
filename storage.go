package llmtracer

import (
	"context"
	"time"
)

type StorageAdapter interface {
	Save(ctx context.Context, request *Request) error

	Get(ctx context.Context, id string) (*Request, error)

	GetByTraceID(ctx context.Context, traceID string) ([]*Request, error)

	Query(ctx context.Context, filter *RequestFilter) ([]*Request, error)

	Aggregate(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error)

	Delete(ctx context.Context, id string) error

	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	Close() error
}
