package transform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Compiled regex patterns for Kimi K2 tool call parsing (optimized for reuse)
var (
	kimiSectionPattern  = regexp.MustCompile(`(?s)<\|tool_calls_section_begin\|>(.*?)<\|tool_calls_section_end\|>`)
	kimiToolCallPattern = regexp.MustCompile(`(?s)<\|tool_call_begin\|>\s*(.+?)\s*<\|tool_call_argument_begin\|>\s*(.*?)\s*<\|tool_call_end\|>`)
	kimiIDPattern       = regexp.MustCompile(`^functions\.(.+?):(\d+)$`)
)

// parseKimiToolCalls extracts tool calls from Kimi K2's special token format.
// Format: <|tool_calls_section_begin|>...<|tool_calls_section_end|>
// Each tool call: <|tool_call_begin|>functions.{name}:{idx}<|tool_call_argument_begin|>{json}<|tool_call_end|>
func parseKimiToolCalls(content string) ([]ToolCall, error) {
	// Check if content contains tool calls section
	if !strings.Contains(content, "<|tool_calls_section_begin|>") {
		// No tool calls present - not an error, return empty array
		return []ToolCall{}, nil
	}

	// Verify section is properly closed
	if !strings.Contains(content, "<|tool_calls_section_end|>") {
		return nil, fmt.Errorf("malformed Kimi tool calls: missing section end token")
	}

	// Extract the tool calls section using pre-compiled regex
	sectionMatch := kimiSectionPattern.FindStringSubmatch(content)
	if len(sectionMatch) < 2 {
		return nil, fmt.Errorf("failed to extract tool calls section")
	}
	section := sectionMatch[1]

	// Extract individual tool calls using pre-compiled regex
	matches := kimiToolCallPattern.FindAllStringSubmatch(section, -1)

	if len(matches) == 0 {
		// Section exists but no valid tool calls
		return []ToolCall{}, nil
	}

	var toolCalls []ToolCall
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		fullID := strings.TrimSpace(match[1])
		argsJSON := strings.TrimSpace(match[2])

		// Parse ID format: functions.{function_name}:{index}
		// Example: functions.get_weather:0
		idMatch := kimiIDPattern.FindStringSubmatch(fullID)
		if len(idMatch) < 3 {
			return nil, fmt.Errorf("invalid Kimi tool call ID format: %q (expected functions.{name}:{idx})", fullID)
		}

		functionName := idMatch[1]

		// Validate JSON arguments
		if !json.Valid([]byte(argsJSON)) {
			return nil, fmt.Errorf("invalid JSON arguments in tool call %q: %s", fullID, argsJSON)
		}

		// Build ToolCall struct
		toolCall := ToolCall{
			ID:   fullID,
			Type: "function",
		}
		toolCall.Function.Name = functionName
		toolCall.Function.Arguments = argsJSON

		toolCalls = append(toolCalls, toolCall)
	}

	return toolCalls, nil
}

// handleKimiStreaming processes streaming chunks for Kimi K2 special token format.
// Buffers chunks until complete tool_calls_section is received, then parses and emits
// Anthropic SSE events. Returns error if buffer limit exceeded or parsing fails.
func handleKimiStreaming(w http.ResponseWriter, flusher http.Flusher, state *StreamState, chunk string) error {
	// Append chunk to buffer
	state.FormatContext.KimiBuffer.WriteString(chunk)

	// Check buffer limit (10KB default)
	if state.FormatContext.KimiBuffer.Len() > state.FormatContext.KimiBufferLimit {
		// Send error event and terminate
		sendStreamError(w, flusher, "overloaded", "Kimi tool call buffer exceeded 10KB limit")
		return fmt.Errorf("Kimi buffer exceeded limit: %d bytes", state.FormatContext.KimiBuffer.Len())
	}

	bufferedContent := state.FormatContext.KimiBuffer.String()

	// Check if we have a complete section
	if !strings.Contains(bufferedContent, "<|tool_calls_section_end|>") {
		// Incomplete section, continue buffering
		return nil
	}

	// Parse complete tool calls
	toolCalls, err := parseKimiToolCalls(bufferedContent)
	if err != nil {
		sendStreamError(w, flusher, "invalid_request_error", fmt.Sprintf("Failed to parse Kimi tool calls: %v", err))
		return err
	}

	// Emit Anthropic SSE events for each tool call
	for _, tc := range toolCalls {
		// Emit content_block_start event
		startEvent := map[string]interface{}{
			"type":  "content_block_start",
			"index": state.ContentBlockIndex,
			"content_block": map[string]interface{}{
				"type": "tool_use",
				"id":   tc.ID,
				"name": tc.Function.Name,
			},
		}
		emitSSEEvent(w, "content_block_start", startEvent)

		// Emit content_block_delta event with input
		deltaEvent := map[string]interface{}{
			"type":  "content_block_delta",
			"index": state.ContentBlockIndex,
			"delta": map[string]interface{}{
				"type":         "input_json_delta",
				"partial_json": tc.Function.Arguments,
			},
		}
		emitSSEEvent(w, "content_block_delta", deltaEvent)

		// Emit content_block_stop event
		stopEvent := map[string]interface{}{
			"type":  "content_block_stop",
			"index": state.ContentBlockIndex,
		}
		emitSSEEvent(w, "content_block_stop", stopEvent)

		state.ContentBlockIndex++
	}

	flusher.Flush()

	// Clear buffer after successful emission
	state.FormatContext.KimiBuffer.Reset()

	return nil
}
