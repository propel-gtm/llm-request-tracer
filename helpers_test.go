package llmtracer

import (
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		model        string
		inputTokens  int
		outputTokens int
		expectedCost float64
	}{
		{
			name:         "OpenAI GPT-4",
			provider:     ProviderOpenAI,
			model:        "gpt-4",
			inputTokens:  1000,
			outputTokens: 500,
			expectedCost: 0.06, // 1000 * 0.00003 + 500 * 0.00006
		},
		{
			name:         "OpenAI GPT-3.5-turbo",
			provider:     ProviderOpenAI,
			model:        "gpt-3.5-turbo",
			inputTokens:  1000,
			outputTokens: 500,
			expectedCost: 0.0025, // 1000 * 0.0000015 + 500 * 0.000002
		},
		{
			name:         "Anthropic Claude-3-sonnet",
			provider:     ProviderAnthropic,
			model:        "claude-3-sonnet",
			inputTokens:  1000,
			outputTokens: 500,
			expectedCost: 0.0105, // 1000 * 0.000003 + 500 * 0.000015 = 0.003 + 0.0075
		},
		{
			name:         "Google Gemini Pro",
			provider:     ProviderGoogle,
			model:        "gemini-pro",
			inputTokens:  1000,
			outputTokens: 500,
			expectedCost: 0.00125, // 1000 * 0.0000005 + 500 * 0.0000015 = 0.0005 + 0.00075
		},
		{
			name:         "Unknown Provider",
			provider:     ProviderCustom,
			model:        "custom-model",
			inputTokens:  1000,
			outputTokens: 500,
			expectedCost: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.provider, tt.model, tt.inputTokens, tt.outputTokens)
			// Use a small epsilon for floating point comparison
			epsilon := 0.000001
			if diff := cost - tt.expectedCost; diff > epsilon || diff < -epsilon {
				t.Errorf("Expected cost %.6f, got %.6f", tt.expectedCost, cost)
			}
		})
	}
}