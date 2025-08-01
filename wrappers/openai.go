package wrappers

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"

	llmtracer "github.com/yourusername/llm-request-tracer"
)

// TrackedOpenAIClient wraps the OpenAI client to automatically track requests
type TrackedOpenAIClient struct {
	client  *openai.Client
	tracker *llmtracer.SimpleTracker
}

// NewTrackedOpenAIClient creates a new tracked OpenAI client
func NewTrackedOpenAIClient(apiKey string, tracker *llmtracer.SimpleTracker) *TrackedOpenAIClient {
	return &TrackedOpenAIClient{
		client:  openai.NewClient(apiKey),
		tracker: tracker,
	}
}

// CreateChatCompletion wraps the OpenAI chat completion call with automatic tracking
func (c *TrackedOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	startTime := time.Now()
	
	response, err := c.client.CreateChatCompletion(ctx, req)
	duration := time.Since(startTime)
	
	// Extract token counts
	var inputTokens, outputTokens int
	if response.Usage.PromptTokens > 0 {
		inputTokens = response.Usage.PromptTokens
	}
	if response.Usage.CompletionTokens > 0 {
		outputTokens = response.Usage.CompletionTokens
	}
	
	// Get dimensions from context
	dimensions := llmtracer.GetDimensionsFromContext(ctx)
	
	// Track the call
	trackErr := c.tracker.TrackWithDimensions(
		llmtracer.ProviderOpenAI,
		req.Model,
		inputTokens,
		outputTokens,
		duration,
		dimensions,
		err,
	)
	
	if trackErr != nil {
		// Log tracking error but don't fail the original request
		// You might want to log this error based on your logging setup
		_ = trackErr
	}
	
	return response, err
}

// CreateChatCompletionStream wraps the streaming chat completion with tracking
func (c *TrackedOpenAIClient) CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error) {
	startTime := time.Now()
	
	stream, err := c.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		duration := time.Since(startTime)
		
		// Track failed request
		dimensions := llmtracer.GetDimensionsFromContext(ctx)
		trackErr := c.tracker.TrackWithDimensions(
			llmtracer.ProviderOpenAI,
			req.Model,
			0, 0, // No token counts available for failed request
			duration,
			dimensions,
			err,
		)
		_ = trackErr
		
		return nil, err
	}
	
	// For now, return the original stream
	// TODO: Implement stream tracking wrapper
	return stream, nil
}