package llmtracer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	mistral "github.com/gage-technologies/mistral-go"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// Logger interface for logging tracking errors
type Logger interface {
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

// Client provides a unified interface for calling different AI providers with automatic token tracking
type Client struct {
	// Internal tracking
	storage        StorageAdapter
	logger         Logger
	asyncTracking  bool
	circuitBreaker *CircuitBreaker
}

// ClientOption allows configuring the Client
type ClientOption func(*Client)

// WithLogger sets a custom logger for the client
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithAsyncTracking enables asynchronous tracking to reduce latency
func WithAsyncTracking(async bool) ClientOption {
	return func(c *Client) {
		c.asyncTracking = async
	}
}

// WithCircuitBreaker enables circuit breaker pattern for storage operations
func WithCircuitBreaker(maxFailures int, resetTimeout time.Duration) ClientOption {
	return func(c *Client) {
		c.circuitBreaker = NewCircuitBreaker(maxFailures, resetTimeout)
	}
}

// NewClient creates a new AI client with token tracking
func NewClient(storage StorageAdapter, opts ...ClientOption) *Client {
	if storage == nil {
		panic("storage adapter cannot be nil")
	}

	client := &Client{
		storage: storage,
		logger:  zap.NewNop(), // Default to no-op logger
	}

	// Apply options
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}

	return client
}

// OpenAICreateChatCompletionFunc represents the signature of OpenAI's CreateChatCompletion method
type OpenAICreateChatCompletionFunc func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)

// AnthropicMessageNewFunc represents the signature of Anthropic's MessageService.New method
type AnthropicMessageNewFunc func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (res *anthropic.Message, err error)

// MistralChatFunc represents the signature of Mistral's Chat method
type MistralChatFunc func(model string, messages []mistral.ChatMessage, params *mistral.ChatRequestParams) (*mistral.ChatCompletionResponse, error)

// GoogleGenerateContentFunc represents the signature of Google's GenerativeModel.GenerateContent method
type GoogleGenerateContentFunc func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)

// TraceOpenAIRequest wraps OpenAI's CreateChatCompletion and automatically tracks token usage
func (c *Client) TraceOpenAIRequest(ctx context.Context, request openai.ChatCompletionRequest, createChatCompletion OpenAICreateChatCompletionFunc) (openai.ChatCompletionResponse, error) {
	if createChatCompletion == nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("createChatCompletion function cannot be nil")
	}

	startTime := time.Now()

	// Make the actual OpenAI API call using the provided function
	response, err := createChatCompletion(ctx, request)

	duration := time.Since(startTime)

	// Track the request - even if it failed
	inputTokens := 0
	outputTokens := 0
	if err == nil && (response.Usage.PromptTokens > 0 || response.Usage.CompletionTokens > 0) {
		inputTokens = response.Usage.PromptTokens
		outputTokens = response.Usage.CompletionTokens
	}

	// Extract tracking context from context if available
	trackingContext := GetDimensionsFromContext(ctx)

	c.track(ctx, ProviderOpenAI, request.Model, inputTokens, outputTokens, duration, err, trackingContext)

	// Return the original response and error
	return response, err
}

// TraceAnthropicRequest wraps Anthropic's MessageService.New and automatically tracks token usage
func (c *Client) TraceAnthropicRequest(ctx context.Context, params anthropic.MessageNewParams, messageNew AnthropicMessageNewFunc) (*anthropic.Message, error) {
	if messageNew == nil {
		return nil, fmt.Errorf("messageNew function cannot be nil")
	}

	startTime := time.Now()

	// Make the actual Anthropic API call using the provided function
	response, err := messageNew(ctx, params)

	duration := time.Since(startTime)

	// Track the request - even if it failed
	inputTokens := 0
	outputTokens := 0
	if err == nil {
		inputTokens = int(response.Usage.InputTokens)
		outputTokens = int(response.Usage.OutputTokens)
	}

	// Extract tracking context from context if available
	trackingContext := GetDimensionsFromContext(ctx)

	c.track(ctx, ProviderAnthropic, string(params.Model), inputTokens, outputTokens, duration, err, trackingContext)

	// Return the original response and error
	return response, err
}

// TraceMistralRequest wraps Mistral's Chat method and automatically tracks token usage
func (c *Client) TraceMistralRequest(ctx context.Context, model string, messages []mistral.ChatMessage, params *mistral.ChatRequestParams, chat MistralChatFunc) (*mistral.ChatCompletionResponse, error) {
	if chat == nil {
		return nil, fmt.Errorf("chat function cannot be nil")
	}
	if model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}

	startTime := time.Now()

	// Make the actual Mistral API call using the provided function
	response, err := chat(model, messages, params)

	duration := time.Since(startTime)

	// Track the request - even if it failed
	inputTokens := 0
	outputTokens := 0
	if err == nil {
		inputTokens = response.Usage.PromptTokens
		outputTokens = response.Usage.CompletionTokens
	}

	// Extract tracking context from context if available
	trackingContext := GetDimensionsFromContext(ctx)

	c.track(ctx, ProviderMistral, model, inputTokens, outputTokens, duration, err, trackingContext)

	// Return the original response and error
	return response, err
}

// TraceGoogleRequest wraps Google's GenerativeModel.GenerateContent method and automatically tracks token usage
func (c *Client) TraceGoogleRequest(ctx context.Context, model string, parts []genai.Part, generateContent GoogleGenerateContentFunc) (*genai.GenerateContentResponse, error) {
	if generateContent == nil {
		return nil, fmt.Errorf("generateContent function cannot be nil")
	}
	if model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}

	startTime := time.Now()

	// Make the actual Google API call using the provided function
	response, err := generateContent(ctx, parts...)

	duration := time.Since(startTime)

	// Track the request - even if it failed
	inputTokens := 0
	outputTokens := 0
	if err == nil && response.UsageMetadata != nil {
		inputTokens = int(response.UsageMetadata.PromptTokenCount)
		outputTokens = int(response.UsageMetadata.CandidatesTokenCount)
	}

	// Extract tracking context from context if available
	trackingContext := GetDimensionsFromContext(ctx)

	c.track(ctx, ProviderGoogle, model, inputTokens, outputTokens, duration, err, trackingContext)

	// Return the original response and error
	return response, err
}

// track handles request tracking, either synchronously or asynchronously
func (c *Client) track(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, apiErr error, trackingContext map[string]interface{}) {
	if c.asyncTracking {
		// Track asynchronously to avoid blocking the API response
		go func() {
			// Create a background context to avoid cancellation issues
			bgCtx := context.Background()
			c.doTrack(bgCtx, provider, model, inputTokens, outputTokens, duration, apiErr, trackingContext)
		}()
	} else {
		// Track synchronously
		c.doTrack(ctx, provider, model, inputTokens, outputTokens, duration, apiErr, trackingContext)
	}
}

// doTrack handles the actual tracking with error logging
func (c *Client) doTrack(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, apiErr error, trackingContext map[string]interface{}) {
	trackErr := c.trackRequest(ctx, provider, model, inputTokens, outputTokens, duration, apiErr, trackingContext)
	if trackErr != nil {
		// Log but don't fail the request
		providerStr := string(provider)
		c.logger.Error("Failed to track request",
			zap.Error(trackErr),
			zap.String("provider", providerStr),
			zap.String("model", model),
			zap.Int("input_tokens", inputTokens),
			zap.Int("output_tokens", outputTokens),
		)
	}
}

// trackRequest internally tracks the token usage
func (c *Client) trackRequest(ctx context.Context, provider Provider, model string, inputTokens, outputTokens int, duration time.Duration, err error, trackingContext map[string]interface{}) error {
	// Validate token counts
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
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

	// Categorize the error if present
	if request.Error != "" {
		request.ErrorType = CategorizeError(errors.New(request.Error))
	}

	// Use circuit breaker if enabled
	if c.circuitBreaker != nil {
		return c.circuitBreaker.Call(func() error {
			return c.storage.Save(ctx, request)
		})
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
