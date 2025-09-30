# Technical Design: OpenAI Proxy

## 1. Overview

### Feature Summary
A stateless HTTP proxy that translates Anthropic API requests to OpenRouter format, enabling seamless access to diverse AI models while maintaining Anthropic API compatibility.

### Architecture Pattern
**Stateless HTTP Proxy with Transformation Layer**
- Single binary with zero external dependencies
- Configuration-driven model mapping
- Strict API compatibility

### Implementation Status
**Production-Ready** - Fully implemented with comprehensive test coverage

## 2. Architecture

### System Architecture Diagram
```ascii
   ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
   │  Claude Code  │ ──► │  HTTP Server  │ ──► │ Transformer  │
   └───────────────┘     └───────────────┘     └───────────────┘
         ▲                                            │
         │                                            ▼
         │                                     ┌───────────────┐
         └─────────────────────────────────────┤  OpenRouter   │
                    Response                   └───────────────┘
```

### Request Flow
1. HTTP Handler receives POST /v1/messages
2. Validate AnthropicRequest
3. Transform to OpenAI/OpenRouter format
4. Forward to upstream API
5. Transform response back to Anthropic format
6. Return response to client

### Key Components
- **HTTP Server**: Request routing and handling
- **Request Transformer**: Format conversion logic
- **Streaming Handler**: SSE event processing
- **Configuration Manager**: Multi-source config loading

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
2. Config files (`athena.yml`, `athena.json`)
3. Environment variables
4. Built-in defaults

### Search Paths
- `~/.config/athena/athena.{yml,json}`
- `./athena.{yml,json}`
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
- Single binary, zero external dependencies
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