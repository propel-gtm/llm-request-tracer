package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	llmtracer "github.com/propel-gtm/llm-request-tracer"
	"gorm.io/gorm"
)

type GormAdapter struct {
	db *gorm.DB
}

func NewGormAdapter(db *gorm.DB) (*GormAdapter, error) {
	// Migrate both Request and DimensionTag tables
	if err := db.AutoMigrate(&llmtracer.DimensionTag{}, &llmtracer.Request{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &GormAdapter{
		db: db,
	}, nil
}

func (a *GormAdapter) Save(ctx context.Context, request *llmtracer.Request) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, find or create dimension tags
		var processedDimensions []llmtracer.DimensionTag
		for _, dim := range request.Dimensions {
			var existingTag llmtracer.DimensionTag
			// Try to find existing tag with same key-value pair
			err := tx.Where("key = ? AND value = ?", dim.Key, dim.Value).First(&existingTag).Error
			if err == gorm.ErrRecordNotFound {
				// Create new tag
				newTag := llmtracer.DimensionTag{
					Key:   dim.Key,
					Value: dim.Value,
				}
				if err := tx.Create(&newTag).Error; err != nil {
					return err
				}
				processedDimensions = append(processedDimensions, newTag)
			} else if err != nil {
				return err
			} else {
				processedDimensions = append(processedDimensions, existingTag)
			}
		}

		// Replace dimensions with processed ones (with IDs)
		request.Dimensions = processedDimensions

		// Save the request with associations
		return tx.Create(request).Error
	})
}

func (a *GormAdapter) Get(ctx context.Context, id string) (*llmtracer.Request, error) {
	var request llmtracer.Request
	if err := a.db.WithContext(ctx).Preload("Dimensions").First(&request, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &request, nil
}

func (a *GormAdapter) GetByTraceID(ctx context.Context, traceID string) ([]*llmtracer.Request, error) {
	var requests []*llmtracer.Request
	if err := a.db.WithContext(ctx).Preload("Dimensions").Where("trace_id = ?", traceID).Find(&requests).Error; err != nil {
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
		query = query.Where("(input_tokens + output_tokens) >= ?", *filter.MinTokens)
	}

	if filter.MaxTokens != nil {
		query = query.Where("(input_tokens + output_tokens) <= ?", *filter.MaxTokens)
	}

	if filter.HasError != nil {
		if *filter.HasError {
			query = query.Where("error IS NOT NULL AND error != ''")
		} else {
			query = query.Where("(error IS NULL OR error = '')")
		}
	}

	if len(filter.Dimensions) > 0 {
		// Join with dimension tags for filtering
		for _, dim := range filter.Dimensions {
			query = query.Joins("JOIN request_dimensions rd ON rd.request_id = requests.id").
				Joins("JOIN dimension_tags dt ON dt.id = rd.dimension_tag_id").
				Where("dt.key = ? AND dt.value = ?", dim.Key, dim.Value)
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
	if err := query.Preload("Dimensions").Find(&requests).Error; err != nil {
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
		"SUM(input_tokens + output_tokens) as total_tokens",
		"0 as total_cost",
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
			Dimensions:    []llmtracer.DimensionTag{},
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
