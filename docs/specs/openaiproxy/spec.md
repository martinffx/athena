# OpenAI Proxy Feature Specification

## Feature Overview
Athena provides a bidirectional API proxy that translates between Anthropic's native API format and OpenRouter's OpenAI-compatible format. This enables Claude Code to seamlessly interact with diverse AI model providers while maintaining full compatibility with native Anthropic API calling patterns.

## User Story
As a Claude Code user, I want to use the proxy to connect to OpenRouter with different model providers so that I can leverage OpenRouter's diverse model selection and competitive pricing while maintaining full compatibility with Claude Code's native API calling patterns.

## Acceptance Criteria

| Scenario | Expected Behavior |
|----------|-------------------|
| Request Transformation | GIVEN a valid Anthropic API request, WHEN the proxy transforms it to OpenAI format, THEN all message structure, content blocks, and metadata are correctly converted maintaining semantic equivalence |
| Response Mapping | GIVEN an OpenAI response from OpenRouter, WHEN the proxy transforms it back, THEN the response matches Anthropic format with proper field mapping |
| Streaming Processing | GIVEN a streaming request, WHEN the proxy processes SSE events from OpenRouter, THEN Anthropic-formatted events are emitted in correct order with <100ms latency overhead |
| Tool/Function Handling | GIVEN a request with tool/function definitions, WHEN the proxy transforms the request, THEN tool schemas are cleaned and tool calls are mapped correctly |
| Unsupported Features | GIVEN a request with unsupported features, WHEN the proxy validates the request, THEN it rejects with clear error messages |
| Workflow Compatibility | GIVEN Claude Code executing core workflows, WHEN using the proxy, THEN all operations succeed with identical behavior to native Anthropic API |

## Business Rules

1. **Strict Mode Operation**
   - Reject requests containing unsupported Anthropic features
   - Prevent forwarding of extended thinking, prompt caching, PDF input, and batch API requests

2. **Model Name Transformation**
   - Prefix all model names with 'anthropic/' when forwarding to OpenRouter
   - Example: 'claude-sonnet-4-20250514' → 'anthropic/claude-sonnet-4-20250514'

3. **System Prompt Management**
   - Convert system prompts to the first message with role='system' in the OpenAI messages array

4. **Error Handling**
   - Transform OpenRouter errors to match Anthropic's error response format
   - Preserve error type and provide clear, actionable messages

5. **Token Usage Mapping**
   - Rename usage statistics:
     - `prompt_tokens` → `input_tokens`
     - `completion_tokens` → `output_tokens`
   - Omit `total_tokens` from the response

6. **Streaming State Management**
   - Maintain proper state machine transitions
   - Emit events in correct order
   - Track content block indices accurately

7. **Tool Call Validation**
   - Validate all `tool_use` calls using `validateToolCalls()`
   - Ensure corresponding tool responses are present and correctly structured

8. **JSON Schema Cleaning**
   - Recursively remove unsupported 'format: uri' properties from tool schemas
   - Use `removeUriFormat()` to ensure OpenRouter compatibility

## Scope

### Included Features
- Bidirectional API translation (Anthropic ↔ OpenAI format)
- Non-streaming and streaming request/response handling
- Tool/function calling translation
- Multi-modal content support
- Error mapping and transformation
- Health check endpoint
- Multi-source configuration
- Request validation

### Excluded Features
- Extended thinking support
- Prompt caching
- Message Batches API
- PDF input support
- Advanced features planned for future phases (request queuing, multiple provider support, etc.)

## Technical Details

### Key Components
- `cmd/athena/main.go`: Application entry point
- `internal/cli/root.go`: Cobra CLI command setup
- `internal/server/server.go`: HTTP server with request handlers
- `internal/transform/transform.go`: Core transformation logic
- `internal/config/config.go`: Configuration management
- Model mapping system

### Data Models
- `AnthropicRequest`: Anthropic Messages API request format
- `AnthropicResponse`: Anthropic Messages API response format
- `OpenAIRequest`: OpenRouter/OpenAI chat completions request
- `ContentBlock`: Handles various content types

### Transformation Flow
1. Anthropic API Input
2. Transform to OpenAI Format (`AnthropicToOpenAI()`)
3. Forward to OpenRouter
4. Transform back to Anthropic Format (`OpenAIToAnthropic()`)
5. Return to Client

### Configuration
- Priority: CLI flags > config files > env vars > defaults
- Search paths:
  - `~/.config/athena/athena.{yml,json}`
  - `./athena.{yml,json}`
  - `./.env`

### Performance Targets
- Transformation Latency: <1ms
- Memory Allocation: <100KB per request
- Streaming Latency: <50ms first byte
- Throughput: 1000+ req/sec on standard hardware

## Dependencies
- Go standard library
- Cobra CLI framework (github.com/spf13/cobra v1.10.1)
- OpenRouter API
- Anthropic API specification
- OpenAI API specification

## Implementation Status
**Production-Ready**: All core features implemented, tested, and deployed.

## Notes
This specification documents the existing implementation of Athena's proxy functionality. Key architectural decisions include:
- Minimal external dependencies (Cobra CLI framework)
- Cobra-based CLI with clean package separation
- HTTP handlers in dedicated server package
- Strict API compatibility enforcement
- Configuration-driven model mapping