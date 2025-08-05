package llmtracer

import (
	"errors"
	"strings"
	"time"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGoogle    Provider = "google"
	ProviderMistral   Provider = "mistral"
)

// ErrorType represents the category of error that occurred
type ErrorType string

const (
	// ErrorTypeNone indicates no error occurred
	ErrorTypeNone ErrorType = ""
	// ErrorTypeNetwork indicates a network-related error
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeRateLimit indicates the request was rate limited
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeAuthentication indicates an authentication/authorization error
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeInvalidRequest indicates the request was malformed
	ErrorTypeInvalidRequest ErrorType = "invalid_request"
	// ErrorTypeTimeout indicates the request timed out
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeServerError indicates a server-side error
	ErrorTypeServerError ErrorType = "server_error"
	// ErrorTypeUnknown indicates an unknown error type
	ErrorTypeUnknown ErrorType = "unknown"
)

// Circuit breaker errors
var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
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
	ErrorType    ErrorType      `json:"error_type,omitempty" gorm:"index"`
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
	ErrorType  ErrorType
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

// CategorizeError analyzes an error message and returns the appropriate ErrorType
func CategorizeError(err error) ErrorType {
	if err == nil {
		return ErrorTypeNone
	}

	errStr := strings.ToLower(err.Error())

	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") {
		return ErrorTypeRateLimit
	}

	// Check for authentication errors
	if strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "api key") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "forbidden") {
		return ErrorTypeAuthentication
	}

	// Check for network errors (before timeout to handle "dial tcp: connection timeout" correctly)
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "dns") ||
		strings.Contains(errStr, "no such host") {
		return ErrorTypeNetwork
	}

	// Check for timeout errors (after network to avoid false positives)
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context canceled") {
		// Special case: 504 Gateway Timeout is a server error
		if strings.Contains(errStr, "504") {
			return ErrorTypeServerError
		}
		return ErrorTypeTimeout
	}

	// Check for invalid request errors
	if strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "bad request") ||
		strings.Contains(errStr, "400") ||
		strings.Contains(errStr, "malformed") {
		return ErrorTypeInvalidRequest
	}

	// Check for server errors
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "server error") ||
		strings.Contains(errStr, "internal error") {
		return ErrorTypeServerError
	}

	return ErrorTypeUnknown
}
