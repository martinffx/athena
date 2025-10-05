# Tool Calling Implementation Tasks

This document provides a human-readable checklist for implementing the tool calling feature using Test-Driven Development (TDD).

## Progress Overview

- **Total Phases**: 7
- **Total Tasks**: 34
- **Parallel Execution**: Phases 3 (Kimi) and 4 (Qwen) can run in parallel

---

## Phase 1: Foundation - Type System

**Dependencies**: None (foundation layer)
**Parallel Execution**: All tasks can be done together (single file)

### ✅ Tasks

- [ ] **1.1** Add ModelFormat enum to types.go
  - Define `ModelFormat` type with iota constants
  - Add constants: `FormatDeepSeek`, `FormatQwen`, `FormatKimi`, `FormatStandard`
  - Implement `String()` method for readable format names
  - **File**: `internal/transform/types.go`

- [ ] **1.2** Add TransformContext struct to types.go
  - Define struct with `Format ModelFormat` and `Config *config.Config` fields
  - Document field purposes
  - **File**: `internal/transform/types.go`

- [ ] **1.3** Add StreamState struct to types.go
  - Define struct with 6 fields: `ContentBlockIndex`, `HasStartedTextBlock`, `IsToolUse`, `CurrentToolCallID`, `ToolCallJSONMap`, `FormatContext`
  - Consolidates 8+ parameters into single struct
  - **File**: `internal/transform/types.go`

- [ ] **1.4** Add FormatStreamContext struct to types.go
  - Define struct with `Format`, `KimiBuffer`, `KimiBufferLimit`, `KimiInToolSection` fields
  - Isolates format-specific streaming state
  - **File**: `internal/transform/types.go`

---

## Phase 2: Provider Detection

**Dependencies**: Phase 1 (needs ModelFormat enum)

### ✅ Tasks

- [ ] **2.1** Create internal/transform/providers.go file
  - Create file with `package transform` declaration
  - Add imports: `strings`, `regexp`, `fmt`
  - **File**: `internal/transform/providers.go`

- [ ] **2.2** Implement DetectModelFormat function (TDD)
  - **Test**: Write 12 test cases in `providers_test.go`
    - `moonshot/kimi-k2` → `FormatKimi`
    - `kimi-k2-instruct` → `FormatKimi`
    - `qwen/qwen3-coder` → `FormatQwen`
    - `deepseek-chat` → `FormatDeepSeek`
    - `KIMI-K2` → `FormatKimi` (case insensitive)
    - `qwen-deepseek-mix` → `FormatQwen` (precedence)
    - `unknown-model` → `FormatStandard` (fallback)
    - 5 more edge cases
  - **Implement**: `func DetectModelFormat(modelID string) ModelFormat`
    - Normalize to lowercase
    - Check OpenRouter format (provider/model split)
    - Keyword matching with precedence: Kimi > Qwen > DeepSeek
    - Fallback to FormatStandard
  - **Refactor**: Optimize string operations
  - **Files**: `internal/transform/providers.go`, `internal/transform/providers_test.go`

---

## Phase 3: Kimi K2 Format Parsing

**Dependencies**: Phase 1 (types), Phase 2 (detection)
**Parallel**: Can run in parallel with Phase 4

### ✅ Tasks

- [ ] **3.1** Implement parseKimiToolCalls function (TDD)
  - **Test**: Write 10 test cases in `providers_test.go`
    - Single tool call
    - Multiple tool calls
    - Nested JSON arguments
    - Malformed special tokens (missing begin/end)
    - No tool calls present
    - Invalid ID format
    - 4 more edge cases
  - **Implement**: `func parseKimiToolCalls(content string) ([]ToolCall, error)`
    - Check for `<|tool_calls_section_begin|>` presence
    - Regex extract tool_calls_section
    - Regex extract individual tool_call blocks
    - Parse ID (`functions.{name}:{idx}`) and JSON arguments
    - Return error for malformed tokens
  - **Refactor**: Optimize regex patterns
  - **File**: `internal/transform/providers.go`

- [ ] **3.2** Create internal/transform/streaming.go file
  - Create file with `package transform` declaration
  - Add imports: `net/http`, `strings`, `fmt`, `encoding/json`
  - **File**: `internal/transform/streaming.go`

- [ ] **3.3** Implement handleKimiStreaming function (TDD)
  - **Test**: Write 5 test cases in `streaming_test.go`
    - Complete section in one chunk
    - Section split across 2 chunks
    - Section split across 5 chunks
    - Buffer limit exceeded (>10KB)
    - Missing end token
  - **Implement**: `func handleKimiStreaming(w http.ResponseWriter, flusher http.Flusher, state *StreamState, chunk string) error`
    - Append chunk to `state.FormatContext.KimiBuffer`
    - Check 10KB buffer limit
    - Detect `<|tool_calls_section_end|>` completion
    - Parse complete section with `parseKimiToolCalls`
    - Emit Anthropic SSE events
    - Clear buffer after emission
  - **Refactor**: Extract event emission helpers
  - **File**: `internal/transform/streaming.go`

---

## Phase 4: Qwen Hermes Format Parsing

**Dependencies**: Phase 1 (types)
**Parallel**: Can run in parallel with Phase 3

### ✅ Tasks

- [ ] **4.1** Implement parseQwenToolCall function (TDD)
  - **Test**: Write 8 test cases in `providers_test.go`
    - `tool_calls` array format
    - `function_call` object format
    - Synthetic ID generation
    - Empty delta
    - Multiple tool calls in array
    - Missing fields
    - 2 more edge cases
  - **Implement**: `func parseQwenToolCall(delta map[string]interface{}) []ToolCall`
    - Check for `tool_calls` array (vLLM format)
    - Check for `function_call` object (Qwen-Agent format)
    - Generate synthetic ID for `function_call`
    - Return unified ToolCall array
  - **Refactor**: Extract ID generation helper
  - **File**: `internal/transform/providers.go`

- [ ] **4.2** Add Qwen streaming support (TDD)
  - **Test**: Update streaming tests with Qwen routing
  - **Implement**: Add Qwen routing to `processStreamDelta`
    - Call `parseQwenToolCall` for `FormatQwen`
    - Handle both `tool_calls` and `function_call` formats
  - **Refactor**: Consolidate format routing logic
  - **File**: `internal/transform/streaming.go`

- [ ] **4.3** Write streaming tests for Qwen
  - Test `tool_calls` array streaming
  - Test `function_call` object streaming
  - Test mixed content (text + tools)
  - Test multiple tool calls
  - All 5 test cases pass
  - **File**: `internal/transform/streaming_test.go`

---

## Phase 5: Integration with Existing Transform Pipeline

**Dependencies**: Phases 2, 3, 4 (all parsing logic complete)

### ✅ Tasks

- [ ] **5.1** Modify AnthropicToOpenAI to create TransformContext (TDD)
  - **Test**: Update tests to verify context creation and format detection
  - **Implement**:
    - Create `TransformContext` at function start
    - Call `DetectModelFormat(mappedModel)`
    - Pass context to `transformMessage`
  - **Refactor**: Ensure existing functionality preserved
  - **File**: `internal/transform/transform.go`

- [ ] **5.2** Modify transformMessage to add ctx parameter
  - Update function signature: `func transformMessage(msg Message, ctx *TransformContext) []OpenAIMessage`
  - Update all callers
  - Verify tests pass (parameter currently unused)
  - **File**: `internal/transform/transform.go`

- [ ] **5.3** Modify OpenAIToAnthropic to add format parameter and call parsers (TDD)
  - **Test**: Write tests for all format routing scenarios
  - **Implement**:
    - Add `format ModelFormat` parameter to signature
    - Switch on format type
    - Call `parseKimiToolCalls` for `FormatKimi`
    - Call `parseQwenToolCall` for `FormatQwen`
    - Preserve existing logic for `FormatStandard`/`FormatDeepSeek`
  - **Refactor**: Extract format routing to helper if needed
  - **File**: `internal/transform/transform.go`

- [ ] **5.4** Modify HandleStreaming to create StreamState (TDD)
  - **Test**: Update streaming tests to verify state creation
  - **Implement**:
    - Create `StreamState` with initialized `FormatStreamContext`
    - Set `KimiBufferLimit` to 10KB
    - Pass state to `processStreamDelta`
  - **Refactor**: Verify state initialization
  - **File**: `internal/transform/streaming.go`

- [ ] **5.5** Modify processStreamDelta to use StreamState and route by format (TDD)
  - **Test**: Update all streaming tests for new signature
  - **Implement**:
    - Change signature: `func processStreamDelta(w http.ResponseWriter, flusher http.Flusher, delta map[string]interface{}, state *StreamState) error`
    - Reduce parameters from 8+ to 4
    - Switch on `state.FormatContext.Format`
    - Route `FormatKimi` to `handleKimiStreaming`
    - Route `FormatQwen` to `parseQwenToolCall`
    - Preserve existing `FormatStandard` logic
  - **Refactor**: Consolidate routing logic
  - **File**: `internal/transform/streaming.go`

- [ ] **5.6** Write integration tests for full request/response cycles
  - Test complete Kimi flow: Anthropic request → OpenRouter → Kimi response → Anthropic format
  - Test complete Qwen flow with both format variants
  - Test complete DeepSeek/Standard flow (baseline)
  - Verify `tool_use` blocks correctly formatted
  - Verify multi-turn conversations with `tool_result`
  - All 3 integration tests pass
  - **File**: `internal/transform/integration_test.go`

---

## Phase 6: Error Handling and Logging

**Dependencies**: Phase 5 (integrated system)

### ✅ Tasks

- [ ] **6.1** Implement sendStreamError helper function (TDD)
  - **Test**: Verify error SSE event format
  - **Implement**: `func sendStreamError(w http.ResponseWriter, flusher http.Flusher, errorType string, message string)`
    - Send `event: error` with Anthropic error format
    - Send `event: message_stop` to terminate stream
    - Flush after each event
  - **Refactor**: Verify event format compliance
  - **File**: `internal/transform/streaming.go`

- [ ] **6.2** Add error handling to transformation functions
  - Malformed tool definitions → 400 errors
  - Regex compilation failures → 500 errors
  - Malformed OpenRouter responses → 502 errors
  - Buffer exceeded → 502 error
  - All error paths tested
  - **Files**: `internal/transform/transform.go`, `internal/transform/providers.go`, `internal/transform/streaming.go`

- [ ] **6.3** Add error handling to server.go after OpenRouter response
  - Capture errors from `OpenAIToAnthropic()`
  - Capture errors from `HandleStreaming()`
  - Map error types to correct status codes (400, 500, 502)
  - Log errors at appropriate levels
  - Return sanitized error messages to client
  - **File**: `internal/server/server.go`

- [ ] **6.4** Add format detection logging to server.go
  - Log detected format with model mapping
  - Example: "provider detected: kimi, model: moonshot/kimi-k2"
  - Use appropriate log level (info or debug)
  - Include request context
  - **File**: `internal/server/server.go`

- [ ] **6.5** Write error scenario tests
  - Test malformed tool definition (400)
  - Test unknown tool_call_id (400)
  - Test regex compilation error (500)
  - Test malformed OpenRouter response (502)
  - Test buffer exceeded (502)
  - Test streaming error event format
  - All error tests pass
  - **File**: `internal/transform/error_test.go`

---

## Phase 7: Documentation Updates

**Dependencies**: Phase 6 (complete implementation)

### ✅ Tasks

- [ ] **7.1** Update CLAUDE.md with tool calling features
  - Document three supported formats (DeepSeek, Qwen, Kimi)
  - Explain format detection strategy
  - Update architecture overview with new components
  - Add tool calling to feature list
  - Documentation is clear and accurate
  - **File**: `CLAUDE.md`

- [ ] **7.2** Create example configurations for all three formats
  - Example config for DeepSeek with tool calling
  - Example config for Qwen3-Coder with tool calling
  - Example config for Kimi K2 with tool calling
  - Each example includes API key placeholder and model mapping
  - Examples are tested and working
  - **Files**: `examples/deepseek-tools.yml`, `examples/qwen-tools.yml`, `examples/kimi-tools.yml`

---

## Implementation Notes

### TDD Workflow

For each service/function task:
1. **Test**: Write comprehensive test cases first
2. **Implement**: Write minimal code to pass tests
3. **Refactor**: Optimize and clean up

### Parallel Execution Opportunities

- **Phase 1**: All type additions (single file edit)
- **Phases 3 & 4**: Kimi and Qwen parsing are independent
- **Within phases**: Test writing can happen alongside implementation

### File Summary

**New Files**:
- `internal/transform/providers.go` (~250 lines)
- `internal/transform/streaming.go` (~300 lines)
- `internal/transform/providers_test.go`
- `internal/transform/streaming_test.go`
- `internal/transform/integration_test.go`
- `internal/transform/error_test.go`
- `examples/deepseek-tools.yml`
- `examples/qwen-tools.yml`
- `examples/kimi-tools.yml`

**Modified Files**:
- `internal/transform/types.go` (~88 → ~188 lines)
- `internal/transform/transform.go` (~400 lines, context additions)
- `internal/server/server.go` (error handling, logging)

### Key Acceptance Criteria

- All unit tests pass (80-100 tests total)
- All integration tests pass (3 full-cycle tests)
- All error scenario tests pass (12 error tests)
- DeepSeek passthrough still works (baseline)
- Kimi special tokens parsed correctly
- Qwen dual formats accepted
- Streaming maintains state consistency
- Error handling returns correct HTTP status codes
- Documentation updated and accurate

---

**Next Step**: `/spec:implement toolcalling` to begin TDD implementation
