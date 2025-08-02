package llmtracer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// Client provides a unified interface for calling different AI providers with automatic token tracking
type Client struct {
	// API clients - starting with OpenAI only for now
	openAIClient *openai.Client

	// Internal tracking
	storage StorageAdapter
}

// NewClient creates a new AI client with token tracking
func NewClient(storage StorageAdapter) *Client {
	return &Client{
		storage: storage,
	}
}

// SetOpenAIKey configures the OpenAI client
func (c *Client) SetOpenAIKey(apiKey string) {
	c.openAIClient = openai.NewClient(apiKey)
}

// TraceOpenAIRequest wraps OpenAI's CreateChatCompletion and automatically tracks token usage
func (c *Client) TraceOpenAIRequest(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	if c.openAIClient == nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("OpenAI client not configured. Call SetOpenAIKey first")
	}

	startTime := time.Now()

	// Make the actual OpenAI API call
	response, err := c.openAIClient.CreateChatCompletion(ctx, request)

	duration := time.Since(startTime)

	// Track the request - even if it failed
	inputTokens := 0
	outputTokens := 0
	if err == nil && response.Usage.PromptTokens > 0 || response.Usage.CompletionTokens > 0 {
		inputTokens = response.Usage.PromptTokens
		outputTokens = response.Usage.CompletionTokens
	}

	// Extract tracking context from context if available
	trackingContext := GetDimensionsFromContext(ctx)

	trackErr := c.trackRequest(ctx, ProviderOpenAI, request.Model,
		inputTokens, outputTokens, duration, err, trackingContext)
	if trackErr != nil {
		// Log but don't fail the request
		fmt.Printf("Failed to track request: %v\n", trackErr)
	}

	// Return the original response and error
	return response, err
}

// Placeholder methods for other providers - can be implemented later
func (c *Client) CallAnthropic(model, systemMessage, userMessage string, trackingContext map[string]interface{}) (string, error) {
	return "", fmt.Errorf("Anthropic integration not implemented yet")
}

func (c *Client) CallMistral(model, systemMessage, userMessage string, trackingContext map[string]interface{}) (string, error) {
	return "", fmt.Errorf("Mistral integration not implemented yet")
}

func (c *Client) CallGoogle(model, systemMessage, userMessage string, trackingContext map[string]interface{}) (string, error) {
	return "", fmt.Errorf("Google integration not implemented yet")
}

// trackRequest internally tracks the token usage
func (c *Client) trackRequest(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, err error, trackingContext map[string]interface{}) error {
	// Convert map dimensions to DimensionTag slice
	var dimensions []DimensionTag
	for key, value := range trackingContext {
		dimensions = append(dimensions, DimensionTag{
			Key:   key,
			Value: fmt.Sprintf("%v", value),
		})
	}

	request := &Request{
		ID:           uuid.New().String(),
		TraceID:      GetTraceIDFromContext(ctx),
		Provider:     provider,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Latency:      duration,
		StatusCode:   200,
		Dimensions:   dimensions,
		RequestedAt:  time.Now().Add(-duration),
		RespondedAt:  time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err != nil {
		request.StatusCode = 500
		request.Error = err.Error()
	}

	return c.storage.Save(ctx, request)
}

// GetTokenStats returns token usage statistics
func (c *Client) GetTokenStats(ctx context.Context, since *time.Time) (map[string]*TokenStats, error) {
	filter := &RequestFilter{}
	if since != nil {
		filter.StartTime = since
	}

	requests, err := c.storage.Query(ctx, filter)
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
	ErrorCount    int64    `json:"error_count"`
}

// Close closes the underlying storage
func (c *Client) Close() error {
	return c.storage.Close()
}
