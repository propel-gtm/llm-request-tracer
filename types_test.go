package llmtracer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorTypeNone,
		},
		{
			name:     "rate limit - explicit",
			err:      errors.New("rate limit exceeded"),
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "rate limit - too many requests",
			err:      errors.New("Too many requests"),
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "rate limit - 429 status",
			err:      errors.New("error 429: too many requests"),
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "authentication - unauthorized",
			err:      errors.New("unauthorized access"),
			expected: ErrorTypeAuthentication,
		},
		{
			name:     "authentication - api key",
			err:      errors.New("invalid API key provided"),
			expected: ErrorTypeAuthentication,
		},
		{
			name:     "authentication - 401",
			err:      errors.New("401 Unauthorized"),
			expected: ErrorTypeAuthentication,
		},
		{
			name:     "authentication - 403",
			err:      errors.New("403 Forbidden"),
			expected: ErrorTypeAuthentication,
		},
		{
			name:     "timeout - explicit",
			err:      errors.New("request timeout"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "timeout - deadline exceeded",
			err:      errors.New("context deadline exceeded"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "timeout - context canceled",
			err:      errors.New("context canceled"),
			expected: ErrorTypeTimeout,
		},
		{
			name:     "network - connection",
			err:      errors.New("connection refused"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "network - dial",
			err:      errors.New("dial tcp: connection timeout"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "network - dns",
			err:      errors.New("DNS resolution failed"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "network - no such host",
			err:      errors.New("no such host"),
			expected: ErrorTypeNetwork,
		},
		{
			name:     "invalid request - explicit",
			err:      errors.New("invalid request format"),
			expected: ErrorTypeInvalidRequest,
		},
		{
			name:     "invalid request - bad request",
			err:      errors.New("bad request"),
			expected: ErrorTypeInvalidRequest,
		},
		{
			name:     "invalid request - 400",
			err:      errors.New("400 Bad Request"),
			expected: ErrorTypeInvalidRequest,
		},
		{
			name:     "invalid request - malformed",
			err:      errors.New("malformed JSON"),
			expected: ErrorTypeInvalidRequest,
		},
		{
			name:     "server error - 500",
			err:      errors.New("500 Internal Server Error"),
			expected: ErrorTypeServerError,
		},
		{
			name:     "server error - 502",
			err:      errors.New("502 Bad Gateway"),
			expected: ErrorTypeServerError,
		},
		{
			name:     "server error - 503",
			err:      errors.New("503 Service Unavailable"),
			expected: ErrorTypeServerError,
		},
		{
			name:     "server error - 504",
			err:      errors.New("504 Gateway Timeout"),
			expected: ErrorTypeServerError,
		},
		{
			name:     "server error - internal",
			err:      errors.New("internal error occurred"),
			expected: ErrorTypeServerError,
		},
		{
			name:     "unknown error",
			err:      errors.New("something went wrong"),
			expected: ErrorTypeUnknown,
		},
		{
			name:     "mixed case should work",
			err:      errors.New("RATE LIMIT EXCEEDED"),
			expected: ErrorTypeRateLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorTypeConstants(t *testing.T) {
	// Ensure all error types are distinct
	errorTypes := []ErrorType{
		ErrorTypeNone,
		ErrorTypeNetwork,
		ErrorTypeRateLimit,
		ErrorTypeAuthentication,
		ErrorTypeInvalidRequest,
		ErrorTypeTimeout,
		ErrorTypeServerError,
		ErrorTypeUnknown,
	}

	seen := make(map[ErrorType]bool)
	for _, et := range errorTypes {
		assert.False(t, seen[et], "Duplicate error type: %s", et)
		seen[et] = true
	}
}

func TestCircuitBreakerError(t *testing.T) {
	assert.Equal(t, "circuit breaker is open", ErrCircuitOpen.Error())
}
