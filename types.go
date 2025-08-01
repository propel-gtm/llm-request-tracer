package llmtracer

import (
	"time"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGoogle    Provider = "google"
	ProviderAWS       Provider = "aws"
	ProviderCustom    Provider = "custom"
)

type Request struct {
	ID             string                 `json:"id" gorm:"primaryKey"`
	TraceID        string                 `json:"trace_id" gorm:"index"`
	Provider       Provider               `json:"provider" gorm:"index"`
	Model          string                 `json:"model" gorm:"index"`
	InputTokens    int                    `json:"input_tokens"`
	OutputTokens   int                    `json:"output_tokens"`
	TotalTokens    int                    `json:"total_tokens"`
	Cost           float64                `json:"cost"`
	Latency        time.Duration          `json:"latency"`
	StatusCode     int                    `json:"status_code"`
	Error          string                 `json:"error,omitempty"`
	Dimensions     map[string]interface{} `json:"dimensions" gorm:"serializer:json"`
	Metadata       map[string]interface{} `json:"metadata" gorm:"serializer:json"`
	RequestedAt    time.Time              `json:"requested_at" gorm:"index"`
	RespondedAt    time.Time              `json:"responded_at"`
	CreatedAt      time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

type RequestFilter struct {
	TraceID      string
	Provider     Provider
	Model        string
	StartTime    *time.Time
	EndTime      *time.Time
	Dimensions   map[string]interface{}
	MinTokens    *int
	MaxTokens    *int
	HasError     *bool
	Limit        int
	Offset       int
	OrderBy      string
	OrderDesc    bool
}

type AggregateResult struct {
	Provider       Provider               `json:"provider"`
	Model          string                 `json:"model"`
	TotalRequests  int64                  `json:"total_requests"`
	TotalTokens    int64                  `json:"total_tokens"`
	TotalCost      float64                `json:"total_cost"`
	AvgLatency     time.Duration          `json:"avg_latency"`
	ErrorCount     int64                  `json:"error_count"`
	Dimensions     map[string]interface{} `json:"dimensions"`
}