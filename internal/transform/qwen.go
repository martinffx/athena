package transform

import (
	"fmt"
	"time"
)

// parseQwenToolCall accepts both OpenAI tool_calls array AND Qwen-Agent
// function_call object from OpenRouter responses. Handles dual format:
// 1. vLLM format: tool_calls array with id, type, function fields
// 2. Qwen-Agent format: function_call object with name and arguments
// Returns unified ToolCall array with synthetic IDs for function_call format.
func parseQwenToolCall(delta map[string]interface{}) []ToolCall {
	var toolCalls []ToolCall

	// Format 1: OpenAI tool_calls array (vLLM with hermes parser)
	if tcArray, ok := delta["tool_calls"].([]interface{}); ok {
		for _, tc := range tcArray {
			tcMap, ok := tc.(map[string]interface{})
			if !ok {
				continue
			}

			toolCall := ToolCall{
				ID:   getString(tcMap, "id"),
				Type: "function",
			}

			// Extract function details
			if fn, ok := tcMap["function"].(map[string]interface{}); ok {
				toolCall.Function.Name = getString(fn, "name")
				toolCall.Function.Arguments = getString(fn, "arguments")
			}

			toolCalls = append(toolCalls, toolCall)
		}

		if len(toolCalls) > 0 {
			return toolCalls
		}
	}

	// Format 2: Qwen-Agent function_call object
	if fcObj, ok := delta["function_call"].(map[string]interface{}); ok {
		toolCall := ToolCall{
			ID:   generateSyntheticID(),
			Type: "function",
		}

		toolCall.Function.Name = getString(fcObj, "name")
		toolCall.Function.Arguments = getString(fcObj, "arguments")

		return []ToolCall{toolCall}
	}

	// No tool calls present
	return nil
}

// getString safely extracts string value from map, returns empty string if not found
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// generateSyntheticID creates a unique ID for function_call format
// Uses timestamp for uniqueness across multiple tool calls
func generateSyntheticID() string {
	return fmt.Sprintf("qwen-tool-%d", time.Now().UnixNano())
}
