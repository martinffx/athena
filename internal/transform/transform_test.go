package transform

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"athena/internal/config"
)

const (
	testToolCallID    = "call_123"
	testToolName      = "search"
	testRoleUser      = "user"
	testRoleAssistant = "assistant"
	testContentType   = "text"
)

func TestMapModel(t *testing.T) {
	cfg := &config.Config{
		Model:       "default/model",
		OpusModel:   "custom/opus",
		SonnetModel: "custom/sonnet",
		HaikuModel:  "custom/haiku",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "opus model",
			input:    "claude-3-opus",
			expected: "custom/opus",
		},
		{
			name:     "opus with version",
			input:    "claude-3-opus-20240229",
			expected: "custom/opus",
		},
		{
			name:     "sonnet model",
			input:    "claude-3-5-sonnet",
			expected: "custom/sonnet",
		},
		{
			name:     "sonnet with version",
			input:    "claude-3-5-sonnet-20240620",
			expected: "custom/sonnet",
		},
		{
			name:     "haiku model",
			input:    "claude-3-haiku",
			expected: "custom/haiku",
		},
		{
			name:     "haiku with version",
			input:    "claude-3-haiku-20240307",
			expected: "custom/haiku",
		},
		{
			name:     "passthrough with slash",
			input:    "anthropic/claude-3-opus",
			expected: "anthropic/claude-3-opus",
		},
		{
			name:     "unknown model defaults",
			input:    "unknown-model",
			expected: "default/model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapModel(tt.input, cfg)
			if result != tt.expected {
				t.Errorf("MapModel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveUriFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes format uri",
			input:    `{"type":"object","properties":{"url":{"type":"string","format":"uri"}}}`,
			expected: `{"properties":{"url":{"type":"string"}},"type":"object"}`,
		},
		{
			name:     "nested format uri",
			input:    `{"type":"object","properties":{"data":{"type":"object","properties":{"link":{"type":"string","format":"uri"}}}}}`,
			expected: `{"properties":{"data":{"properties":{"link":{"type":"string"}},"type":"object"}},"type":"object"}`,
		},
		{
			name:     "preserves other formats",
			input:    `{"type":"object","properties":{"date":{"type":"string","format":"date-time"}}}`,
			expected: `{"properties":{"date":{"format":"date-time","type":"string"}},"type":"object"}`,
		},
		{
			name:     "no format field",
			input:    `{"type":"object","properties":{"name":{"type":"string"}}}`,
			expected: `{"properties":{"name":{"type":"string"}},"type":"object"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeUriFormat(json.RawMessage(tt.input))

			// Unmarshal both to compare structure
			var resultMap, expectedMap map[string]interface{}
			if err := json.Unmarshal(result, &resultMap); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &expectedMap); err != nil {
				t.Fatalf("Failed to unmarshal expected: %v", err)
			}

			resultJSON, _ := json.Marshal(resultMap)
			expectedJSON, _ := json.Marshal(expectedMap)

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("removeUriFormat() = %s, expected %s", resultJSON, expectedJSON)
			}
		})
	}
}

func TestAnthropicToOpenAI_SimpleMessage(t *testing.T) {
	cfg := &config.Config{
		Model:       "test/model",
		OpusModel:   "test/opus",
		SonnetModel: "test/sonnet",
		HaikuModel:  "test/haiku",
	}

	req := AnthropicRequest{
		Model: "claude-3-sonnet",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello, how are you?"`),
			},
		},
	}

	result := AnthropicToOpenAI(req, cfg)

	if result.Model != "test/sonnet" {
		t.Errorf("Model = %q, expected %q", result.Model, "test/sonnet")
	}

	if len(result.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result.Messages))
	}

	if result.Messages[0].Role != "user" {
		t.Errorf("Message role = %q, expected %q", result.Messages[0].Role, "user")
	}

	content, ok := result.Messages[0].Content.(string)
	if !ok {
		t.Fatalf("Message content is not a string")
	}

	if content != "Hello, how are you?" {
		t.Errorf("Message content = %q, expected %q", content, "Hello, how are you?")
	}
}

func TestAnthropicToOpenAI_WithSystem(t *testing.T) {
	cfg := &config.Config{
		Model:       "test/model",
		OpusModel:   "test/opus",
		SonnetModel: "test/sonnet",
		HaikuModel:  "test/haiku",
	}

	req := AnthropicRequest{
		Model:  "claude-3-sonnet",
		System: json.RawMessage(`"You are a helpful assistant."`),
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
	}

	result := AnthropicToOpenAI(req, cfg)

	if len(result.Messages) != 2 {
		t.Fatalf("Expected 2 messages (system + user), got %d", len(result.Messages))
	}

	if result.Messages[0].Role != "system" {
		t.Errorf("First message role = %q, expected %q", result.Messages[0].Role, "system")
	}

	// System content should be an array with text block
	contentArray, ok := result.Messages[0].Content.([]map[string]interface{})
	if !ok {
		t.Fatalf("System message content is not an array")
	}

	if len(contentArray) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(contentArray))
	}

	if contentArray[0]["type"] != testContentType {
		t.Errorf("Content block type = %q, expected %q", contentArray[0]["type"], testContentType)
	}

	if contentArray[0]["text"] != "You are a helpful assistant." {
		t.Errorf("Content block text = %q, expected %q", contentArray[0]["text"], "You are a helpful assistant.")
	}
}

func TestAnthropicToOpenAI_WithTools(t *testing.T) {
	cfg := &config.Config{
		Model:       "test/model",
		OpusModel:   "test/opus",
		SonnetModel: "test/sonnet",
		HaikuModel:  "test/haiku",
	}

	toolSchema := `{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`

	req := AnthropicRequest{
		Model: "claude-3-sonnet",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Search for something"`),
			},
		},
		Tools: []Tool{
			{
				Name:        testToolName,
				Description: "Search the web",
				InputSchema: json.RawMessage(toolSchema),
			},
		},
	}

	result := AnthropicToOpenAI(req, cfg)

	if len(result.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Type != "function" {
		t.Errorf("Tool type = %q, expected %q", result.Tools[0].Type, "function")
	}

	if result.Tools[0].Function.Name != testToolName {
		t.Errorf("Tool name = %q, expected %q", result.Tools[0].Function.Name, testToolName)
	}

	if result.Tools[0].Function.Description != "Search the web" {
		t.Errorf("Tool description = %q, expected %q", result.Tools[0].Function.Description, "Search the web")
	}
}

func TestAnthropicToOpenAI_ToolCall(t *testing.T) {
	cfg := &config.Config{Model: "test/model"}

	req := AnthropicRequest{
		Model: "test-model",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Search for cats"`),
			},
			{
				Role: "assistant",
				Content: json.RawMessage(`[
					{"type":"text","text":"I'll search for that."},
					{"type":"tool_use","id":"` + testToolCallID + `","name":"` + testToolName + `","input":{"query":"cats"}}
				]`),
			},
			{
				Role: "user",
				Content: json.RawMessage(`[
					{"type":"tool_result","tool_use_id":"` + testToolCallID + `","content":"Found 10 results"}
				]`),
			},
		},
	}

	result := AnthropicToOpenAI(req, cfg)

	if len(result.Messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(result.Messages))
	}

	// Check assistant message with tool call
	assistantMsg := result.Messages[1]
	if assistantMsg.Role != testRoleAssistant {
		t.Errorf("Assistant message role = %q, expected %q", assistantMsg.Role, testRoleAssistant)
	}

	if len(assistantMsg.ToolCalls) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
	}

	toolCall := assistantMsg.ToolCalls[0]
	if toolCall.ID != testToolCallID {
		t.Errorf("Tool call ID = %q, expected %q", toolCall.ID, testToolCallID)
	}

	if toolCall.Function.Name != testToolName {
		t.Errorf("Tool call function name = %q, expected %q", toolCall.Function.Name, testToolName)
	}

	// Check tool response message
	toolMsg := result.Messages[2]
	if toolMsg.Role != "tool" {
		t.Errorf("Tool message role = %q, expected %q", toolMsg.Role, "tool")
	}

	if toolMsg.ToolCallID != testToolCallID {
		t.Errorf("Tool message call ID = %q, expected %q", toolMsg.ToolCallID, testToolCallID)
	}
}

func TestValidateToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		messages []OpenAIMessage
		expected int // expected number of messages after validation
	}{
		{
			name: "valid tool call with response",
			messages: []OpenAIMessage{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function"},
					},
				},
				{
					Role:       "tool",
					ToolCallID: "call_1",
					Content:    "result",
				},
			},
			expected: 2,
		},
		{
			name: "tool call without response gets removed",
			messages: []OpenAIMessage{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{ID: "call_orphan", Type: "function"},
					},
				},
				{
					Role:    "user",
					Content: "next message",
				},
			},
			expected: 1, // assistant message removed because no tool calls remain
		},
		{
			name: "tool response without call gets removed",
			messages: []OpenAIMessage{
				{
					Role:    "user",
					Content: "hello",
				},
				{
					Role:       "tool",
					ToolCallID: "call_missing",
					Content:    "result",
				},
			},
			expected: 1, // tool message removed
		},
		{
			name: "multiple tool calls partial match",
			messages: []OpenAIMessage{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{ID: "call_1", Type: "function"},
						{ID: "call_2", Type: "function"},
					},
				},
				{
					Role:       "tool",
					ToolCallID: "call_1",
					Content:    "result1",
				},
			},
			expected: 2, // assistant keeps only call_1, tool message stays
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateToolCalls(tt.messages)
			if len(result) != tt.expected {
				t.Errorf("validateToolCalls() returned %d messages, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestOpenAIToAnthropic(t *testing.T) {
	resp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": "This is a response",
				},
				"finish_reason": "end_turn",
			},
		},
	}

	result := OpenAIToAnthropic(resp, "test/model")

	if result["type"] != "message" {
		t.Errorf("Response type = %v, expected %q", result["type"], "message")
	}

	if result["role"] != "assistant" {
		t.Errorf("Response role = %v, expected %q", result["role"], "assistant")
	}

	if result["model"] != "test/model" {
		t.Errorf("Response model = %v, expected %q", result["model"], "test/model")
	}

	if result["stop_reason"] != "end_turn" {
		t.Errorf("Response stop_reason = %v, expected %q", result["stop_reason"], "end_turn")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Response content is not an array")
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != testContentType {
		t.Errorf("Content type = %v, expected %q", content[0]["type"], testContentType)
	}

	if content[0]["text"] != "This is a response" {
		t.Errorf("Content text = %v, expected %q", content[0]["text"], "This is a response")
	}
}

func TestOpenAIToAnthropic_WithToolCalls(t *testing.T) {
	resp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": "Let me search for that",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_abc",
							"type": "function",
							"function": map[string]interface{}{
								"name":      testToolName,
								"arguments": `{"query":"test"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}

	result := OpenAIToAnthropic(resp, "test/model")

	if result["stop_reason"] != "tool_use" {
		t.Errorf("Response stop_reason = %v, expected %q", result["stop_reason"], "tool_use")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Response content is not an array")
	}

	if len(content) != 2 {
		t.Fatalf("Expected 2 content blocks (text + tool_use), got %d", len(content))
	}

	// Check text block
	if content[0]["type"] != testContentType {
		t.Errorf("First content type = %v, expected %q", content[0]["type"], testContentType)
	}

	// Check tool_use block
	if content[1]["type"] != "tool_use" {
		t.Errorf("Second content type = %v, expected %q", content[1]["type"], "tool_use")
	}

	if content[1]["id"] != "call_abc" {
		t.Errorf("Tool use ID = %v, expected %q", content[1]["id"], "call_abc")
	}

	if content[1]["name"] != testToolName {
		t.Errorf("Tool use name = %v, expected %q", content[1]["name"], testToolName)
	}

	input, ok := content[1]["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool use input is not a map")
	}

	if input["query"] != "test" {
		t.Errorf("Tool use input query = %v, expected %q", input["query"], "test")
	}
}

func TestTransformMessage_AssistantWithText(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: json.RawMessage(`[{"type":"text","text":"Hello there"}]`),
	}

	result := transformMessage(msg)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	if result[0].Role != "assistant" {
		t.Errorf("Role = %q, expected %q", result[0].Role, "assistant")
	}

	content, ok := result[0].Content.(string)
	if !ok {
		t.Fatalf("Content is not a string")
	}

	if content != "Hello there" {
		t.Errorf("Content = %q, expected %q", content, "Hello there")
	}
}

func TestTransformMessage_UserWithToolResult(t *testing.T) {
	msg := Message{
		Role: "user",
		Content: json.RawMessage(`[
			{"type":"text","text":"Here's the result:"},
			{"type":"tool_result","tool_use_id":"` + testToolCallID + `","content":"Result data"}
		]`),
	}

	result := transformMessage(msg)

	if len(result) != 2 {
		t.Fatalf("Expected 2 messages (user + tool), got %d", len(result))
	}

	// Check user message
	if result[0].Role != "user" {
		t.Errorf("First message role = %q, expected %q", result[0].Role, "user")
	}

	// Check tool message
	if result[1].Role != "tool" {
		t.Errorf("Second message role = %q, expected %q", result[1].Role, "tool")
	}

	if result[1].ToolCallID != testToolCallID {
		t.Errorf("Tool call ID = %q, expected %q", result[1].ToolCallID, testToolCallID)
	}

	content, ok := result[1].Content.(string)
	if !ok {
		t.Fatalf("Tool message content is not a string")
	}

	if content != "Result data" {
		t.Errorf("Tool content = %q, expected %q", content, "Result data")
	}
}

func TestRemoveUriFormatFromInterface(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name: "removes format uri from map",
			input: map[string]interface{}{
				"format": "uri",
				"type":   "string",
			},
			expected: map[string]interface{}{
				"type": "string",
			},
		},
		{
			name: "preserves other formats",
			input: map[string]interface{}{
				"format": "date-time",
				"type":   "string",
			},
			expected: map[string]interface{}{
				"format": "date-time",
				"type":   "string",
			},
		},
		{
			name: "handles nested maps",
			input: map[string]interface{}{
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"format": "uri",
						"type":   "string",
					},
				},
			},
			expected: map[string]interface{}{
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
		{
			name: "handles arrays",
			input: []interface{}{
				map[string]interface{}{
					"format": "uri",
					"type":   "string",
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"type": "string",
				},
			},
		},
		{
			name:     "preserves primitives",
			input:    "plain string",
			expected: "plain string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeUriFormatFromInterface(tt.input)

			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(tt.expected)

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("removeUriFormatFromInterface() = %s, expected %s", resultJSON, expectedJSON)
			}
		})
	}
}

func TestHandleNonStreaming(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   map[string]interface{}
		expectedStatus int
	}{
		{
			name:       "successful response",
			statusCode: 200,
			responseBody: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"content": "Test response",
						},
						"finish_reason": "stop",
					},
				},
			},
			expectedStatus: 200,
		},
		{
			name:           "error response",
			statusCode:     500,
			responseBody:   map[string]interface{}{"error": "Server error"},
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP response
			var body []byte
			var err error
			if tt.responseBody != nil {
				body, err = json.Marshal(tt.responseBody)
				if err != nil {
					t.Fatalf("Failed to marshal response body: %v", err)
				}
			}

			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}

			w := httptest.NewRecorder()
			HandleNonStreaming(w, resp, "test/model")

			result := w.Result()
			defer result.Body.Close()

			if result.StatusCode != tt.expectedStatus {
				t.Errorf("Status code = %d, expected %d", result.StatusCode, tt.expectedStatus)
			}
		})
	}
}

func TestHandleStreaming(t *testing.T) {
	// Create a mock streaming response
	streamData := `data: {"choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: [DONE]

`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(streamData)),
		Header:     make(http.Header),
	}

	w := httptest.NewRecorder()
	HandleStreaming(w, resp, "test/model")

	result := w.Result()
	defer result.Body.Close()

	if result.StatusCode != 200 {
		t.Errorf("Status code = %d, expected %d", result.StatusCode, 200)
	}

	if result.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, expected %q", result.Header.Get("Content-Type"), "text/event-stream")
	}

	body, _ := io.ReadAll(result.Body)
	bodyStr := string(body)

	// Verify streaming events
	if !strings.Contains(bodyStr, "event: message_start") {
		t.Error("Response should contain message_start event")
	}

	if !strings.Contains(bodyStr, "event: content_block_start") {
		t.Error("Response should contain content_block_start event")
	}

	if !strings.Contains(bodyStr, "event: content_block_delta") {
		t.Error("Response should contain content_block_delta event")
	}

	if !strings.Contains(bodyStr, "event: message_stop") {
		t.Error("Response should contain message_stop event")
	}
}

func TestHandleStreaming_Error(t *testing.T) {
	resp := &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader(`{"error":"Unauthorized"}`)),
		Header:     make(http.Header),
	}

	w := httptest.NewRecorder()
	HandleStreaming(w, resp, "test/model")

	result := w.Result()
	defer result.Body.Close()

	if result.StatusCode != 401 {
		t.Errorf("Status code = %d, expected %d", result.StatusCode, 401)
	}
}

func TestHandleStreaming_WithToolCalls(t *testing.T) {
	// Create a mock streaming response with tool calls
	streamData := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"id":"` + testToolCallID + `","type":"function","function":{"name":"` + testToolName + `"}}]},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"{\"q"}}]},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"uery\":"}}]},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"\"test\"}"}}]},"finish_reason":null}]}

data: [DONE]

`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(streamData)),
		Header:     make(http.Header),
	}

	w := httptest.NewRecorder()
	HandleStreaming(w, resp, "test/model")

	result := w.Result()
	defer result.Body.Close()

	if result.StatusCode != 200 {
		t.Errorf("Status code = %d, expected %d", result.StatusCode, 200)
	}

	body, _ := io.ReadAll(result.Body)
	bodyStr := string(body)

	// Verify tool use events
	if !strings.Contains(bodyStr, "\"type\":\"tool_use\"") {
		t.Error("Response should contain tool_use content block")
	}

	if !strings.Contains(bodyStr, "input_json_delta") {
		t.Error("Response should contain input_json_delta events")
	}
}

func TestHandleStreaming_TextAndToolMix(t *testing.T) {
	// Test streaming that transitions from tool calls to text
	streamData := `data: {"choices":[{"index":0,"delta":{"tool_calls":[{"id":"call_456","type":"function","function":{"name":"calculate"}}]},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"tool_calls":[{"function":{"arguments":"{\"x\":5}"}}]},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"content":"Result is"},"finish_reason":null}]}

data: {"choices":[{"index":0,"delta":{"content":" ready"},"finish_reason":null}]}

data: [DONE]

`

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(streamData)),
		Header:     make(http.Header),
	}

	w := httptest.NewRecorder()
	HandleStreaming(w, resp, "test/model")

	result := w.Result()
	defer result.Body.Close()

	body, _ := io.ReadAll(result.Body)
	bodyStr := string(body)

	// Should have both tool use and text content blocks
	if !strings.Contains(bodyStr, "\"type\":\"tool_use\"") {
		t.Error("Response should contain tool_use content block")
	}

	if !strings.Contains(bodyStr, "\"type\":\"text\"") {
		t.Error("Response should contain text content block")
	}

	// Should have content_block_stop for tool use before text starts
	stops := strings.Count(bodyStr, "content_block_stop")
	if stops < 2 {
		t.Errorf("Expected at least 2 content_block_stop events, got %d", stops)
	}
}

func TestAnthropicToOpenAI_ProviderRouting(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		cfg              *config.Config
		expectProvider   bool
		expectedOrder    []string
		expectedFallback bool
	}{
		{
			name:  "kimi-k2 model gets Groq provider from config",
			model: "kimi-k2-test",
			cfg: &config.Config{
				Model: "moonshotai/kimi-k2-0905",
				DefaultProvider: &config.ProviderConfig{
					Order:          []string{"Groq"},
					AllowFallbacks: false,
				},
			},
			expectProvider:   true,
			expectedOrder:    []string{"Groq"},
			expectedFallback: false,
		},
		{
			name:  "opus model uses opus provider config",
			model: "claude-3-opus",
			cfg: &config.Config{
				OpusModel: "anthropic/claude-3-opus",
				OpusProvider: &config.ProviderConfig{
					Order:          []string{"Anthropic"},
					AllowFallbacks: true,
				},
			},
			expectProvider:   true,
			expectedOrder:    []string{"Anthropic"},
			expectedFallback: true,
		},
		{
			name:  "model without provider config has no provider",
			model: "claude-3-sonnet",
			cfg: &config.Config{
				SonnetModel: "anthropic/claude-3.5-sonnet",
			},
			expectProvider: false,
		},
		{
			name:  "direct model ID uses default provider",
			model: "openai/gpt-4",
			cfg: &config.Config{
				Model: "openai/gpt-4",
				DefaultProvider: &config.ProviderConfig{
					Order:          []string{"OpenAI"},
					AllowFallbacks: true,
				},
			},
			expectProvider:   true,
			expectedOrder:    []string{"OpenAI"},
			expectedFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := AnthropicRequest{
				Model: tt.model,
				Messages: []Message{
					{
						Role:    "user",
						Content: json.RawMessage(`"test message"`),
					},
				},
			}

			result := AnthropicToOpenAI(req, tt.cfg)

			if tt.expectProvider {
				if result.Provider == nil {
					t.Fatal("Expected Provider to be set")
				}
				if len(result.Provider.Order) != len(tt.expectedOrder) {
					t.Errorf("Provider.Order length = %d, expected %d", len(result.Provider.Order), len(tt.expectedOrder))
				}
				for i, provider := range tt.expectedOrder {
					if result.Provider.Order[i] != provider {
						t.Errorf("Provider.Order[%d] = %q, expected %q", i, result.Provider.Order[i], provider)
					}
				}
				if result.Provider.AllowFallbacks != tt.expectedFallback {
					t.Errorf("Provider.AllowFallbacks = %v, expected %v", result.Provider.AllowFallbacks, tt.expectedFallback)
				}
			} else if result.Provider != nil {
				t.Errorf("Expected Provider to be nil, got %+v", result.Provider)
			}
		})
	}
}

func TestGetProviderForModel(t *testing.T) {
	cfg := &config.Config{
		Model: "moonshotai/kimi-k2-0905",
		DefaultProvider: &config.ProviderConfig{
			Order:          []string{"Groq"},
			AllowFallbacks: false,
		},
		OpusProvider: &config.ProviderConfig{
			Order:          []string{"Anthropic"},
			AllowFallbacks: true,
		},
		SonnetProvider: &config.ProviderConfig{
			Order:          []string{"Anthropic", "OpenAI"},
			AllowFallbacks: true,
		},
	}

	tests := []struct {
		name           string
		model          string
		expectProvider bool
		expectedOrder  []string
	}{
		{
			name:           "opus model",
			model:          "claude-3-opus",
			expectProvider: true,
			expectedOrder:  []string{"Anthropic"},
		},
		{
			name:           "sonnet model",
			model:          "claude-3-5-sonnet",
			expectProvider: true,
			expectedOrder:  []string{"Anthropic", "OpenAI"},
		},
		{
			name:           "haiku model with no config",
			model:          "claude-3-haiku",
			expectProvider: false,
		},
		{
			name:           "direct model ID uses default",
			model:          "openai/gpt-4",
			expectProvider: true,
			expectedOrder:  []string{"Groq"},
		},
		{
			name:           "unknown model uses default",
			model:          "unknown-model",
			expectProvider: true,
			expectedOrder:  []string{"Groq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := GetProviderForModel(tt.model, cfg)

			if tt.expectProvider {
				if provider == nil {
					t.Fatal("Expected provider config, got nil")
				}
				if len(provider.Order) != len(tt.expectedOrder) {
					t.Errorf("Provider order length = %d, expected %d", len(provider.Order), len(tt.expectedOrder))
				}
				for i, expected := range tt.expectedOrder {
					if provider.Order[i] != expected {
						t.Errorf("Provider.Order[%d] = %q, expected %q", i, provider.Order[i], expected)
					}
				}
			} else if provider != nil {
				t.Errorf("Expected nil provider, got %+v", provider)
			}
		})
	}
}
