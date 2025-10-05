# Tool Calling - Spec Lite

## Summary
Enhance Athena's proxy to reliably translate tool calling between Anthropic and OpenRouter formats with provider-specific handling for DeepSeek, Qwen3-Coder, and Kimi K2.

## User Story
As an Athena user, I want the proxy to reliably translate tool calling between Anthropic and OpenRouter formats so that I can use Claude Code with alternative models (DeepSeek, Qwen3-Coder, Kimi K2) that support function calling.

## Top 5 Acceptance Criteria

1. **Tool Schema Translation** - GIVEN an Anthropic API request with tool definitions WHEN the request is proxied to OpenRouter THEN tool schemas are correctly translated to OpenAI format with unsupported properties removed

2. **Streaming Tool Call Events** - GIVEN an OpenRouter streaming response with tool call deltas WHEN processed by the streaming handler THEN content_block_start, content_block_delta, and content_block_stop events are emitted in Anthropic SSE format

3. **Kimi K2 Provider Support** - GIVEN a Kimi K2 model request WHEN provider-specific handling is applied THEN special tokens are properly managed in tool call translation

4. **Qwen3-Coder Provider Support** - GIVEN a Qwen3-Coder model request WHEN provider-specific handling is applied THEN Hermes-style tool calling format is used

5. **Tool Call Validation** - GIVEN a multi-turn conversation with tool calls and responses WHEN validateToolCalls() is invoked THEN all tool_use blocks have matching tool_result blocks

## Top 3 Business Rules

1. **Multi-Model Support** - All three target models must be supported: DeepSeek (standard OpenAI format), Qwen3-Coder (Hermes-style), Kimi K2 (special tokens)

2. **Transparency** - Tool calling translation must be transparent to the Claude Code client

3. **Provider Detection** - Provider-specific quirks must be detected based on model name/identifier in the request

## Scope

**Included:**
- Provider detection logic based on model identifier
- Provider-specific tool call format transformations
- Enhanced streaming support for provider-specific formats
- Integration with existing transformation functions
- Testing with all three target models

**Excluded:**
- Custom tool calling implementations beyond format translation
- Retry logic or error recovery for failed tool calls
- Tool execution (only translation between API formats)
- Support for additional models beyond the initial three

## Top 5 Dependencies

1. `transform.AnthropicToOpenAI()` - Main transformation function that needs provider-specific logic
2. `transform.transformMessage()` - Message conversion that handles tool_use and tool_result blocks
3. `transform.HandleStreaming()` - Streaming handler for SSE processing
4. `transform.processStreamDelta()` - Processes streaming deltas including tool call chunks
5. `transform.validateToolCalls()` - Tool call validation ensuring matching responses

## Technical Details

1. **Provider Detection** - Identify which model/provider is being used to apply appropriate tool calling format
2. **Kimi K2 Special Token Handling** - Apply Kimi-specific special tokens to tool call requests
3. **Qwen Hermes-Style Format** - Convert tool calls to Hermes-style format for Qwen3-Coder
4. **DeepSeek Standard Format** - Use standard OpenAI tool calling format for DeepSeek
5. **Streaming Provider Handling** - Apply provider-specific transformations in streaming mode
6. **Tool Schema Transformation** - Ensure tool schemas are compatible with provider-specific requirements
7. **Error Handling** - Proper error propagation when provider-specific transformation fails
8. **Configuration** - Optional provider override configuration
