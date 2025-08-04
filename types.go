package llmtracer

import (
	"time"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGoogle    Provider = "google"
	ProviderMistral   Provider = "mistral"
)

type Request struct {
	ID           string         `json:"id" gorm:"primaryKey"`
	TraceID      string         `json:"trace_id" gorm:"index"`
	Provider     Provider       `json:"provider" gorm:"index"`
	Model        string         `json:"model" gorm:"index"`
	InputTokens  int            `json:"input_tokens"`
	OutputTokens int            `json:"output_tokens"`
	Latency      time.Duration  `json:"latency"`
	StatusCode   int            `json:"status_code"`
	Error        string         `json:"error,omitempty"`
	Dimensions   []DimensionTag `json:"dimensions,omitempty" gorm:"many2many:request_dimensions;"`
	RequestedAt  time.Time      `json:"requested_at" gorm:"index"`
	RespondedAt  time.Time      `json:"responded_at"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

type RequestFilter struct {
	TraceID    string
	Provider   Provider
	Model      string
	StartTime  *time.Time
	EndTime    *time.Time
	Dimensions []DimensionTag
	MinTokens  *int
	MaxTokens  *int
	HasError   *bool
	Limit      int
	Offset     int
	OrderBy    string
	OrderDesc  bool
}

type AggregateResult struct {
	Provider      Provider       `json:"provider"`
	Model         string         `json:"model"`
	TotalRequests int64          `json:"total_requests"`
	TotalTokens   int64          `json:"total_tokens"`
	AvgLatency    time.Duration  `json:"avg_latency"`
	ErrorCount    int64          `json:"error_count"`
	Dimensions    []DimensionTag `json:"dimensions"`
}

type DimensionTag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex:idx_key_value;size:100"`
	Value     string    `json:"value" gorm:"uniqueIndex:idx_key_value;size:255"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
