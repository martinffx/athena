package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AnthropicRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	System      json.RawMessage `json:"system,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
}

type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type OpenAIRequest struct {
	Model       string         `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature *float64       `json:"temperature,omitempty"`
	Stream      bool           `json:"stream,omitempty"`
	Tools       []OpenAITool   `json:"tools,omitempty"`
}

type OpenAIMessage struct {
	Role       string         `json:"role"`
	Content    interface{}    `json:"content"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type ContentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
}

type Config struct {
	Port        string `json:"port"`
	APIKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	OpusModel   string `json:"opus_model"`
	SonnetModel string `json:"sonnet_model"`
	HaikuModel  string `json:"haiku_model"`
}

var config Config

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (JSON)")
	flag.StringVar(&config.Port, "port", "", "Port to run the server on")
	flag.StringVar(&config.APIKey, "api-key", "", "OpenRouter API key")
	flag.StringVar(&config.BaseURL, "base-url", "", "OpenRouter base URL")
	flag.StringVar(&config.Model, "model", "", "Default model to use")
	flag.StringVar(&config.OpusModel, "opus-model", "", "Model to map claude-opus requests to")
	flag.StringVar(&config.SonnetModel, "sonnet-model", "", "Model to map claude-sonnet requests to")
	flag.StringVar(&config.HaikuModel, "haiku-model", "", "Model to map claude-haiku requests to")
	flag.Parse()

	loadConfig(configFile)

	if config.APIKey == "" {
		log.Fatal("OpenRouter API key is required. Use -api-key flag, config file, or OPENROUTER_API_KEY env var")
	}

	http.HandleFunc("/v1/messages", handleMessages)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Printf("Starting server on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func loadConfig(configFile string) {
	// Load .env file if it exists
	loadEnvFile(".env")
	
	// Set defaults
	if config.Port == "" {
		config.Port = getEnvWithDefault("PORT", "11434")
	}
	if config.APIKey == "" {
		config.APIKey = getEnvWithDefault("OPENROUTER_API_KEY", "")
	}
	if config.BaseURL == "" {
		config.BaseURL = getEnvWithDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api")
	}
	if config.Model == "" {
		config.Model = getEnvWithDefault("DEFAULT_MODEL", "google/gemini-2.0-flash-exp:free")
	}
	if config.OpusModel == "" {
		config.OpusModel = getEnvWithDefault("OPUS_MODEL", "anthropic/claude-3-opus")
	}
	if config.SonnetModel == "" {
		config.SonnetModel = getEnvWithDefault("SONNET_MODEL", "anthropic/claude-3.5-sonnet")
	}
	if config.HaikuModel == "" {
		config.HaikuModel = getEnvWithDefault("HAIKU_MODEL", "anthropic/claude-3.5-haiku")
	}

	// Load config file if specified
	if configFile != "" {
		loadConfigFromFile(configFile)
	} else {
		// Try to load from default locations
		homeDir, _ := os.UserHomeDir()
		configPaths := []string{
			filepath.Join(homeDir, ".config", "openrouter-cc", "openrouter.yml"),
			filepath.Join(homeDir, ".config", "openrouter-cc", "openrouter.json"),
			"openrouter.yml",
			"openrouter.json",
		}

		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				loadConfigFromFile(path)
				break
			}
		}
	}
}

func loadConfigFromFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", filename, err)
		return
	}

	var fileConfig Config
	
	if strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml") {
		if err := parseYAML(data, &fileConfig); err != nil {
			log.Printf("Warning: Could not parse YAML config file %s: %v", filename, err)
			return
		}
	} else {
		if err := json.Unmarshal(data, &fileConfig); err != nil {
			log.Printf("Warning: Could not parse JSON config file %s: %v", filename, err)
			return
		}
	}

	log.Printf("Loaded config from %s", filename)

	// Override with config file values only if they're not empty
	if fileConfig.Port != "" && config.Port == getEnvWithDefault("PORT", "8080") {
		config.Port = fileConfig.Port
	}
	if fileConfig.APIKey != "" && config.APIKey == "" {
		config.APIKey = fileConfig.APIKey
	}
	if fileConfig.BaseURL != "" && config.BaseURL == getEnvWithDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api") {
		config.BaseURL = fileConfig.BaseURL
	}
	if fileConfig.Model != "" && config.Model == getEnvWithDefault("DEFAULT_MODEL", "google/gemini-2.0-flash-exp:free") {
		config.Model = fileConfig.Model
	}
	if fileConfig.OpusModel != "" && config.OpusModel == getEnvWithDefault("OPUS_MODEL", "anthropic/claude-3-opus") {
		config.OpusModel = fileConfig.OpusModel
	}
	if fileConfig.SonnetModel != "" && config.SonnetModel == getEnvWithDefault("SONNET_MODEL", "anthropic/claude-3.5-sonnet") {
		config.SonnetModel = fileConfig.SonnetModel
	}
	if fileConfig.HaikuModel != "" && config.HaikuModel == getEnvWithDefault("HAIKU_MODEL", "anthropic/claude-3.5-haiku") {
		config.HaikuModel = fileConfig.HaikuModel
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseYAML(data []byte, config *Config) error {
	lines := strings.Split(string(data), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		switch key {
		case "port":
			config.Port = value
		case "api_key":
			config.APIKey = value
		case "base_url":
			config.BaseURL = value
		case "model":
			config.Model = value
		case "opus_model":
			config.OpusModel = value
		case "sonnet_model":
			config.SonnetModel = value
		case "haiku_model":
			config.HaikuModel = value
		}
	}
	
	return nil
}

func loadEnvFile(filename string) {
	if _, err := os.Stat(filename); err != nil {
		return // File doesn't exist, skip silently
	}
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}
		
		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	
	log.Printf("Loaded environment variables from %s", filename)
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var anthropicReq AnthropicRequest
	if err := json.NewDecoder(r.Body).Decode(&anthropicReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	openAIReq := transformAnthropicToOpenAI(anthropicReq)

	bearerToken := r.Header.Get("X-Api-Key")
	if bearerToken == "" {
		if auth := r.Header.Get("Authorization"); auth != "" {
			bearerToken = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	if bearerToken == "" {
		bearerToken = config.APIKey
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	openAIResp, err := http.Post(
		fmt.Sprintf("%s/v1/chat/completions", config.BaseURL),
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		http.Error(w, "Failed to call OpenRouter", http.StatusInternalServerError)
		return
	}
	defer openAIResp.Body.Close()

	openAIResp.Header.Set("Authorization", "Bearer "+bearerToken)

	if !openAIReq.Stream {
		handleNonStreamingResponse(w, openAIResp, openAIReq.Model)
	} else {
		handleStreamingResponse(w, openAIResp, openAIReq.Model)
	}
}

func transformAnthropicToOpenAI(req AnthropicRequest) OpenAIRequest {
	messages := []OpenAIMessage{}

	// Handle system messages
	if len(req.System) > 0 {
		var systemContent interface{}
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
		_ = systemContent
	}

	// Transform messages
	for _, msg := range req.Messages {
		openAIMsgs := transformMessage(msg)
		messages = append(messages, openAIMsgs...)
	}

	// Validate tool calls
	messages = append(messages[:len(messages)-len(messages)+len(messages[:0])], validateToolCalls(messages)...)

	result := OpenAIRequest{
		Model:       mapModel(req.Model),
		Messages:    messages,
		Temperature: req.Temperature,
		Stream:      req.Stream,
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

	if msg.Role == "assistant" {
		assistantMsg := OpenAIMessage{
			Role:    "assistant",
			Content: nil,
		}
		textContent := ""
		toolCalls := []ToolCall{}

		for _, block := range content {
			if block.Type == "text" {
				textContent += block.Text + "\n"
			} else if block.Type == "tool_use" {
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
	} else if msg.Role == "user" {
		userText := ""
		toolMessages := []OpenAIMessage{}

		for _, block := range content {
			if block.Type == "text" {
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

func validateToolCalls(messages []OpenAIMessage) []OpenAIMessage {
	validated := []OpenAIMessage{}

	for i, msg := range messages {
		currentMsg := msg

		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			validToolCalls := []ToolCall{}
			
			// Collect immediately following tool messages
			immediateTools := []OpenAIMessage{}
			j := i + 1
			for j < len(messages) && messages[j].Role == "tool" {
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
		} else if msg.Role == "tool" {
			// Check if previous message has matching tool call
			hasMatch := false
			if i > 0 {
				prevMsg := messages[i-1]
				if prevMsg.Role == "assistant" {
					for _, toolCall := range prevMsg.ToolCalls {
						if toolCall.ID == msg.ToolCallID {
							hasMatch = true
							break
						}
					}
				} else if prevMsg.Role == "tool" {
					// Check for assistant message before tool sequence
					for k := i - 1; k >= 0; k-- {
						if messages[k].Role == "tool" {
							continue
						}
						if messages[k].Role == "assistant" {
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

func mapModel(anthropicModel string) string {
	if strings.Contains(anthropicModel, "/") {
		return anthropicModel
	}

	if strings.Contains(anthropicModel, "haiku") {
		return config.HaikuModel
	} else if strings.Contains(anthropicModel, "sonnet") {
		return config.SonnetModel
	} else if strings.Contains(anthropicModel, "opus") {
		return config.OpusModel
	}

	return config.Model // Use default model
}

func removeUriFormat(schema json.RawMessage) json.RawMessage {
	var data interface{}
	if err := json.Unmarshal(schema, &data); err != nil {
		return schema
	}

	cleaned := removeUriFormatFromInterface(data)
	result, _ := json.Marshal(cleaned)
	return result
}

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

func handleNonStreamingResponse(w http.ResponseWriter, resp *http.Response, modelName string) {
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

	anthropicResp := transformOpenAIToAnthropic(openAIResp, modelName)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anthropicResp)
}

func transformOpenAIToAnthropic(resp map[string]interface{}, modelName string) map[string]interface{} {
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
					json.Unmarshal([]byte(args), &input)
				}
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    toolCall["id"],
					"name":  function["name"],
					"input": input,
				})
			}
		}
		
		finishReason := choice["finish_reason"].(string)
		stopReason := "end_turn"
		if finishReason == "tool_calls" {
			stopReason = "tool_use"
		}

		return map[string]interface{}{
			"id":           messageID,
			"type":         "message",
			"role":         "assistant",
			"content":      content,
			"stop_reason":  stopReason,
			"stop_sequence": nil,
			"model":        modelName,
		}
	}

	return map[string]interface{}{
		"id":           messageID,
		"type":         "message",
		"role":         "assistant",
		"content":      content,
		"stop_reason":  "end_turn",
		"stop_sequence": nil,
		"model":        modelName,
	}
}

func handleStreamingResponse(w http.ResponseWriter, resp *http.Response, modelName string) {
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
			"id":           messageID,
			"type":         "message",
			"role":         "assistant",
			"content":      []interface{}{},
			"model":        modelName,
			"stop_reason":  nil,
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
	stopReason := "end_turn"
	if isToolUse {
		stopReason = "tool_use"
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

func processStreamDelta(w http.ResponseWriter, flusher http.Flusher, delta map[string]interface{},
	contentBlockIndex *int, hasStartedTextBlock *bool, isToolUse *bool,
	currentToolCallID *string, toolCallJSONMap map[string]string) {

	// Handle tool calls
	if toolCalls, ok := delta["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		for _, tc := range toolCalls {
			toolCall := tc.(map[string]interface{})
			if id, ok := toolCall["id"].(string); ok && id != *currentToolCallID {
				// Close previous block if exists
				if *isToolUse || *hasStartedTextBlock {
					sendSSE(w, flusher, "content_block_stop", map[string]interface{}{
						"type":  "content_block_stop",
						"index": *contentBlockIndex,
					})
				}

				*isToolUse = true
				*hasStartedTextBlock = false
				*currentToolCallID = id
				*contentBlockIndex++
				toolCallJSONMap[id] = ""

				var name string
				if function, ok := toolCall["function"].(map[string]interface{}); ok {
					if n, ok := function["name"].(string); ok {
						name = n
					}
				}

				sendSSE(w, flusher, "content_block_start", map[string]interface{}{
					"type":  "content_block_start",
					"index": *contentBlockIndex,
					"content_block": map[string]interface{}{
						"type":  "tool_use",
						"id":    id,
						"name":  name,
						"input": map[string]interface{}{},
					},
				})
			}

			if function, ok := toolCall["function"].(map[string]interface{}); ok {
				if args, ok := function["arguments"].(string); ok && *currentToolCallID != "" {
					toolCallJSONMap[*currentToolCallID] += args
					sendSSE(w, flusher, "content_block_delta", map[string]interface{}{
						"type":  "content_block_delta",
						"index": *contentBlockIndex,
						"delta": map[string]interface{}{
							"type":         "input_json_delta",
							"partial_json": args,
						},
					})
				}
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

func sendSSE(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	flusher.Flush()
}