package transform

import (
	"encoding/json"
	"strings"

	"athena/internal/config"
)

// Constants for repeated strings
const (
	RoleAssistant = "assistant"
	RoleTool      = "tool"
	TypeToolUse   = "tool_use"
)

// AnthropicRequest represents the Anthropic Messages API request format
type AnthropicRequest struct {
	Model       string          `json:"model"`
	Messages    []Message       `json:"messages"`
	System      json.RawMessage `json:"system,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []Tool          `json:"tools,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// Tool represents a tool definition
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// OpenAIRequest represents the OpenAI/OpenRouter chat completions request format
type OpenAIRequest struct {
	Model       string                 `json:"model"`
	Messages    []OpenAIMessage        `json:"messages"`
	Temperature *float64               `json:"temperature,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Tools       []OpenAITool           `json:"tools,omitempty"`
	Provider    *config.ProviderConfig `json:"provider,omitempty"`
}

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool call in OpenAI format
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// OpenAITool represents a tool definition in OpenAI format
type OpenAITool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// ContentBlock represents a content block in Anthropic format
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
}

// ModelFormat identifies which tool calling response format OpenRouter will return
// based on the model being used
type ModelFormat int

const (
	FormatDeepSeek ModelFormat = iota // Standard OpenAI format
	FormatQwen                        // Hermes-style format
	FormatKimi                        // Special tokens format
	FormatStandard                    // Default OpenAI-compatible fallback
)

// String returns a human-readable format name for logging
func (f ModelFormat) String() string {
	switch f {
	case FormatDeepSeek:
		return "deepseek"
	case FormatQwen:
		return "qwen"
	case FormatKimi:
		return "kimi"
	default:
		return "standard"
	}
}

// Context encapsulates model format information and configuration
// for the transformation pipeline, passed through transformation functions
// instead of multiple parameters
type Context struct {
	Format ModelFormat    // The detected tool call format for this request based on model ID
	Config *config.Config // Reference to global configuration for model mappings
}

// StreamState consolidates all streaming state into a single struct to reduce
// parameter count from 8+ to 2 in processStreamDelta
type StreamState struct {
	ContentBlockIndex   int                  // Current content block index in Anthropic format
	HasStartedTextBlock bool                 // Whether a text content block has been started
	IsToolUse           bool                 // Whether currently processing tool calls
	CurrentToolCallID   string               // ID of the current tool call being streamed
	ToolCallJSONMap     map[string]string    // Accumulated JSON arguments per tool call ID
	FormatContext       *FormatStreamContext // Model format-specific streaming state
}

// FormatStreamContext isolates model format-specific streaming state
// (primarily Kimi K2 buffering) from general streaming state
type FormatStreamContext struct {
	Format            ModelFormat     // Which tool call format is being streamed
	KimiBuffer        strings.Builder // Buffer for Kimi K2 special tokens across chunks
	KimiBufferLimit   int             // Max buffer size (10KB)
	KimiInToolSection bool            // Whether currently inside tool_calls_section
}
