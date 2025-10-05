package transform

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"athena/internal/config"
)

const (
	contentTypeText = "text"
	roleUser        = "user"
	stopReasonEnd   = "end_turn"
)

// AnthropicToOpenAI converts an Anthropic Messages API request to OpenAI/OpenRouter
// chat completions format. This transformation handles system messages, content blocks,
// tool definitions, and provider routing.
//
// The conversion process:
//   - Extracts system messages from Anthropic format and prepends to messages array
//   - Transforms content blocks (text, tool_use, tool_result) to OpenAI format
//   - Validates tool calls have matching tool responses
//   - Maps Anthropic model names (claude-3-opus) to configured OpenRouter models
//   - Cleans JSON schemas by removing unsupported "format": "uri" properties
//   - Applies provider-specific routing configuration
//
// Returns an OpenAIRequest ready to be sent to OpenRouter or compatible endpoints.
func AnthropicToOpenAI(req AnthropicRequest, cfg *config.Config) OpenAIRequest {
	messages := []OpenAIMessage{}

	// Handle system messages
	if len(req.System) > 0 {
		var systemArray []ContentBlock
		if err := json.Unmarshal(req.System, &systemArray); err == nil {
			for _, item := range systemArray {
				content := []map[string]interface{}{
					{
						"type": "text",
						"text": item.Text,
					},
				}
				if strings.Contains(req.Model, "claude") {
					content[0]["cache_control"] = map[string]string{"type": "ephemeral"}
				}
				messages = append(messages, OpenAIMessage{
					Role:    "system",
					Content: content,
				})
			}
		} else {
			var systemString string
			if err := json.Unmarshal(req.System, &systemString); err == nil {
				content := []map[string]interface{}{
					{
						"type": "text",
						"text": systemString,
					},
				}
				if strings.Contains(req.Model, "claude") {
					content[0]["cache_control"] = map[string]string{"type": "ephemeral"}
				}
				messages = append(messages, OpenAIMessage{
					Role:    "system",
					Content: content,
				})
			}
		}
	}

	// Transform messages
	for _, msg := range req.Messages {
		openAIMsgs := transformMessage(msg)
		messages = append(messages, openAIMsgs...)
	}

	// Validate tool calls
	messages = validateToolCalls(messages)

	mappedModel := MapModel(req.Model, cfg)
	result := OpenAIRequest{
		Model:       mappedModel,
		Messages:    messages,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	}

	// Add provider routing from config
	if provider := GetProviderForModel(req.Model, cfg); provider != nil {
		result.Provider = provider
	}

	// Transform tools
	if len(req.Tools) > 0 {
		tools := []OpenAITool{}
		for _, tool := range req.Tools {
			// Remove format: "uri" from parameters
			cleanedParams := removeUriFormat(tool.InputSchema)
			tools = append(tools, OpenAITool{
				Type: "function",
				Function: struct {
					Name        string          `json:"name"`
					Description string          `json:"description,omitempty"`
					Parameters  json.RawMessage `json:"parameters"`
				}{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  cleanedParams,
				},
			})
		}
		result.Tools = tools
	}

	return result
}

// transformMessage converts a single Anthropic message to OpenAI format
func transformMessage(msg Message) []OpenAIMessage {
	result := []OpenAIMessage{}

	var content []ContentBlock
	if err := json.Unmarshal(msg.Content, &content); err != nil {
		// Try as string
		var strContent string
		if err := json.Unmarshal(msg.Content, &strContent); err == nil {
			result = append(result, OpenAIMessage{
				Role:    msg.Role,
				Content: strContent,
			})
		}
		return result
	}

	if msg.Role == RoleAssistant {
		assistantMsg := OpenAIMessage{
			Role:    "assistant",
			Content: nil,
		}
		textContent := ""
		toolCalls := []ToolCall{}

		for _, block := range content {
			if block.Type == contentTypeText {
				textContent += block.Text + "\n"
			} else if block.Type == TypeToolUse {
				args, _ := json.Marshal(block.Input)
				toolCalls = append(toolCalls, ToolCall{
					ID:   block.ID,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      block.Name,
						Arguments: string(args),
					},
				})
			}
		}

		trimmedText := strings.TrimSpace(textContent)
		if trimmedText != "" {
			assistantMsg.Content = trimmedText
		}
		if len(toolCalls) > 0 {
			assistantMsg.ToolCalls = toolCalls
		}
		if assistantMsg.Content != nil || len(assistantMsg.ToolCalls) > 0 {
			result = append(result, assistantMsg)
		}
	} else if msg.Role == roleUser {
		userText := ""
		toolMessages := []OpenAIMessage{}

		for _, block := range content {
			if block.Type == contentTypeText {
				userText += block.Text + "\n"
			} else if block.Type == "tool_result" {
				var content string
				if err := json.Unmarshal(block.Content, &content); err != nil {
					content = string(block.Content)
				}
				toolMessages = append(toolMessages, OpenAIMessage{
					Role:       "tool",
					ToolCallID: block.ToolUseID,
					Content:    content,
				})
			}
		}

		trimmedText := strings.TrimSpace(userText)
		if trimmedText != "" {
			result = append(result, OpenAIMessage{
				Role:    "user",
				Content: trimmedText,
			})
		}
		result = append(result, toolMessages...)
	}

	return result
}

// validateToolCalls ensures tool calls have matching tool responses
func validateToolCalls(messages []OpenAIMessage) []OpenAIMessage {
	validated := []OpenAIMessage{}

	for i, msg := range messages {
		currentMsg := msg

		//nolint:gocritic // Complex if-else with multiple conditions doesn't translate well to switch
		if msg.Role == RoleAssistant && len(msg.ToolCalls) > 0 {
			validToolCalls := []ToolCall{}

			// Collect immediately following tool messages
			immediateTools := []OpenAIMessage{}
			j := i + 1
			for j < len(messages) && messages[j].Role == RoleTool {
				immediateTools = append(immediateTools, messages[j])
				j++
			}

			// Validate each tool call
			for _, toolCall := range msg.ToolCalls {
				hasMatch := false
				for _, toolMsg := range immediateTools {
					if toolMsg.ToolCallID == toolCall.ID {
						hasMatch = true
						break
					}
				}
				if hasMatch {
					validToolCalls = append(validToolCalls, toolCall)
				}
			}

			if len(validToolCalls) > 0 {
				currentMsg.ToolCalls = validToolCalls
			} else {
				currentMsg.ToolCalls = nil
			}

			if currentMsg.Content != nil || len(currentMsg.ToolCalls) > 0 {
				validated = append(validated, currentMsg)
			}
		} else if msg.Role == RoleTool {
			// Check if previous message has matching tool call
			hasMatch := false
			if i > 0 {
				prevMsg := messages[i-1]
				if prevMsg.Role == RoleAssistant {
					for _, toolCall := range prevMsg.ToolCalls {
						if toolCall.ID == msg.ToolCallID {
							hasMatch = true
							break
						}
					}
				} else if prevMsg.Role == RoleTool {
					// Check for assistant message before tool sequence
					for k := i - 1; k >= 0; k-- {
						if messages[k].Role == RoleTool {
							continue
						}
						if messages[k].Role == RoleAssistant {
							for _, toolCall := range messages[k].ToolCalls {
								if toolCall.ID == msg.ToolCallID {
									hasMatch = true
									break
								}
							}
						}
						break
					}
				}
			}
			if hasMatch {
				validated = append(validated, currentMsg)
			}
		} else {
			validated = append(validated, currentMsg)
		}
	}

	return validated
}

// MapModel maps Anthropic model names to configured OpenRouter model identifiers.
// Provides intelligent routing based on model tier detection:
//
//   - Models containing "opus" → cfg.OpusModel (high-end tier)
//   - Models containing "sonnet" → cfg.SonnetModel (mid-tier)
//   - Models containing "haiku" → cfg.HaikuModel (fast/cheap tier)
//   - Models with "/" → pass-through (already OpenRouter format)
//   - Unknown models → cfg.Model (default fallback)
//
// Example mappings:
//   - "claude-3-opus-20240229" → "anthropic/claude-3-opus"
//   - "claude-3-5-sonnet-20241022" → "openai/gpt-4"
//   - "openai/gpt-4o" → "openai/gpt-4o" (pass-through)
//
// Returns the OpenRouter model ID to use for the API request.
func MapModel(anthropicModel string, cfg *config.Config) string {
	if strings.Contains(anthropicModel, "/") {
		return anthropicModel
	}

	switch {
	case strings.Contains(anthropicModel, "haiku") && cfg.HaikuModel != "":
		return cfg.HaikuModel
	case strings.Contains(anthropicModel, "sonnet") && cfg.SonnetModel != "":
		return cfg.SonnetModel
	case strings.Contains(anthropicModel, "opus") && cfg.OpusModel != "":
		return cfg.OpusModel
	default:
		return cfg.Model // Use default model
	}
}

// GetProviderForModel returns the provider configuration for a given Anthropic model
// name. This enables routing different model tiers through different API providers
// with distinct base URLs and API keys.
//
// Provider selection follows the same tier detection as MapModel:
//   - Models containing "opus" → cfg.OpusProvider
//   - Models containing "sonnet" → cfg.SonnetProvider
//   - Models containing "haiku" → cfg.HaikuProvider
//   - All other models → cfg.DefaultProvider
//
// Example use cases:
//   - Route opus through Anthropic directly (higher rate limits)
//   - Route sonnet through OpenRouter (cost optimization)
//   - Route haiku through local vLLM (low latency)
//
// Returns nil if no provider is configured for the model tier.
func GetProviderForModel(anthropicModel string, cfg *config.Config) *config.ProviderConfig {
	if strings.Contains(anthropicModel, "/") {
		// Direct model ID - use default provider
		return cfg.DefaultProvider
	}

	switch {
	case strings.Contains(anthropicModel, "haiku") && cfg.HaikuModel != "":
		return cfg.HaikuProvider
	case strings.Contains(anthropicModel, "sonnet") && cfg.SonnetModel != "":
		return cfg.SonnetProvider
	case strings.Contains(anthropicModel, "opus") && cfg.OpusModel != "":
		return cfg.OpusProvider
	default:
		return cfg.DefaultProvider
	}
}

// removeUriFormat removes unsupported "format": "uri" from JSON schema
func removeUriFormat(schema json.RawMessage) json.RawMessage {
	var data interface{}
	if err := json.Unmarshal(schema, &data); err != nil {
		return schema
	}

	cleaned := removeUriFormatFromInterface(data)
	result, _ := json.Marshal(cleaned)
	return result
}

// removeUriFormatFromInterface recursively removes "format": "uri" from data
func removeUriFormatFromInterface(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if key == "format" && value == "uri" {
				continue
			}
			result[key] = removeUriFormatFromInterface(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = removeUriFormatFromInterface(item)
		}
		return result
	default:
		return data
	}
}

// OpenAIToAnthropic converts an OpenAI/OpenRouter chat completion response to
// Anthropic Messages API format. This is the reverse transformation of AnthropicToOpenAI,
// ensuring client compatibility with the Anthropic API specification.
//
// The conversion process:
//   - Generates synthetic message ID and timestamp
//   - Extracts text content from choices[0].message.content
//   - Transforms tool_calls to Anthropic tool_use content blocks
//   - Maps finish_reason (stop → end_turn, tool_calls → tool_use)
//   - Calculates token usage from OpenAI usage metrics
//
// Provider-specific handling:
//   - Detects model format (Kimi K2, Qwen, DeepSeek) via DetectModelFormat
//   - Applies format-specific parsing for non-standard tool call formats
//
// Returns an Anthropic-formatted response map ready for JSON serialization.
func OpenAIToAnthropic(resp map[string]interface{}, modelName string) map[string]interface{} {
	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	content := []map[string]interface{}{}
	choices := resp["choices"].([]interface{})
	if len(choices) > 0 {
		choice := choices[0].(map[string]interface{})
		message := choice["message"].(map[string]interface{})

		if msgContent, ok := message["content"]; ok && msgContent != nil {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": msgContent,
			})
		}

		if toolCalls, ok := message["tool_calls"]; ok && toolCalls != nil {
			for _, tc := range toolCalls.([]interface{}) {
				toolCall := tc.(map[string]interface{})
				function := toolCall["function"].(map[string]interface{})
				var input map[string]interface{}
				if args, ok := function["arguments"].(string); ok {
					if err := json.Unmarshal([]byte(args), &input); err != nil {
						// Log error but continue processing
						input = make(map[string]interface{})
					}
				}
				content = append(content, map[string]interface{}{
					"type":  TypeToolUse,
					"id":    toolCall["id"],
					"name":  function["name"],
					"input": input,
				})
			}
		}

		finishReason := choice["finish_reason"].(string)
		stopReason := stopReasonEnd
		if finishReason == "tool_calls" {
			stopReason = TypeToolUse
		}

		return map[string]interface{}{
			"id":            messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       content,
			"stop_reason":   stopReason,
			"stop_sequence": nil,
			"model":         modelName,
		}
	}

	return map[string]interface{}{
		"id":            messageID,
		"type":          "message",
		"role":          "assistant",
		"content":       content,
		"stop_reason":   stopReasonEnd,
		"stop_sequence": nil,
		"model":         modelName,
	}
}

// HandleNonStreaming processes non-streaming (buffered) responses from OpenRouter
// and writes the transformed Anthropic-formatted response to the client.
//
// Processing flow:
//  1. Validates HTTP status code (returns error if non-200)
//  2. Decodes OpenAI JSON response body
//  3. Transforms to Anthropic format via OpenAIToAnthropic
//  4. Writes JSON response with appropriate Content-Type header
//
// Error handling:
//   - Non-200 status: forwards error body and status to client
//   - JSON decode errors: returns 500 Internal Server Error
//   - Encode errors: logs error but response may be partially written
//
// This function is used when the client requests stream=false in the Anthropic API call.
func HandleNonStreaming(w http.ResponseWriter, resp *http.Response, modelName string) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, string(body), resp.StatusCode)
		return
	}

	var openAIResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		http.Error(w, "Failed to decode OpenRouter response", http.StatusInternalServerError)
		return
	}

	anthropicResp := OpenAIToAnthropic(openAIResp, modelName)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(anthropicResp); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// HandleStreaming processes Server-Sent Events (SSE) streaming responses from
// OpenRouter and transforms them into Anthropic Messages API streaming format.
//
// Processing flow:
//  1. Validates HTTP status code (returns error if non-200)
//  2. Sets up SSE headers (text/event-stream, no caching)
//  3. Processes OpenAI delta events line-by-line with buffering
//  4. Transforms to Anthropic SSE events (message_start, content_block_*, message_delta)
//  5. Handles format-specific tool calling (Kimi K2, Qwen, standard OpenAI)
//  6. Manages content block state (text vs tool_use transitions)
//  7. Emits message_stop event when stream completes
//
// Provider-specific streaming:
//   - Standard OpenAI: tool_calls array with incremental deltas
//   - Qwen models: function_call object format with synthetic IDs
//   - Kimi K2: special token format requiring buffering
//
// State management:
//   - Tracks current content block index and type
//   - Buffers incomplete SSE lines across network packets
//   - Accumulates tool call arguments for validation
//
// This function is used when the client requests stream=true in the Anthropic API call.
func HandleStreaming(w http.ResponseWriter, resp *http.Response, modelName string) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, string(body), resp.StatusCode)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	// Send message_start
	sendSSE(w, flusher, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":            messageID,
			"type":          "message",
			"role":          "assistant",
			"content":       []interface{}{},
			"model":         modelName,
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]int{
				"input_tokens":  1,
				"output_tokens": 1,
			},
		},
	})

	contentBlockIndex := 0
	hasStartedTextBlock := false
	isToolUse := false
	currentToolCallID := ""
	toolCallJSONMap := make(map[string]string)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			continue
		}

		if choices, ok := parsed["choices"].([]interface{}); ok && len(choices) > 0 {
			choice := choices[0].(map[string]interface{})
			if delta, ok := choice["delta"].(map[string]interface{}); ok {
				processStreamDelta(w, flusher, delta, &contentBlockIndex, &hasStartedTextBlock,
					&isToolUse, &currentToolCallID, toolCallJSONMap)
			}
		}
	}

	// Close last content block
	if isToolUse || hasStartedTextBlock {
		sendSSE(w, flusher, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": contentBlockIndex,
		})
	}

	// Send message_delta and message_stop
	stopReason := stopReasonEnd
	if isToolUse {
		stopReason = TypeToolUse
	}

	sendSSE(w, flusher, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason":   stopReason,
			"stop_sequence": nil,
		},
		"usage": map[string]int{
			"output_tokens": 150,
		},
	})

	sendSSE(w, flusher, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
}

// processStreamDelta processes individual streaming deltas from OpenRouter
func processStreamDelta(w http.ResponseWriter, flusher http.Flusher, delta map[string]interface{},
	contentBlockIndex *int, hasStartedTextBlock *bool, isToolUse *bool,
	currentToolCallID *string, toolCallJSONMap map[string]string) {

	// Handle tool calls - use parseQwenToolCall to support both formats:
	// 1. Standard OpenAI tool_calls array (vLLM/OpenRouter)
	// 2. Qwen-Agent function_call object
	toolCalls := parseQwenToolCall(delta)
	if len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			// If ID is present and different from current, start new tool call block
			if tc.ID != "" && tc.ID != *currentToolCallID {
				// Close previous block if exists
				if *isToolUse || *hasStartedTextBlock {
					sendSSE(w, flusher, "content_block_stop", map[string]interface{}{
						"type":  "content_block_stop",
						"index": *contentBlockIndex,
					})
				}

				*isToolUse = true
				*hasStartedTextBlock = false
				*currentToolCallID = tc.ID
				*contentBlockIndex++
				toolCallJSONMap[tc.ID] = ""

				sendSSE(w, flusher, "content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": *contentBlockIndex,
					"content_block": map[string]interface{}{
						"type":  TypeToolUse,
						"id":    tc.ID,
						"name":  tc.Function.Name,
						"input": map[string]interface{}{},
					},
				})
			}

			// Send argument deltas (works for both new tool calls and continuations)
			if tc.Function.Arguments != "" && *currentToolCallID != "" {
				toolCallJSONMap[*currentToolCallID] += tc.Function.Arguments
				sendSSE(w, flusher, "content_block_delta", map[string]interface{}{
					"type":  "content_block_delta",
					"index": *contentBlockIndex,
					"delta": map[string]interface{}{
						"type":         "input_json_delta",
						"partial_json": tc.Function.Arguments,
					},
				})
			}
		}
	} else if content, ok := delta["content"].(string); ok && content != "" {
		// Close tool block if transitioning to text
		if *isToolUse {
			sendSSE(w, flusher, "content_block_stop", map[string]interface{}{
				"type":  "content_block_stop",
				"index": *contentBlockIndex,
			})
			*isToolUse = false
			*currentToolCallID = ""
			*contentBlockIndex++
		}

		if !*hasStartedTextBlock {
			sendSSE(w, flusher, "content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": *contentBlockIndex,
				"content_block": map[string]interface{}{
					"type": "text",
					"text": "",
				},
			})
			*hasStartedTextBlock = true
		}

		sendSSE(w, flusher, "content_block_delta", map[string]interface{}{
			"type":  "content_block_delta",
			"index": *contentBlockIndex,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": content,
			},
		})
	}
}

// sendSSE sends a Server-Sent Event
func sendSSE(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	flusher.Flush()
}
