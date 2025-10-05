# Tool Calling Feature Specification

## Feature Overview

**Feature Name:** Tool Calling Provider Translation

**Summary:** Enhance Athena's proxy capabilities to reliably translate tool calling between Anthropic and OpenRouter formats, supporting provider-specific quirks for DeepSeek, Qwen3-Coder, and Kimi K2 models. This enables Claude Code users to leverage alternative models with function calling capabilities while maintaining transparent compatibility with the Anthropic API format.

**Problem Context:** Open-source models use incompatible tool calling formats that break in production. Kimi K2 outputs raw special tokens, Qwen uses Hermes-style parsing, while DeepSeek follows OpenAI standards. Third-party providers fail to parse non-standard formats correctly, causing silent failures in agentic workflows. Tool calling failures compound across multi-turn conversations, making single-request benchmarks misleading for real-world applications.

**Impact:** Claude Code users cannot reliably switch between models without rewriting integration code. This feature enables model diversity while maintaining full compatibility with Claude Code's tool calling workflows.

## User Story

As an Athena user, I want the proxy to reliably translate tool calling between Anthropic and OpenRouter formats so that I can use Claude Code with alternative models (DeepSeek, Qwen3-Coder, Kimi K2) that support function calling.

## Acceptance Criteria

1. **Tool Schema Translation**
   - GIVEN an Anthropic API request with tool definitions
   - WHEN the request is proxied to OpenRouter
   - THEN tool schemas are correctly translated to OpenAI format with unsupported properties removed

2. **Tool Use Block Transformation**
   - GIVEN a tool_use content block in Anthropic format
   - WHEN transformed to OpenAI format
   - THEN it becomes a tool_call with proper ID, function name, and arguments

3. **Tool Result Block Transformation**
   - GIVEN a tool_result content block in Anthropic format
   - WHEN transformed to OpenAI format
   - THEN it becomes a message with role='tool' and matching tool_call_id

4. **Streaming Tool Call Events**
   - GIVEN an OpenRouter streaming response with tool call deltas
   - WHEN processed by the streaming handler
   - THEN content_block_start, content_block_delta, and content_block_stop events are emitted in Anthropic SSE format

5. **Tool Call Validation**
   - GIVEN a multi-turn conversation with tool calls and responses
   - WHEN validateToolCalls() is invoked
   - THEN all tool_use blocks have matching tool_result blocks

6. **Kimi K2 Provider Support**
   - GIVEN a Kimi K2 model request
   - WHEN provider-specific handling is applied
   - THEN special tokens are properly managed in tool call translation

7. **Qwen3-Coder Provider Support**
   - GIVEN a Qwen3-Coder model request
   - WHEN provider-specific handling is applied
   - THEN Hermes-style tool calling format is used

8. **DeepSeek Provider Support**
   - GIVEN a DeepSeek model request
   - WHEN provider-specific handling is applied
   - THEN standard OpenAI tool calling format is used

9. **Error Handling**
   - System shall return proper error responses to the client when tool call translation or validation fails

10. **Provider Detection Ambiguity**
    - GIVEN a model name containing multiple provider keywords or no clear provider match
    - WHEN provider detection is performed
    - THEN system uses precedence order (Kimi > Qwen > DeepSeek) or falls back to standard OpenAI format

11. **Streaming Failure Recovery**
    - GIVEN a partial tool call chunk that fails provider-specific transformation
    - WHEN streaming error is detected
    - THEN system sends error SSE event and gracefully terminates stream with appropriate error message

12. **Provider Format Validation**
    - GIVEN a tool call transformation to provider-specific format
    - WHEN validation is performed before sending to OpenRouter
    - THEN malformed provider formats are rejected with clear error messages (HTTP 400)

## Business Rules

1. **Multi-Model Support**: All three target models must be supported: DeepSeek (standard OpenAI format), Qwen3-Coder (Hermes-style), Kimi K2 (special tokens)

2. **Transparency**: Tool calling translation must be transparent to the Claude Code client

3. **Provider Detection**: Provider-specific quirks must be detected based on model name/identifier in the request

4. **Streaming State Consistency**: Streaming tool calls must maintain state consistency across SSE events

5. **Tool Call Validation**: Tool call validation must ensure every tool_use has a corresponding tool_result in multi-turn conversations

6. **Schema Cleaning**: JSON schema cleaning must continue to remove unsupported properties like 'format: uri'

7. **Error Propagation**: Error conditions in tool translation must propagate proper HTTP status codes to the client

8. **Provider Precedence**: Provider detection must use precedence order (Kimi > Qwen > DeepSeek > Standard) when model name is ambiguous or contains multiple provider keywords

9. **Streaming Buffer Limits**: Streaming tool calls must buffer provider-specific tokens up to 10KB before erroring to prevent memory exhaustion

10. **Pre-Send Validation**: Provider format validation failures must return HTTP 400 with error details before sending to OpenRouter to prevent unnecessary upstream requests

## Scope

### Included
- Provider detection logic based on model identifier
- Provider-specific tool call format transformations (Kimi K2 special tokens, Qwen Hermes-style, DeepSeek standard)
- Enhanced streaming support for provider-specific tool call formats
- Integration with existing transformAnthropicToOpenAI() and OpenAIToAnthropic() functions
- Integration with existing validateToolCalls() function
- Testing with all three target models (DeepSeek, Qwen3-Coder, Kimi K2)

### Excluded
- Custom tool calling implementations beyond format translation
- Fixes for provider-side reliability issues (external to Athena)
- Retry logic or error recovery for failed tool calls
- Tool execution (only translation between API formats)
- Support for additional models beyond DeepSeek, Qwen3-Coder, and Kimi K2 in initial implementation
- Real-time performance matching native implementations (acceptable <1ms transformation overhead)
- Fixing underlying model issues (that's on the providers)
- Supporting every possible edge case on day one
- Request-side transformation (all models accept OpenAI tool definitions)

## Alignment with Product Vision

Enhances Athena's core value proposition of enabling Claude Code users to access diverse AI models by ensuring tool calling works transparently across providers with different implementation quirks, maintaining full compatibility with Claude Code workflows while extending model choice.

## Dependencies

- `transform.AnthropicToOpenAI()` - Main transformation function that needs provider-specific logic
- `transform.transformMessage()` - Message conversion that handles tool_use and tool_result blocks
- `transform.validateToolCalls()` - Tool call validation ensuring matching responses
- `transform.OpenAIToAnthropic()` - Response transformation back to Anthropic format
- `transform.HandleStreaming()` - Streaming handler for SSE processing
- `transform.processStreamDelta()` - Processes streaming deltas including tool call chunks
- `transform.removeUriFormat()` - JSON schema cleaning function
- `AnthropicRequest.Tools` - Tool definition structure
- `OpenAIMessage.ToolCalls` - OpenAI tool call structure
- `ContentBlock` - Handles tool_use and tool_result content types
- `Config.Model`, `Config.OpusModel`, `Config.SonnetModel`, `Config.HaikuModel` - Model mapping for provider detection

## Technical Details

### 1. Provider Detection

**Description:** Identify which model/provider is being used to apply appropriate tool calling format

**Implementation Notes:**
- Add provider detection function that inspects the resolved model name (after model mapping) and returns provider type enum (DeepSeek, Qwen, Kimi)
- Pattern match on model strings: contains 'deepseek' → DeepSeek, contains 'qwen' → Qwen, contains 'kimi' or 'k2' → Kimi
- Function should be called early in the transformation pipeline to inform subsequent translation logic

### 2. Kimi K2 Special Token Handling

**Description:** Parse Kimi-specific special tokens from tool call responses

**Implementation Notes:**
- **Special Tokens Format** (see provider-formats.md for details):
  - `<|tool_calls_section_begin|>` - Start of tool calls section
  - `<|tool_calls_section_end|>` - End of tool calls section
  - `<|tool_call_begin|>` - Start of individual tool call
  - `<|tool_call_end|>` - End of individual tool call
  - `<|tool_call_argument_begin|>` - Separator between tool ID and arguments
- **Tool ID Format**: `functions.{func_name}:{idx}` (e.g., `functions.get_weather:0`)
- **Raw Output Example**:
  ```
  <|tool_calls_section_begin|>
  <|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
  <|tool_calls_section_end|>
  ```
- **Parsing Logic**: Use regex pattern to extract tool calls from special token format and convert to OpenAI-compatible structure
- **Streaming Consideration**: Special tokens may be split across chunks, requiring buffering until `<|tool_calls_section_end|>` is received
- **Note**: This is a **response-side transformation only** - Kimi accepts standard OpenAI tool definitions in requests

### 3. Qwen Hermes-Style Format

**Description:** Handle Hermes-style tool call format for Qwen3-Coder

**Implementation Notes:**
- **Format Overview** (see provider-formats.md for details):
  - Qwen3-Coder uses Hermes-style function calling when served via vLLM with `--tool-call-parser hermes`
  - **Tool definitions**: Same as standard OpenAI format (no changes needed)
  - **Tool call responses**: Can be either OpenAI `tool_calls` array OR Qwen-Agent `function_call` object
- **Response Formats**:
  - **vLLM/OpenRouter**: Returns standard `tool_calls` array with `id`, `type`, `function` fields
  - **Qwen-Agent**: Returns `function_call` object with `name` and `arguments` fields
- **Tool Result Formats**:
  - **OpenAI style**: `role: "tool"` with `tool_call_id` (preferred for OpenRouter)
  - **Qwen style**: `role: "function"` with `name` field (alternative)
- **Known Issues**:
  - Qwen2.5-Coder has unreliable tool calling (GitHub #180) - **must avoid**
  - Qwen3/Qwen3-Coder dramatically improved reliability
  - Context window approaching limits (>100K) may cause "nonsense" generation with tools
- **Implementation Strategy**: Accept both formats on response, output OpenAI-compatible format for consistency

### 4. DeepSeek Standard Format

**Description:** Use standard OpenAI tool calling format for DeepSeek

**Implementation Notes:**
- **Format**: Pure OpenAI-compatible, no modifications required
- **Tool definitions**: Standard OpenAI `tools` array with `type: "function"` and `function` object
- **Tool call responses**: Standard `tool_calls` array with `id`, `type: "function"`, and `function` containing `name` and `arguments`
- **Tool results**: Standard `role: "tool"` with `tool_call_id` field
- **Streaming**: Standard SSE format with `delta.tool_calls` chunks
- **API Endpoints**:
  - Standard: `https://api.deepseek.com/v1/chat/completions`
  - Beta (strict mode): `https://api.deepseek.com/beta` with `strict: true` in function definitions
- **Implementation**: Default behavior, existing transform.go logic works as-is
- **Verification**: DeepSeek is confirmed to follow OpenAI conventions exactly

### 5. Streaming Provider Handling

**Description:** Apply provider-specific transformations in streaming mode

**Implementation Notes:**
- Extend processStreamDelta() to apply provider-specific logic when processing tool_call deltas
- Ensure toolCallJSONMap state tracking handles provider formats correctly
- May need provider context passed through streaming state
- Handle partial tool call chunks that may be split across multiple SSE events

### 6. Tool Schema Transformation

**Description:** Ensure tool schemas are compatible with provider-specific requirements

**Implementation Notes:**
- Extend removeUriFormat() or create provider-specific schema transformation functions
- Some providers may have additional unsupported schema properties beyond 'format: uri'
- Document which schema features are supported/unsupported for each provider
- Consider adding schema validation before sending to provider

### 7. Error Handling

**Description:** Proper error propagation when provider-specific transformation fails

**Implementation Notes:**
- Add error returns to transformation functions when provider-specific logic encounters invalid input
- Map to appropriate HTTP status codes (400 for client errors, 502 for upstream provider issues) in server.handleMessages()
- Provide clear error messages that help users understand what went wrong
- Log detailed error information for debugging while returning user-friendly messages to client

### 8. Configuration

**Description:** Optional provider override configuration

**Implementation Notes:**
- Consider adding optional config field to force provider type for specific model mappings, bypassing auto-detection
- Low priority - implement if auto-detection proves unreliable
- Would allow users to manually specify provider type in athena.yml config file
- Format example: `provider_override: { "anthropic/claude-3-opus": "deepseek" }`

## Error Scenario Matrix

| Scenario | Provider | Expected Behavior | HTTP Status |
|----------|----------|-------------------|-------------|
| **Malformed tool call from provider** | Kimi K2 | Parse failure → return error to client | 502 (Bad Gateway) |
| **Special tokens split across chunks** | Kimi K2 | Buffer until complete, or timeout after 10KB | 500 (if timeout) |
| **Missing tool_call_id in response** | All | Generate synthetic ID, log warning | 200 (continue) |
| **Tool schema incompatibility** | All | Remove unsupported properties, validate | 400 (if validation fails) |
| **Context window exceeded with tools** | Qwen | Return error before sending to provider | 400 (Bad Request) |
| **Provider detection ambiguity** | All | Use precedence order (Kimi > Qwen > DeepSeek > Standard) | 200 (continue) |
| **Unknown provider keyword** | All | Fall back to Standard (OpenAI) format | 200 (continue) |
| **Tool result without matching tool_use** | All | Reject in validateToolCalls() | 400 (Bad Request) |
| **Streaming chunk parse failure** | All | Send error SSE event, terminate stream | 200 (stream error event) |
| **Tool call response truncated** | All | Return error indicating incomplete response | 502 (Bad Gateway) |
| **Invalid JSON in tool arguments** | All | Return error with details | 400 (Bad Request) |
| **Provider API format change** | Kimi/Qwen | Parsing failure → log error, return 502 | 502 (Bad Gateway) |

## Configuration Schema

### Proposed Configuration (athena.yml)

```yaml
# Provider-specific settings (optional)
providers:
  kimi_k2:
    # Override special token detection (if format changes)
    start_token: "<|tool_calls_section_begin|>"
    end_token: "<|tool_calls_section_end|>"
    buffer_limit_kb: 10  # Max buffer size for streaming

  qwen_hermes:
    # Format version override (if needed)
    format_version: "v1"
    context_limit_kb: 100  # Warn when approaching context limits

  # Manual provider override (bypass auto-detection)
  provider_override:
    "anthropic/claude-3-opus": "qwen"
    "custom-model-id": "kimi"
```

### Configuration Priority
1. Manual `provider_override` (if configured)
2. Auto-detection from model ID
3. Fallback to Standard OpenAI format

## Future Enhancements

**Not in initial scope, consider for later phases:**

### Phase 5: Advanced Features
- **Auto-retry with fallback models** on parse failures
- **Tool call validation against schemas** (validate arguments match parameter types)
- **Cost tracking per model/tool** (monitor which models/tools used)
- **A/B testing framework** for model comparison
- **Reasoning trace handling** for DeepSeek R1 and Qwen3-Next-80B Thinking variants

### Phase 6: Optimization
- **Native bindings** for performance-critical parsing (if <1ms target not met)
- **Response caching** for identical tool calls (reduce redundant API calls)
- **Predictive context management** (warn before approaching context limits)
- **Multi-model consensus** for critical operations (parallel tool calls, compare results)

### Additional Model Support
- LLaMA-based models with tool calling
- Mistral function calling format
- Gemini function calling support
- Additional provider-specific formats as they emerge

## References

- **Provider Format Details**: See `docs/specs/toolcalling/provider-formats.md` for comprehensive format documentation with examples
- **Kimi K2 Tool Calling Guide**: https://huggingface.co/moonshotai/Kimi-K2-Instruct/blob/main/docs/tool_call_guidance.md
- **Qwen3 Function Calling**: https://qwen.readthedocs.io/en/latest/framework/function_call.html
- **DeepSeek Function Calling**: https://api-docs.deepseek.com/guides/function_calling
- **Architecture Decisions**: See `docs/specs/toolcalling/architecture.md` (to be created)
