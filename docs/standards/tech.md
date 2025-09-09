# OpenRouter CC - Technical Standards

## Architecture Overview

### Core Design Principles
- **Simplicity**: Single binary with zero runtime dependencies
- **Reliability**: Robust error handling and graceful degradation
- **Performance**: Efficient request/response transformation and streaming
- **Maintainability**: Clear separation of concerns despite monolithic structure
- **Compatibility**: Full API compatibility with Anthropic Messages API

### System Architecture
```
Claude Code → OpenRouter CC Proxy → OpenRouter API → AI Model Providers
     ↑              ↓                    ↓                    ↓
Anthropic API   Translation Layer    OpenAI Format      Various Models
```

## Tech Stack

### Language & Runtime
- **Language**: Go 1.21+
- **Dependencies**: Standard library only (net/http, encoding/json, etc.)
- **Build**: Single static binary compilation
- **Platforms**: Linux, macOS, Windows (AMD64 + ARM64)

### API Standards
- **Input Format**: Anthropic Messages API v1 (messages, tools, streaming)
- **Output Format**: OpenRouter Chat Completions API (OpenAI-compatible)
- **Transport**: HTTP/1.1 with Server-Sent Events for streaming
- **Content Types**: application/json, text/event-stream

### Configuration Standards
- **Formats**: YAML (primary), JSON (secondary), Environment Variables
- **Precedence**: CLI flags → config files → env vars → defaults
- **Locations**: `~/.config/openrouter-cc/`, `./`, environment

## Data Flow Architecture

### Request Processing Pipeline
1. **HTTP Handler** (`/v1/messages`)
   - Request validation and parsing
   - Authentication header processing
   - Content-Type verification

2. **Message Transformation** (`transformAnthropicToOpenAI`)
   - System message extraction and positioning
   - Content block normalization (text, tool_use, tool_result)
   - Tool schema cleaning (remove unsupported format properties)

3. **Upstream Request** (`makeOpenRouterRequest`)
   - Model mapping resolution (claude-3-* → configured models)
   - Header propagation and API key management
   - Request serialization and transmission

4. **Response Processing**
   - **Non-streaming**: Direct JSON transformation
   - **Streaming**: SSE line buffering and event mapping
   - Error handling with proper status code mapping

### Streaming Architecture
```go
// Streaming response processing
func handleStreamingResponse(resp *http.Response, w http.ResponseWriter) {
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            // Transform OpenAI delta → Anthropic content_block_delta
            processSSELine(line, w)
        }
    }
}
```

## Component Architecture

### Core Components

#### 1. Configuration System
```go
type Config struct {
    Port        string `yaml:"port" json:"port"`
    BaseURL     string `yaml:"base_url" json:"base_url"`
    APIKey      string `yaml:"api_key" json:"api_key"`
    Model       string `yaml:"model" json:"model"`
    OpusModel   string `yaml:"opus_model" json:"opus_model"`
    SonnetModel string `yaml:"sonnet_model" json:"sonnet_model"`
    HaikuModel  string `yaml:"haiku_model" json:"haiku_model"`
}
```

#### 2. Message Transformation
- **AnthropicRequest** → **OpenAIRequest**: System message extraction, content normalization
- **OpenAIResponse** → **AnthropicResponse**: Content block reconstruction, tool call mapping
- **JSON Schema Cleaning**: Remove unsupported properties for compatibility

#### 3. Model Mapping Strategy
```go
func resolveModel(requested string, config Config) string {
    if strings.Contains(requested, "opus") return config.OpusModel
    if strings.Contains(requested, "sonnet") return config.SonnetModel
    if strings.Contains(requested, "haiku") return config.HaikuModel
    if strings.Contains(requested, "/") return requested // OpenRouter ID
    return config.Model // Default fallback
}
```

### Data Structures

#### Request/Response Types
```go
// Anthropic API format
type AnthropicRequest struct {
    Model    string         `json:"model"`
    Messages []Message      `json:"messages"`
    System   json.RawMessage `json:"system,omitempty"`
    Tools    []Tool         `json:"tools,omitempty"`
    Stream   bool           `json:"stream,omitempty"`
}

// OpenRouter/OpenAI format  
type OpenAIRequest struct {
    Model    string      `json:"model"`
    Messages []Message   `json:"messages"`
    Tools    []Tool      `json:"tools,omitempty"`
    Stream   bool        `json:"stream,omitempty"`
}

// Flexible content handling
type ContentBlock struct {
    Type     string          `json:"type"`
    Text     string          `json:"text,omitempty"`
    ToolUse  *ToolUse        `json:"tool_use,omitempty"`
    ToolResult *ToolResult   `json:"tool_result,omitempty"`
}
```

## Performance & Scalability

### Performance Characteristics
- **Memory**: Minimal allocation through json.RawMessage usage
- **CPU**: Efficient JSON parsing and transformation
- **Latency**: <50ms overhead for request/response transformation
- **Throughput**: Handles concurrent requests through Go's goroutine model

### Scalability Patterns
- **Stateless Design**: No server-side state between requests
- **Connection Pooling**: Reuse HTTP connections to upstream providers
- **Resource Efficiency**: Single binary with minimal memory footprint

### Optimization Strategies
- **JSON Processing**: Use json.RawMessage for pass-through data
- **String Operations**: Minimize string allocations in hot paths
- **Buffer Management**: Efficient SSE line buffering for streaming
- **Build Optimization**: Use `-ldflags="-s -w"` for binary size reduction

## Security Architecture

### Authentication Flow
- **Client Authentication**: X-Api-Key header passed through to OpenRouter
- **No Key Storage**: Proxy doesn't store or manage API keys
- **Transport Security**: HTTPS enforcement for production deployments

### Input Validation
- **Request Sanitization**: Validate JSON structure and required fields
- **Content Filtering**: Basic validation of message content and tools
- **Schema Enforcement**: Ensure proper tool schema format

### Error Handling
- **Information Disclosure**: Avoid exposing internal implementation details
- **Status Code Mapping**: Proper HTTP status codes for different error types
- **Upstream Error Propagation**: Pass through provider errors appropriately

## Development Standards

### Code Organization (Single File Structure)
```go
// main.go structure
package main

// Configuration and data structures
type Config struct { ... }
type AnthropicRequest struct { ... }

// Core transformation functions  
func transformAnthropicToOpenAI() { ... }
func transformOpenAIToAnthropic() { ... }

// HTTP handlers
func handleMessages() { ... }
func handleHealth() { ... }

// Utility functions
func loadConfig() { ... }
func validateToolCalls() { ... }

// Main entry point
func main() { ... }
```

### Testing Strategy
- **Unit Tests**: Test transformation functions with known input/output pairs
- **Integration Tests**: Test full request/response cycles with mock servers
- **Error Scenarios**: Test all error conditions and edge cases
- **Performance Tests**: Benchmark transformation performance and memory usage

### Build & Release
- **Cross Compilation**: Support 6 platforms (3 OS × 2 architectures)
- **GitHub Actions**: Automated testing, building, and releasing
- **Semantic Versioning**: Follow semver for releases and compatibility
- **Asset Distribution**: Binaries, wrapper scripts, and example configs