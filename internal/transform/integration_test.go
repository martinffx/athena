package transform

import (
	"encoding/json"
	"testing"

	"athena/internal/config"
)

// TestIntegration_KimiFlow tests the complete flow for Kimi models:
// Anthropic request → OpenRouter → Kimi response → Anthropic format
func TestIntegration_KimiFlow(t *testing.T) {
	// Step 1: Create Anthropic request with tools
	anthropicReq := AnthropicRequest{
		Model: "kimi-k2",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"What's the weather in Tokyo?"`),
			},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get weather for a city",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
			},
		},
	}

	cfg := &config.Config{
		Model: "moonshot/kimi-k2-0905",
	}

	// Step 2: Transform to OpenAI format
	openAIReq := AnthropicToOpenAI(anthropicReq, cfg)

	// Verify transformation
	if openAIReq.Model != "moonshot/kimi-k2-0905" {
		t.Errorf("Expected model moonshot/kimi-k2-0905, got %s", openAIReq.Model)
	}

	if len(openAIReq.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(openAIReq.Tools))
	}

	if openAIReq.Tools[0].Function.Name != "get_weather" {
		t.Errorf("Expected tool name get_weather, got %s", openAIReq.Tools[0].Function.Name)
	}

	// Step 3: Simulate Kimi response with special tokens
	kimiResponse := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:1<|tool_call_argument_begin|>{"city":"Tokyo"}<|tool_call_end|>
<|tool_calls_section_end|>`,
				},
				"finish_reason": "stop",
			},
		},
	}

	// Step 4: Transform back to Anthropic format with Kimi format detection
	anthropicResp, err := OpenAIToAnthropic(kimiResponse, "moonshot/kimi-k2-0905", FormatKimi)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() unexpected error: %v", err)
	}

	// Step 5: Verify Anthropic response
	if anthropicResp["type"] != "message" {
		t.Errorf("Expected type message, got %v", anthropicResp["type"])
	}

	if anthropicResp["role"] != testRoleAssistant {
		t.Errorf("Expected role assistant, got %v", anthropicResp["role"])
	}

	if anthropicResp["stop_reason"] != TypeToolUse {
		t.Errorf("Expected stop_reason tool_use, got %v", anthropicResp["stop_reason"])
	}

	content, ok := anthropicResp["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Content is not an array of maps")
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != TypeToolUse {
		t.Errorf("Expected content type tool_use, got %v", content[0]["type"])
	}

	if content[0]["name"] != testFuncGetWeather {
		t.Errorf("Expected tool name get_weather, got %v", content[0]["name"])
	}

	if content[0]["id"] != "functions.get_weather:1" {
		t.Errorf("Expected tool ID functions.get_weather:1, got %v", content[0]["id"])
	}

	input, ok := content[0]["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool input is not a map")
	}

	if input["city"] != "Tokyo" {
		t.Errorf("Expected city Tokyo, got %v", input["city"])
	}
}

// TestIntegration_QwenFlow_vLLM tests Qwen with vLLM format (tool_calls array)
func TestIntegration_QwenFlow_vLLM(t *testing.T) {
	// Step 1: Create Anthropic request
	anthropicReq := AnthropicRequest{
		Model: "qwen-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Search for Python tutorials"`),
			},
		},
		Tools: []Tool{
			{
				Name:        "search",
				Description: "Search the web",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
			},
		},
	}

	cfg := &config.Config{
		Model: "qwen/qwen-2.5-72b-instruct",
	}

	// Step 2: Transform to OpenAI format
	openAIReq := AnthropicToOpenAI(anthropicReq, cfg)

	if openAIReq.Model != "qwen/qwen-2.5-72b-instruct" {
		t.Errorf("Expected model qwen/qwen-2.5-72b-instruct, got %s", openAIReq.Model)
	}

	// Step 3: Simulate Qwen vLLM response (tool_calls array)
	qwenResponse := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": nil,
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call-abc123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "search",
								"arguments": `{"query":"Python tutorials"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}

	// Step 4: Transform back to Anthropic format
	anthropicResp, err := OpenAIToAnthropic(qwenResponse, "qwen/qwen-2.5-72b-instruct", FormatQwen)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() unexpected error: %v", err)
	}

	// Step 5: Verify response
	if anthropicResp["stop_reason"] != TypeToolUse {
		t.Errorf("Expected stop_reason tool_use, got %v", anthropicResp["stop_reason"])
	}

	content, ok := anthropicResp["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Content is not an array of maps")
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != TypeToolUse {
		t.Errorf("Expected content type tool_use, got %v", content[0]["type"])
	}

	if content[0]["name"] != "search" {
		t.Errorf("Expected tool name search, got %v", content[0]["name"])
	}

	input, ok := content[0]["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool input is not a map")
	}

	if input["query"] != "Python tutorials" {
		t.Errorf("Expected query 'Python tutorials', got %v", input["query"])
	}
}

// TestIntegration_QwenFlow_Agent tests Qwen with Qwen-Agent format (function_call object)
func TestIntegration_QwenFlow_Agent(t *testing.T) {
	// Step 1: Create Anthropic request
	anthropicReq := AnthropicRequest{
		Model: "qwen-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Get weather for Beijing"`),
			},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get weather",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		},
	}

	cfg := &config.Config{
		Model: "qwen/qwen3-coder",
	}

	// Step 2: Transform to OpenAI format
	openAIReq := AnthropicToOpenAI(anthropicReq, cfg)

	if openAIReq.Model != "qwen/qwen3-coder" {
		t.Errorf("Expected model qwen/qwen3-coder, got %s", openAIReq.Model)
	}

	// Step 3: Simulate Qwen-Agent response (function_call object)
	qwenResponse := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": nil,
					"function_call": map[string]interface{}{
						"name":      "get_weather",
						"arguments": `{"city":"Beijing"}`,
					},
				},
				"finish_reason": "function_call",
			},
		},
	}

	// Step 4: Transform back to Anthropic format with Qwen format
	anthropicResp, err := OpenAIToAnthropic(qwenResponse, "qwen/qwen3-coder", FormatQwen)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() unexpected error: %v", err)
	}

	// Step 5: Verify response
	if anthropicResp["stop_reason"] != TypeToolUse {
		t.Errorf("Expected stop_reason tool_use, got %v", anthropicResp["stop_reason"])
	}

	content, ok := anthropicResp["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Content is not an array of maps")
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	if content[0]["type"] != TypeToolUse {
		t.Errorf("Expected content type tool_use, got %v", content[0]["type"])
	}

	if content[0]["name"] != testFuncGetWeather {
		t.Errorf("Expected tool name get_weather, got %v", content[0]["name"])
	}

	// Verify synthetic ID was generated
	if _, idOk := content[0]["id"].(string); !idOk {
		t.Errorf("Expected tool ID to be a string, got %T", content[0]["id"])
	}

	input, ok := content[0]["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool input is not a map")
	}

	if input["city"] != "Beijing" {
		t.Errorf("Expected city Beijing, got %v", input["city"])
	}
}

// TestIntegration_StandardFlow tests standard OpenAI format (DeepSeek baseline)
func TestIntegration_StandardFlow(t *testing.T) {
	// Step 1: Create Anthropic request
	anthropicReq := AnthropicRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Calculate 42 * 13"`),
			},
		},
		Tools: []Tool{
			{
				Name:        "calculator",
				Description: "Perform calculations",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"expression":{"type":"string"}}}`),
			},
		},
	}

	cfg := &config.Config{
		Model: "deepseek/deepseek-chat",
	}

	// Step 2: Transform to OpenAI format
	openAIReq := AnthropicToOpenAI(anthropicReq, cfg)

	if openAIReq.Model != "deepseek/deepseek-chat" {
		t.Errorf("Expected model deepseek/deepseek-chat, got %s", openAIReq.Model)
	}

	// Step 3: Simulate standard OpenAI response
	standardResponse := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"content": "I'll calculate that for you.",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_xyz",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "calculator",
								"arguments": `{"expression":"42 * 13"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}

	// Step 4: Transform back to Anthropic format
	anthropicResp, err := OpenAIToAnthropic(standardResponse, "deepseek/deepseek-chat", FormatStandard)
	if err != nil {
		t.Fatalf("OpenAIToAnthropic() unexpected error: %v", err)
	}

	// Step 5: Verify response
	if anthropicResp["stop_reason"] != TypeToolUse {
		t.Errorf("Expected stop_reason tool_use, got %v", anthropicResp["stop_reason"])
	}

	content, ok := anthropicResp["content"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Content is not an array of maps")
	}

	// Should have text block + tool use block
	if len(content) != 2 {
		t.Fatalf("Expected 2 content blocks (text + tool_use), got %d", len(content))
	}

	// Verify text block
	if content[0]["type"] != "text" {
		t.Errorf("Expected first block type text, got %v", content[0]["type"])
	}

	if content[0]["text"] != "I'll calculate that for you." {
		t.Errorf("Expected text 'I'll calculate that for you.', got %v", content[0]["text"])
	}

	// Verify tool use block
	if content[1]["type"] != TypeToolUse {
		t.Errorf("Expected second block type tool_use, got %v", content[1]["type"])
	}

	if content[1]["name"] != "calculator" {
		t.Errorf("Expected tool name calculator, got %v", content[1]["name"])
	}

	input, ok := content[1]["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Tool input is not a map")
	}

	if input["expression"] != "42 * 13" {
		t.Errorf("Expected expression '42 * 13', got %v", input["expression"])
	}
}

// TestIntegration_MultiTurnConversation tests multi-turn conversation with tool results
func TestIntegration_MultiTurnConversation(t *testing.T) {
	cfg := &config.Config{
		Model: "deepseek/deepseek-chat",
	}

	// Step 1: Initial request
	anthropicReq := AnthropicRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"What's the weather?"`),
			},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get weather",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		},
	}

	openAIReq := AnthropicToOpenAI(anthropicReq, cfg)
	if len(openAIReq.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(openAIReq.Messages))
	}

	// Step 2: Follow-up with tool result
	anthropicReq2 := AnthropicRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"What's the weather?"`),
			},
			{
				Role: "assistant",
				Content: json.RawMessage(`[
					{"type":"tool_use","id":"call_123","name":"get_weather","input":{"city":"Tokyo"}}
				]`),
			},
			{
				Role: "user",
				Content: json.RawMessage(`[
					{"type":"tool_result","tool_use_id":"call_123","content":"Sunny, 25°C"}
				]`),
			},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get weather",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
			},
		},
	}

	openAIReq2 := AnthropicToOpenAI(anthropicReq2, cfg)

	// Verify multi-turn transformation
	if len(openAIReq2.Messages) < 3 {
		t.Fatalf("Expected at least 3 messages (user, assistant with tool, tool result), got %d", len(openAIReq2.Messages))
	}

	// Find the tool role message
	var foundToolMessage bool
	for _, msg := range openAIReq2.Messages {
		if msg.Role == RoleTool {
			foundToolMessage = true
			if msg.ToolCallID != "call_123" {
				t.Errorf("Expected tool_call_id call_123, got %s", msg.ToolCallID)
			}
			if content, ok := msg.Content.(string); ok {
				if content != "Sunny, 25°C" {
					t.Errorf("Expected tool content 'Sunny, 25°C', got %s", content)
				}
			}
			break
		}
	}

	if !foundToolMessage {
		t.Error("Expected to find a tool role message in the conversation")
	}
}
