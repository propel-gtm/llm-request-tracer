package llmtracer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	mistral "github.com/gage-technologies/mistral-go"
	"github.com/google/generative-ai-go/genai"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockStorageAdapter implements StorageAdapter for testing
type MockStorageAdapter struct {
	SaveFunc           func(ctx context.Context, request *Request) error
	GetFunc            func(ctx context.Context, id string) (*Request, error)
	GetByTraceIDFunc   func(ctx context.Context, traceID string) ([]*Request, error)
	QueryFunc          func(ctx context.Context, filter *RequestFilter) ([]*Request, error)
	AggregateFunc      func(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error)
	DeleteFunc         func(ctx context.Context, id string) error
	DeleteOlderThanFunc func(ctx context.Context, before time.Time) (int64, error)
	CloseFunc          func() error

	// Track calls for assertions
	SaveCalls []SaveCall
	QueryCalls []QueryCall
}

type SaveCall struct {
	Ctx     context.Context
	Request *Request
}

type QueryCall struct {
	Ctx    context.Context
	Filter *RequestFilter
}

func (m *MockStorageAdapter) Save(ctx context.Context, request *Request) error {
	m.SaveCalls = append(m.SaveCalls, SaveCall{Ctx: ctx, Request: request})
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, request)
	}
	return nil
}

func (m *MockStorageAdapter) Get(ctx context.Context, id string) (*Request, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockStorageAdapter) GetByTraceID(ctx context.Context, traceID string) ([]*Request, error) {
	if m.GetByTraceIDFunc != nil {
		return m.GetByTraceIDFunc(ctx, traceID)
	}
	return nil, nil
}

func (m *MockStorageAdapter) Query(ctx context.Context, filter *RequestFilter) ([]*Request, error) {
	m.QueryCalls = append(m.QueryCalls, QueryCall{Ctx: ctx, Filter: filter})
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, filter)
	}
	return []*Request{}, nil
}

func (m *MockStorageAdapter) Aggregate(ctx context.Context, groupBy []string, filter *RequestFilter) ([]*AggregateResult, error) {
	if m.AggregateFunc != nil {
		return m.AggregateFunc(ctx, groupBy, filter)
	}
	return nil, errors.New("not implemented")
}

func (m *MockStorageAdapter) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func (m *MockStorageAdapter) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	if m.DeleteOlderThanFunc != nil {
		return m.DeleteOlderThanFunc(ctx, before)
	}
	return 0, errors.New("not implemented")
}

func (m *MockStorageAdapter) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// MockLogger implements Logger for testing
type MockLogger struct {
	ErrorCalls []LogCall
	WarnCalls  []LogCall
	InfoCalls  []LogCall
	DebugCalls []LogCall
}

type LogCall struct {
	Message string
	Fields  []zap.Field
}

func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	m.DebugCalls = append(m.DebugCalls, LogCall{Message: msg, Fields: fields})
}

// Test NewClient
func TestNewClient(t *testing.T) {
	storage := &MockStorageAdapter{}
	client := NewClient(storage)
	
	assert.NotNil(t, client)
	assert.Equal(t, storage, client.storage)
	assert.NotNil(t, client.logger) // Should have default no-op logger
}

// Test NewClient with logger
func TestNewClientWithLogger(t *testing.T) {
	storage := &MockStorageAdapter{}
	logger := &MockLogger{}
	client := NewClient(storage, WithLogger(logger))
	
	assert.NotNil(t, client)
	assert.Equal(t, storage, client.storage)
	assert.Equal(t, logger, client.logger)
	assert.False(t, client.asyncTracking) // Default is sync
}

// Test NewClient with async tracking
func TestNewClientWithAsyncTracking(t *testing.T) {
	storage := &MockStorageAdapter{}
	client := NewClient(storage, WithAsyncTracking(true))
	
	assert.NotNil(t, client)
	assert.Equal(t, storage, client.storage)
	assert.True(t, client.asyncTracking)
}

// Test NewClient with multiple options
func TestNewClientWithMultipleOptions(t *testing.T) {
	storage := &MockStorageAdapter{}
	logger := &MockLogger{}
	client := NewClient(storage, WithLogger(logger), WithAsyncTracking(true))
	
	assert.NotNil(t, client)
	assert.Equal(t, storage, client.storage)
	assert.Equal(t, logger, client.logger)
	assert.True(t, client.asyncTracking)
}

// Test TraceOpenAIRequest
func TestTraceOpenAIRequest(t *testing.T) {
	tests := []struct {
		name           string
		request        openai.ChatCompletionRequest
		response       openai.ChatCompletionResponse
		responseErr    error
		saveErr        error
		expectedSave   bool
		checkRequest   func(t *testing.T, req *Request)
	}{
		{
			name: "successful request with tracking",
			request: openai.ChatCompletionRequest{
				Model: "gpt-3.5-turbo",
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleUser, Content: "Hello"},
				},
			},
			response: openai.ChatCompletionResponse{
				Model: "gpt-3.5-turbo",
				Usage: openai.Usage{
					PromptTokens:     5,
					CompletionTokens: 10,
					TotalTokens:      15,
				},
				Choices: []openai.ChatCompletionChoice{
					{Message: openai.ChatCompletionMessage{Content: "Hi there!"}},
				},
			},
			responseErr:  nil,
			saveErr:      nil,
			expectedSave: true,
			checkRequest: func(t *testing.T, req *Request) {
				assert.Equal(t, ProviderOpenAI, req.Provider)
				assert.Equal(t, "gpt-3.5-turbo", req.Model)
				assert.Equal(t, 5, req.InputTokens)
				assert.Equal(t, 10, req.OutputTokens)
				assert.Equal(t, 200, req.StatusCode)
				assert.Empty(t, req.Error)
				assert.NotEmpty(t, req.ID)
				assert.NotEmpty(t, req.TraceID)
			},
		},
		{
			name: "failed API request still tracks",
			request: openai.ChatCompletionRequest{
				Model: "gpt-4",
			},
			response:     openai.ChatCompletionResponse{},
			responseErr:  errors.New("API error: rate limit exceeded"),
			saveErr:      nil,
			expectedSave: true,
			checkRequest: func(t *testing.T, req *Request) {
				assert.Equal(t, ProviderOpenAI, req.Provider)
				assert.Equal(t, "gpt-4", req.Model)
				assert.Equal(t, 0, req.InputTokens)
				assert.Equal(t, 0, req.OutputTokens)
				assert.Equal(t, 500, req.StatusCode)
				assert.Contains(t, req.Error, "rate limit exceeded")
			},
		},
		{
			name: "tracking failure is logged",
			request: openai.ChatCompletionRequest{
				Model: "gpt-3.5-turbo",
			},
			response: openai.ChatCompletionResponse{
				Model: "gpt-3.5-turbo",
				Usage: openai.Usage{
					PromptTokens:     5,
					CompletionTokens: 10,
				},
			},
			responseErr:  nil,
			saveErr:      errors.New("database connection failed"),
			expectedSave: true,
		},
		{
			name: "context with user metadata",
			request: openai.ChatCompletionRequest{
				Model: "gpt-3.5-turbo",
			},
			response: openai.ChatCompletionResponse{
				Model: "gpt-3.5-turbo",
				Usage: openai.Usage{
					PromptTokens: 5,
				},
			},
			responseErr:  nil,
			saveErr:      nil,
			expectedSave: true,
			checkRequest: func(t *testing.T, req *Request) {
				assert.Len(t, req.Dimensions, 2)
				// Check that dimensions contain user_id and feature
				foundUserID := false
				foundFeature := false
				for _, dim := range req.Dimensions {
					if dim.Key == "user_id" && dim.Value == "test-user-123" {
						foundUserID = true
					}
					if dim.Key == "feature" && dim.Value == "test-feature" {
						foundFeature = true
					}
				}
				assert.True(t, foundUserID, "user_id dimension not found")
				assert.True(t, foundFeature, "feature dimension not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock storage
			mockStorage := &MockStorageAdapter{
				SaveFunc: func(ctx context.Context, request *Request) error {
					return tt.saveErr
				},
			}

			// Setup mock logger for tracking failure test
			var mockLogger *MockLogger
			var client *Client
			if tt.name == "tracking failure is logged" {
				mockLogger = &MockLogger{}
				client = NewClient(mockStorage, WithLogger(mockLogger))
			} else {
				client = NewClient(mockStorage)
			}

			// Setup context with metadata for the last test
			ctx := context.Background()
			if tt.name == "context with user metadata" {
				ctx = WithUserID(ctx, "test-user-123")
				ctx = WithFeature(ctx, "test-feature")
			}

			// Mock OpenAI function
			mockOpenAIFunc := func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
				// Verify the request is passed through correctly
				assert.Equal(t, tt.request.Model, request.Model)
				return tt.response, tt.responseErr
			}

			// Call the method
			response, err := client.TraceOpenAIRequest(ctx, tt.request, mockOpenAIFunc)

			// Verify response and error are passed through
			if tt.responseErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.responseErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.response, response)
			}

			// Verify tracking was attempted
			if tt.expectedSave {
				assert.Len(t, mockStorage.SaveCalls, 1)
				
				if tt.checkRequest != nil && len(mockStorage.SaveCalls) > 0 {
					tt.checkRequest(t, mockStorage.SaveCalls[0].Request)
				}
			}

			// Verify logging for tracking failure test
			if tt.name == "tracking failure is logged" && mockLogger != nil {
				assert.Len(t, mockLogger.ErrorCalls, 1)
				assert.Equal(t, "Failed to track request", mockLogger.ErrorCalls[0].Message)
				
				// Verify error fields contain expected information
				fields := mockLogger.ErrorCalls[0].Fields
				assert.True(t, len(fields) > 0)
				
				// Check that error field is present
				var hasError bool
				for _, field := range fields {
					if field.Key == "error" {
						hasError = true
						break
					}
				}
				assert.True(t, hasError, "Error field should be present in log")
			}
		})
	}
}

// Test async tracking
func TestAsyncTracking(t *testing.T) {
	// Channel to detect when async operation completes
	saveDone := make(chan bool, 1)
	
	mockStorage := &MockStorageAdapter{
		SaveFunc: func(ctx context.Context, request *Request) error {
			saveDone <- true // Signal that save was called
			return nil
		},
	}

	// Create client with async tracking
	client := NewClient(mockStorage, WithAsyncTracking(true))

	// Setup test data
	request := openai.ChatCompletionRequest{
		Model: "gpt-3.5-turbo",
	}

	response := openai.ChatCompletionResponse{
		Model: "gpt-3.5-turbo",
		Usage: openai.Usage{
			PromptTokens:     5,
			CompletionTokens: 10,
		},
	}

	// Mock OpenAI function
	mockOpenAIFunc := func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
		return response, nil
	}

	// Call the method - should return immediately even though tracking is async
	result, err := client.TraceOpenAIRequest(context.Background(), request, mockOpenAIFunc)

	// Verify API call succeeded immediately
	assert.NoError(t, err)
	assert.Equal(t, response, result)

	// Wait for async tracking to complete (with timeout)
	select {
	case <-saveDone:
		// Good, async tracking completed
		assert.Len(t, mockStorage.SaveCalls, 1)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Async tracking did not complete within timeout")
	}
}

// Test sync vs async tracking timing
func TestSyncVsAsyncTracking(t *testing.T) {
	// This test verifies that async tracking doesn't block the API response
	slowSaveCount := 0
	mockStorage := &MockStorageAdapter{
		SaveFunc: func(ctx context.Context, request *Request) error {
			slowSaveCount++
			time.Sleep(50 * time.Millisecond) // Simulate slow storage
			return nil
		},
	}

	tests := []struct {
		name          string
		asyncTracking bool
		maxDuration   time.Duration
	}{
		{
			name:          "sync tracking waits for storage",
			asyncTracking: false,
			maxDuration:   100 * time.Millisecond, // Should take at least 50ms
		},
		{
			name:          "async tracking doesn't wait",
			asyncTracking: true,
			maxDuration:   30 * time.Millisecond, // Should be much faster
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(mockStorage, WithAsyncTracking(tt.asyncTracking))

			request := openai.ChatCompletionRequest{Model: "gpt-3.5-turbo"}
			response := openai.ChatCompletionResponse{
				Model: "gpt-3.5-turbo",
				Usage: openai.Usage{PromptTokens: 5},
			}

			mockFunc := func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
				return response, nil
			}

			start := time.Now()
			result, err := client.TraceOpenAIRequest(context.Background(), request, mockFunc)
			duration := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, response, result)

			if tt.asyncTracking {
				// Async should return quickly
				assert.Less(t, duration, tt.maxDuration, "Async tracking should not block API response")
				
				// Wait a bit for async operation to complete, then verify it happened
				time.Sleep(100 * time.Millisecond)
			} else {
				// Sync should take longer due to slow storage
				assert.Greater(t, duration, 40*time.Millisecond, "Sync tracking should wait for storage")
			}
		})
	}
}

// Test TraceAnthropicRequest
func TestTraceAnthropicRequest(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	client := NewClient(mockStorage)

	// Setup test data
	request := anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5SonnetLatest,
		MaxTokens: 1000,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock("Hello"),
			),
		},
	}

	response := &anthropic.Message{
		Model: anthropic.ModelClaude3_5SonnetLatest,
		Usage: anthropic.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		// Content field is complex, we'll just verify the tracking works
	}

	// Mock Anthropic function
	mockAnthropicFunc := func(ctx context.Context, body anthropic.MessageNewParams, opts ...option.RequestOption) (*anthropic.Message, error) {
		return response, nil
	}

	// Call the method
	ctx := WithTraceID(context.Background(), "test-trace-123")
	result, err := client.TraceAnthropicRequest(ctx, request, mockAnthropicFunc)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, response, result)

	// Verify tracking
	require.Len(t, mockStorage.SaveCalls, 1)
	savedRequest := mockStorage.SaveCalls[0].Request
	
	assert.Equal(t, ProviderAnthropic, savedRequest.Provider)
	assert.Equal(t, string(anthropic.ModelClaude3_5SonnetLatest), savedRequest.Model)
	assert.Equal(t, 10, savedRequest.InputTokens)
	assert.Equal(t, 20, savedRequest.OutputTokens)
	assert.Equal(t, "test-trace-123", savedRequest.TraceID)
	assert.Equal(t, 200, savedRequest.StatusCode)
}

// Test TraceMistralRequest
func TestTraceMistralRequest(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	client := NewClient(mockStorage)

	// Setup test data
	model := mistral.ModelMistralLargeLatest
	messages := []mistral.ChatMessage{
		{Role: mistral.RoleUser, Content: "Hello"},
	}
	params := &mistral.ChatRequestParams{
		MaxTokens: 1000,
	}

	response := &mistral.ChatCompletionResponse{
		Model: model,
		Usage: mistral.UsageInfo{
			PromptTokens:     15,
			CompletionTokens: 25,
			TotalTokens:      40,
		},
		Choices: []mistral.ChatCompletionResponseChoice{
			{
				Index: 0,
				Message: mistral.ChatMessage{
					Role:    mistral.RoleAssistant,
					Content: "Hi!",
				},
				FinishReason: "stop",
			},
		},
	}

	// Mock Mistral function
	mockMistralFunc := func(model string, messages []mistral.ChatMessage, params *mistral.ChatRequestParams) (*mistral.ChatCompletionResponse, error) {
		return response, nil
	}

	// Call the method
	result, err := client.TraceMistralRequest(context.Background(), model, messages, params, mockMistralFunc)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, response, result)

	// Verify tracking
	require.Len(t, mockStorage.SaveCalls, 1)
	savedRequest := mockStorage.SaveCalls[0].Request
	
	assert.Equal(t, ProviderMistral, savedRequest.Provider)
	assert.Equal(t, model, savedRequest.Model)
	assert.Equal(t, 15, savedRequest.InputTokens)
	assert.Equal(t, 25, savedRequest.OutputTokens)
	assert.Equal(t, 200, savedRequest.StatusCode)
}

// Test TraceGoogleRequest
func TestTraceGoogleRequest(t *testing.T) {
	mockStorage := &MockStorageAdapter{}
	client := NewClient(mockStorage)

	// Setup test data
	parts := []genai.Part{
		genai.Text("Hello Google"),
	}

	response := &genai.GenerateContentResponse{
		UsageMetadata: &genai.UsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 15,
			TotalTokenCount:      20,
		},
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []genai.Part{genai.Text("Hello from Google!")},
				},
			},
		},
	}

	// Mock Google function
	mockGoogleFunc := func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
		return response, nil
	}

	// Call the method with model parameter
	result, err := client.TraceGoogleRequest(context.Background(), "gemini-pro", parts, mockGoogleFunc)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, response, result)

	// Verify tracking
	require.Len(t, mockStorage.SaveCalls, 1)
	savedRequest := mockStorage.SaveCalls[0].Request
	
	assert.Equal(t, ProviderGoogle, savedRequest.Provider)
	assert.Equal(t, "gemini-pro", savedRequest.Model) // Now uses the actual model parameter
	assert.Equal(t, 5, savedRequest.InputTokens)
	assert.Equal(t, 15, savedRequest.OutputTokens)
	assert.Equal(t, 200, savedRequest.StatusCode)
}

// Test GetTokenStats
func TestGetTokenStats(t *testing.T) {
	tests := []struct {
		name           string
		since          *time.Time
		mockRequests   []*Request
		expectedStats  map[string]*TokenStats
	}{
		{
			name:  "all time stats",
			since: nil,
			mockRequests: []*Request{
				{
					Provider:     ProviderOpenAI,
					Model:        "gpt-4",
					InputTokens:  100,
					OutputTokens: 200,
				},
				{
					Provider:     ProviderOpenAI,
					Model:        "gpt-4",
					InputTokens:  150,
					OutputTokens: 250,
					Error:        "timeout",
				},
				{
					Provider:     ProviderAnthropic,
					Model:        "claude-3-opus",
					InputTokens:  300,
					OutputTokens: 400,
				},
			},
			expectedStats: map[string]*TokenStats{
				"openai/gpt-4": {
					Provider:      ProviderOpenAI,
					Model:         "gpt-4",
					TotalRequests: 2,
					InputTokens:   250,
					OutputTokens:  450,
					ErrorCount:    1,
				},
				"anthropic/claude-3-opus": {
					Provider:      ProviderAnthropic,
					Model:         "claude-3-opus",
					TotalRequests: 1,
					InputTokens:   300,
					OutputTokens:  400,
					ErrorCount:    0,
				},
			},
		},
		{
			name:  "stats since specific time",
			since: func() *time.Time { t := time.Now().Add(-1 * time.Hour); return &t }(),
			mockRequests: []*Request{
				{
					Provider:     ProviderMistral,
					Model:        "mistral-large",
					InputTokens:  50,
					OutputTokens: 100,
				},
			},
			expectedStats: map[string]*TokenStats{
				"mistral/mistral-large": {
					Provider:      ProviderMistral,
					Model:         "mistral-large",
					TotalRequests: 1,
					InputTokens:   50,
					OutputTokens:  100,
					ErrorCount:    0,
				},
			},
		},
		{
			name:          "no requests",
			since:         nil,
			mockRequests:  []*Request{},
			expectedStats: map[string]*TokenStats{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorageAdapter{
				QueryFunc: func(ctx context.Context, filter *RequestFilter) ([]*Request, error) {
					// Verify filter
					if tt.since != nil {
						assert.Equal(t, tt.since, filter.StartTime)
					}
					return tt.mockRequests, nil
				},
			}

			client := NewClient(mockStorage)
			stats, err := client.GetTokenStats(context.Background(), tt.since)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStats, stats)
		})
	}
}

// Test error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("storage query error", func(t *testing.T) {
		mockStorage := &MockStorageAdapter{
			QueryFunc: func(ctx context.Context, filter *RequestFilter) ([]*Request, error) {
				return nil, errors.New("database connection failed")
			},
		}

		client := NewClient(mockStorage)
		stats, err := client.GetTokenStats(context.Background(), nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection failed")
		assert.Nil(t, stats)
	})

	t.Run("nil usage data handling", func(t *testing.T) {
		mockStorage := &MockStorageAdapter{}
		client := NewClient(mockStorage)

		// Mistral with empty usage
		mistralResponse := &mistral.ChatCompletionResponse{
			Model: "mistral-small",
			Usage: mistral.UsageInfo{}, // empty usage
		}

		mockMistralFunc := func(model string, messages []mistral.ChatMessage, params *mistral.ChatRequestParams) (*mistral.ChatCompletionResponse, error) {
			return mistralResponse, nil
		}

		_, err := client.TraceMistralRequest(
			context.Background(),
			"mistral-small",
			[]mistral.ChatMessage{{Role: mistral.RoleUser, Content: "test"}},
			nil,
			mockMistralFunc,
		)

		assert.NoError(t, err)
		require.Len(t, mockStorage.SaveCalls, 1)
		assert.Equal(t, 0, mockStorage.SaveCalls[0].Request.InputTokens)
		assert.Equal(t, 0, mockStorage.SaveCalls[0].Request.OutputTokens)
	})
}

// Test Close method
func TestClose(t *testing.T) {
	closeCalled := false
	mockStorage := &MockStorageAdapter{
		CloseFunc: func() error {
			closeCalled = true
			return nil
		},
	}

	client := NewClient(mockStorage)
	err := client.Close()

	assert.NoError(t, err)
	assert.True(t, closeCalled)
}

// Test Close with error
func TestCloseError(t *testing.T) {
	mockStorage := &MockStorageAdapter{
		CloseFunc: func() error {
			return errors.New("failed to close connection")
		},
	}

	client := NewClient(mockStorage)
	err := client.Close()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close connection")
}

// Test validation
func TestValidation(t *testing.T) {
	t.Run("NewClient with nil storage panics", func(t *testing.T) {
		assert.Panics(t, func() {
			NewClient(nil)
		})
	})

	t.Run("NewClient with nil option is handled gracefully", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage, nil) // nil option
		assert.NotNil(t, client)
	})

	t.Run("TraceOpenAIRequest with nil function", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		request := openai.ChatCompletionRequest{Model: "gpt-3.5-turbo"}
		_, err := client.TraceOpenAIRequest(context.Background(), request, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("TraceAnthropicRequest with nil function", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		params := anthropic.MessageNewParams{
			Model: anthropic.ModelClaude3_5SonnetLatest,
		}
		_, err := client.TraceAnthropicRequest(context.Background(), params, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("TraceMistralRequest with nil function", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		_, err := client.TraceMistralRequest(context.Background(), "model", nil, nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("TraceMistralRequest with empty model", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		mockFunc := func(model string, messages []mistral.ChatMessage, params *mistral.ChatRequestParams) (*mistral.ChatCompletionResponse, error) {
			return &mistral.ChatCompletionResponse{}, nil
		}

		_, err := client.TraceMistralRequest(context.Background(), "", nil, nil, mockFunc)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model cannot be empty")
	})

	t.Run("TraceGoogleRequest with nil function", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		_, err := client.TraceGoogleRequest(context.Background(), "model", nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("TraceGoogleRequest with empty model", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		mockFunc := func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{}, nil
		}

		_, err := client.TraceGoogleRequest(context.Background(), "", nil, mockFunc)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "model cannot be empty")
	})

	t.Run("Negative token counts are normalized to zero", func(t *testing.T) {
		storage := &MockStorageAdapter{}
		client := NewClient(storage)

		// Use trackRequest directly to test token validation
		err := client.trackRequest(
			context.Background(),
			ProviderOpenAI,
			"gpt-3.5-turbo",
			-10, // negative input tokens
			-5,  // negative output tokens
			time.Millisecond,
			nil,
			nil,
		)

		assert.NoError(t, err)
		require.Len(t, storage.SaveCalls, 1)
		
		request := storage.SaveCalls[0].Request
		assert.Equal(t, 0, request.InputTokens)  // Should be normalized to 0
		assert.Equal(t, 0, request.OutputTokens) // Should be normalized to 0
	})
}