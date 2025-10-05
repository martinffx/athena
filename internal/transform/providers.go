package transform

import (
	"strings"
)

// DetectModelFormat analyzes model identifier to determine which tool calling
// response format OpenRouter will use. Returns ModelFormat enum based on model
// name pattern matching with precedence: Kimi > Qwen > DeepSeek > Standard
func DetectModelFormat(modelID string) ModelFormat {
	// Normalize to lowercase for case-insensitive matching
	normalized := strings.ToLower(modelID)

	// 1. Check OpenRouter format (provider/model)
	// Note: Only handles two-part format. Multi-part paths (e.g., provider/model/version)
	// fall through to keyword matching
	if strings.Contains(normalized, "/") {
		parts := strings.Split(normalized, "/")
		if len(parts) == 2 {
			provider := parts[0]
			switch provider {
			case "moonshot": // Kimi's OpenRouter provider
				return FormatKimi
			case "qwen":
				return FormatQwen
			case "deepseek":
				return FormatDeepSeek
			}
		}
	}

	// 2. Keyword matching with precedence order: Kimi > Qwen > DeepSeek
	// Check Kimi first (highest precedence)
	// Be more specific with k2 matching to avoid false positives
	if strings.Contains(normalized, "kimi") ||
		strings.Contains(normalized, "moonshot-k2") ||
		strings.Contains(normalized, "-k2") {
		return FormatKimi
	}

	// Check Qwen second
	if strings.Contains(normalized, "qwen") {
		return FormatQwen
	}

	// Check DeepSeek third
	if strings.Contains(normalized, "deepseek") {
		return FormatDeepSeek
	}

	// 3. Default fallback
	return FormatStandard
}
