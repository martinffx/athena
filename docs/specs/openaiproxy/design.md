# Technical Design: OpenAI Proxy

## 1. Overview

### Feature Summary
A stateless HTTP proxy that translates Anthropic API requests to OpenRouter format, enabling seamless access to diverse AI models while maintaining Anthropic API compatibility.

### Architecture Pattern
**Stateless HTTP Proxy with Transformation Layer**
- Single binary with minimal external dependencies (Cobra CLI)
- Cobra-based command-line interface
- Configuration-driven model mapping
- Strict API compatibility

### Implementation Status
**Production-Ready** - Fully implemented with comprehensive test coverage

## 2. Architecture

### System Architecture Diagram
```ascii
   ┌───────────────┐     ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
   │  Claude Code  │ ──► │   Cobra CLI   │ ──► │  HTTP Server  │ ──► │ Transformer  │
   └───────────────┘     └───────────────┘     └───────────────┘     └───────────────┘
         ▲                                                                    │
         │                                                                    ▼
         │                                                             ┌───────────────┐
         └─────────────────────────────────────────────────────────────┤  OpenRouter   │
                                      Response                         └───────────────┘
```

### Request Flow
1. CLI entry point (cmd/athena/main.go) invokes Cobra command (internal/cli/root.go)
2. Server package (internal/server/server.go) handles HTTP requests
3. HTTP Handler receives POST /v1/messages
4. Validate AnthropicRequest
5. Transform to OpenAI/OpenRouter format (internal/transform/transform.go)
6. Forward to upstream API
7. Transform response back to Anthropic format
8. Return response to client

### Key Components
- **CLI Layer** (`cmd/athena/main.go`, `internal/cli/root.go`): Cobra command setup and configuration
- **HTTP Server** (`internal/server/server.go`): Request routing and handling
- **Request Transformer** (`internal/transform/transform.go`): Format conversion logic
- **Streaming Handler**: SSE event processing
- **Configuration Manager** (`internal/config/config.go`): Multi-source config loading

## 3. Domain Model

### Entities

#### AnthropicRequest
- **Purpose**: Parse and validate Anthropic API requests
- **Key Fields**:
  - `model`: Target AI model
  - `messages`: Conversation history
  - `system`: System prompt
  - `tools`: Function/tool definitions
- **Validation**: Detect unsupported features

#### OpenAIRequest
- **Purpose**: Construct OpenRouter-compatible requests
- **Key Fields**:
  - `model`: Mapped OpenRouter model
  - `messages`: Transformed message array
  - `tools`: Cleaned tool definitions

#### ContentBlock
- **Purpose**: Handle diverse message content types
- **Supported Types**:
  - `text`: Plain text content
  - `tool_use`: Function call request
  - `tool_result`: Function call result
  - `image`: Multi-modal image input

### Services

#### TransformationService
- **Core Functions**:
  - `AnthropicToOpenAI()`: Request transformation
  - `OpenAIToAnthropic()`: Response transformation
  - `MapModel()`: Dynamic model name resolution
  - `removeUriFormat()`: JSON schema cleaning
  - `validateToolCalls()`: Tool validation

#### StreamingService
- **Core Responsibilities**:
  - SSE event processing
  - Content block state tracking
  - Event generation for Anthropic format
  - Line-by-line SSE buffering

## 4. API Specification

### POST /v1/messages
- **Request Format**: Anthropic Messages API
- **Response Format**: Anthropic Messages API
- **Supported Features**:
  - Non-streaming responses
  - Server-Sent Events (SSE) streaming
  - Tool/function calling
  - Multi-modal content
  - System prompts
  - Temperature and token controls

#### Example Request
```json
{
  "model": "claude-3-sonnet",
  "messages": [{"role": "user", "content": "Hello"}],
  "max_tokens": 300,
  "stream": true
}
```

### GET /health
- **Purpose**: Service monitoring and health checks
- Returns JSON with:
  - Service status
  - Version
  - Uptime
  - Request metrics

## 5. Transformation Patterns

### System Message Handling
- **Input**: Anthropic system parameter (string)
- **Output**: First message with `role='system'`
- Extracted and prepended to messages array

### Content Normalization
- Supports both simple text strings and complex content block arrays
- Converts to normalized OpenAI content structure

### Tool Schema Cleaning
- Removes unsupported `format: uri` properties from tool definitions
- Enables cross-platform tool compatibility

### Model Mapping
- Dynamically resolves model names
- Supports configuration-driven mappings
- Example: `claude-sonnet` → `anthropic/claude-sonnet`

## 6. Streaming Architecture

### SSE Event Flow
1. Detect `stream=true`
2. Establish SSE connection
3. Process OpenAI SSE lines
4. Generate Anthropic-formatted events:
   - `message_start`
   - `content_block_start`
   - `content_block_delta`
   - `message_stop`

### State Machine
- Track message start/stop flags
- Monitor block indices
- Track input/output token counts

## 7. Configuration

### Sources (Precedence Order)
1. CLI flags
2. Config files (`athena.yml`)
3. Environment variables
4. Built-in defaults

### Search Paths
- `~/.config/athena/athena.yml`
- `./athena.yml`
- `./.env`

### Key Parameters
- `port`: HTTP server port
- `base_url`: OpenRouter API endpoint
- `api_key`: Authentication
- Model mappings: `opus_model`, `sonnet_model`, `haiku_model`

## 8. Performance Characteristics

### Targets
- Transformation Latency: < 1ms
- Memory Allocation: < 100KB per request
- Streaming First Byte: < 50ms
- Throughput: 1000+ requests/second

### Optimization Techniques
- Minimal string allocations
- `json.RawMessage` for efficient parsing
- Stateless design for concurrency
- Efficient SSE buffering

## 9. Implementation Notes

### Architectural Decisions
- Single binary with minimal external dependencies (Cobra CLI framework)
- Cobra-based CLI for flexible command structure
- HTTP handlers in dedicated server package
- Stateless design for horizontal scaling
- Configuration-driven model mapping
- Strict API compatibility
- Functional transformation approach

### Key Challenges Solved
- Streaming state management via tracked indices
- Tool schema cleaning with recursive processing
- System message positioning
- Error format preservation

### Testing Strategy
- Unit tests for transformation functions
- Integration tests for request/response cycles
- Streaming event generation tests
- Comprehensive error scenario coverage
- Performance benchmarks

**Status**: Production-Ready Implementation