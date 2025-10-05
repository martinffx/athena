package transform

import (
	"testing"
)

func TestDetectModelFormat(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected ModelFormat
	}{
		// OpenRouter format tests (provider/model)
		{
			name:     "OpenRouter Kimi format",
			modelID:  "moonshot/kimi-k2",
			expected: FormatKimi,
		},
		{
			name:     "OpenRouter Qwen format",
			modelID:  "qwen/qwen3-coder",
			expected: FormatQwen,
		},
		{
			name:     "OpenRouter DeepSeek format",
			modelID:  "deepseek/deepseek-chat",
			expected: FormatDeepSeek,
		},

		// Keyword matching tests
		{
			name:     "Keyword Kimi detection",
			modelID:  "kimi-k2-instruct",
			expected: FormatKimi,
		},
		{
			name:     "Keyword Qwen detection",
			modelID:  "qwen3-coder-plus",
			expected: FormatQwen,
		},
		{
			name:     "Keyword DeepSeek detection",
			modelID:  "deepseek-r1",
			expected: FormatDeepSeek,
		},

		// Case insensitivity tests
		{
			name:     "Case insensitive Kimi",
			modelID:  "KIMI-K2",
			expected: FormatKimi,
		},
		{
			name:     "Case insensitive DeepSeek",
			modelID:  "DeepSeek-V3",
			expected: FormatDeepSeek,
		},

		// Precedence tests (Kimi > Qwen > DeepSeek)
		{
			name:     "Precedence: Qwen over DeepSeek",
			modelID:  "qwen-deepseek-mix",
			expected: FormatQwen,
		},
		{
			name:     "Precedence: Kimi over Qwen",
			modelID:  "kimi-qwen-hybrid",
			expected: FormatKimi,
		},

		// Fallback tests
		{
			name:     "Unknown model fallback",
			modelID:  "unknown-model",
			expected: FormatStandard,
		},
		{
			name:     "GPT model fallback",
			modelID:  "gpt-4",
			expected: FormatStandard,
		},
		{
			name:     "Empty model ID fallback",
			modelID:  "",
			expected: FormatStandard,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectModelFormat(tt.modelID)
			if got != tt.expected {
				t.Errorf("DetectModelFormat(%q) = %v, want %v",
					tt.modelID, got, tt.expected)
			}
		})
	}
}
