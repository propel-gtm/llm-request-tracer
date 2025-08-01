package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	llmtracer "github.com/propel-gtm/llm-request-tracer"
)

type GormAdapter struct {
	db *gorm.DB
}

func NewGormAdapter(db *gorm.DB) (*GormAdapter, error) {
	if err := db.AutoMigrate(&llmtracer.Request{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &GormAdapter{
		db: db,
	}, nil
}

func (a *GormAdapter) Save(ctx context.Context, request *llmtracer.Request) error {
	return a.db.WithContext(ctx).Create(request).Error
}

func (a *GormAdapter) Get(ctx context.Context, id string) (*llmtracer.Request, error) {
	var request llmtracer.Request
	if err := a.db.WithContext(ctx).First(&request, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (a *GormAdapter) GetByTraceID(ctx context.Context, traceID string) ([]*llmtracer.Request, error) {
	var requests []*llmtracer.Request
	if err := a.db.WithContext(ctx).Where("trace_id = ?", traceID).Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (a *GormAdapter) Query(ctx context.Context, filter *llmtracer.RequestFilter) ([]*llmtracer.Request, error) {
	query := a.db.WithContext(ctx)

	if filter.TraceID != "" {
		query = query.Where("trace_id = ?", filter.TraceID)
	}

	if filter.Provider != "" {
		query = query.Where("provider = ?", filter.Provider)
	}

	if filter.Model != "" {
		query = query.Where("model = ?", filter.Model)
	}

	if filter.StartTime != nil {
		query = query.Where("requested_at >= ?", *filter.StartTime)
	}

	if filter.EndTime != nil {
		query = query.Where("requested_at <= ?", *filter.EndTime)
	}

	if filter.MinTokens != nil {
		query = query.Where("total_tokens >= ?", *filter.MinTokens)
	}

	if filter.MaxTokens != nil {
		query = query.Where("total_tokens <= ?", *filter.MaxTokens)
	}

	if filter.HasError != nil {
		if *filter.HasError {
			query = query.Where("error IS NOT NULL AND error != ''")
		} else {
			query = query.Where("(error IS NULL OR error = '')")
		}
	}

	if filter.Dimensions != nil {
		for key, value := range filter.Dimensions {
			query = query.Where("JSON_EXTRACT(dimensions, ?) = ?", "$."+key, value)
		}
	}

	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}

	if filter.OrderDesc {
		orderBy += " DESC"
	} else {
		orderBy += " ASC"
	}
	query = query.Order(orderBy)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var requests []*llmtracer.Request
	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}

	return requests, nil
}

func (a *GormAdapter) Aggregate(ctx context.Context, groupBy []string, filter *llmtracer.RequestFilter) ([]*llmtracer.AggregateResult, error) {
	query := a.db.WithContext(ctx).Model(&llmtracer.Request{})

	if filter != nil {
		if filter.Provider != "" {
			query = query.Where("provider = ?", filter.Provider)
		}

		if filter.Model != "" {
			query = query.Where("model = ?", filter.Model)
		}

		if filter.StartTime != nil {
			query = query.Where("requested_at >= ?", *filter.StartTime)
		}

		if filter.EndTime != nil {
			query = query.Where("requested_at <= ?", *filter.EndTime)
		}
	}

	selectFields := []string{
		"COUNT(*) as total_requests",
		"SUM(total_tokens) as total_tokens",
		"SUM(cost) as total_cost",
		"AVG(latency) as avg_latency",
		"SUM(CASE WHEN error IS NOT NULL AND error != '' THEN 1 ELSE 0 END) as error_count",
	}

	var groupFields []string
	for _, field := range groupBy {
		switch field {
		case "provider", "model":
			selectFields = append(selectFields, field)
			groupFields = append(groupFields, field)
		default:
			continue
		}
	}

	selectClause := strings.Join(selectFields, ", ")
	query = query.Select(selectClause)

	if len(groupFields) > 0 {
		query = query.Group(strings.Join(groupFields, ", "))
	}

	type aggregateRow struct {
		Provider      llmtracer.Provider `json:"provider"`
		Model         string             `json:"model"`
		TotalRequests int64              `json:"total_requests"`
		TotalTokens   int64              `json:"total_tokens"`
		TotalCost     float64            `json:"total_cost"`
		AvgLatency    float64            `json:"avg_latency"`
		ErrorCount    int64              `json:"error_count"`
	}

	var rows []aggregateRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	var results []*llmtracer.AggregateResult
	for _, row := range rows {
		result := &llmtracer.AggregateResult{
			Provider:      row.Provider,
			Model:         row.Model,
			TotalRequests: row.TotalRequests,
			TotalTokens:   row.TotalTokens,
			TotalCost:     row.TotalCost,
			AvgLatency:    time.Duration(int64(row.AvgLatency)),
			ErrorCount:    row.ErrorCount,
			Dimensions:    make(map[string]interface{}),
		}
		results = append(results, result)
	}

	return results, nil
}

func (a *GormAdapter) Delete(ctx context.Context, id string) error {
	result := a.db.WithContext(ctx).Delete(&llmtracer.Request{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (a *GormAdapter) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := a.db.WithContext(ctx).Where("created_at < ?", before).Delete(&llmtracer.Request{})
	return result.RowsAffected, result.Error
}

func (a *GormAdapter) Close() error {
	if db, err := a.db.DB(); err == nil {
		return db.Close()
	}
	return nil
}