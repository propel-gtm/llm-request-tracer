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
	
	// Wrap the stream to track completion
	return &TrackedChatCompletionStream{
		stream:     stream,
		tracker:    c.tracker,
		model:      req.Model,
		startTime:  startTime,
		ctx:        ctx,
		dimensions: llmtracer.GetDimensionsFromContext(ctx),
	}, nil
}

// TrackedChatCompletionStream wraps the OpenAI stream to track when it completes
type TrackedChatCompletionStream struct {
	stream     *openai.ChatCompletionStream
	tracker    *llmtracer.SimpleTracker
	model      string
	startTime  time.Time
	ctx        context.Context
	dimensions map[string]interface{}
	
	inputTokens  int
	outputTokens int
	tracked      bool
}

// Recv receives the next chat completion chunk and tracks usage when done
func (s *TrackedChatCompletionStream) Recv() (openai.ChatCompletionStreamResponse, error) {
	response, err := s.stream.Recv()
	
	// Track usage information if available
	if response.Usage != nil {
		s.inputTokens = response.Usage.PromptTokens
		s.outputTokens = response.Usage.CompletionTokens
	}
	
	// If stream is done or error, track the request
	if err != nil && !s.tracked {
		s.trackRequest(err)
	}
	
	return response, err
}

// Close closes the stream and ensures tracking is done
func (s *TrackedChatCompletionStream) Close() {
	s.stream.Close()
	if !s.tracked {
		s.trackRequest(nil)
	}
}

func (s *TrackedChatCompletionStream) trackRequest(err error) {
	if s.tracked {
		return
	}
	
	duration := time.Since(s.startTime)
	trackErr := s.tracker.TrackWithDimensions(
		llmtracer.ProviderOpenAI,
		s.model,
		s.inputTokens,
		s.outputTokens,
		duration,
		s.dimensions,
		err,
	)
	_ = trackErr
	s.tracked = true
}