# Tool Calling Technical Design

## Overview

**Feature**: Tool Calling Format Translation
**Type**: Transformation Layer Enhancement
**Scope**: Extend Athena's existing request/response transformation to handle three different tool calling formats returned by OpenRouter

### Key Insight

This feature detects **model formats** (how OpenRouter returns tool calls for different models), not infrastructure providers (Groq, DeepInfra, etc.). All requests flow through OpenRouter - we adapt to how OpenRouter formats responses for each model type.

### Architecture Summary

Athena proxies Anthropic API requests to OpenRouter, translating between formats. This enhancement adds format-specific handling for tool calls:

- **FormatDeepSeek / FormatStandard**: Standard OpenAI tool calling (no changes needed)
- **FormatQwen**: Hermes-style tool calling (dual format acceptance)
- **FormatKimi**: Special token-based tool calling (requires parsing and buffering)

**No database, no new APIs** - purely transformation logic within the existing `/v1/messages` endpoint.

---

## Domain Model

### Data Structures

#### ModelFormat Enum

```go
// internal/transform/types.go

type ModelFormat int

const (
    FormatDeepSeek ModelFormat = iota  // Standard OpenAI format
    FormatQwen                          // Hermes-style format
    FormatKimi                          // Special tokens format
    FormatStandard                      // Default OpenAI-compatible fallback
)

// String representation for logging
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
```

**Purpose**: Identifies which tool calling response format OpenRouter will return based on the model being used.

#### TransformContext Struct

```go
// internal/transform/types.go

type TransformContext struct {
    Format ModelFormat    // Detected tool call format for this request
    Config *config.Config // Reference to global configuration
}
```

**Purpose**: Encapsulates format information and configuration, passed through transformation pipeline instead of multiple parameters.

**Usage**:
```go
ctx := &TransformContext{
    Format: DetectModelFormat(mappedModel),
    Config: cfg,
}
messages := transformMessage(msg, ctx)
```

#### StreamState Struct

```go
// internal/transform/types.go

type StreamState struct {
    ContentBlockIndex   int               // Current content block index
    HasStartedTextBlock bool              // Whether text block started
    IsToolUse          bool              // Currently processing tool calls
    CurrentToolCallID  string            // ID of current tool call
    ToolCallJSONMap    map[string]string // Accumulated JSON per tool call ID
    FormatContext      *FormatStreamContext // Format-specific streaming state
}
```

**Purpose**: Consolidates all streaming state into a single struct, reducing `processStreamDelta` parameter count from 8+ to 2.

**Before**:
```go
func processStreamDelta(w, flusher, delta, contentBlockIndex,
                        hasStartedTextBlock, isToolUse, currentToolCallID,
                        toolCallJSONMap) // 8 parameters!
```

**After**:
```go
func processStreamDelta(w http.ResponseWriter, flusher http.Flusher,
                        delta map[string]interface{}, state *StreamState) // 4 parameters
```

#### FormatStreamContext Struct

```go
// internal/transform/types.go

type FormatStreamContext struct {
    Format            ModelFormat      // Which format is being streamed
    KimiBuffer        strings.Builder  // Buffer for Kimi special tokens
    KimiBufferLimit   int             // Max buffer size (10KB)
    KimiInToolSection bool            // Inside <|tool_calls_section|>
}
```

**Purpose**: Isolates format-specific streaming state (primarily Kimi K2 buffering) from general streaming state.

---

## Services/Functions

### Format Detection

#### DetectModelFormat

```go
// internal/transform/providers.go

func DetectModelFormat(modelID string) ModelFormat
```

**Purpose**: Analyzes model identifier to determine which tool calling response format OpenRouter will use.

**Algorithm**:
1. Normalize to lowercase
2. Check OpenRouter format (`provider/model` → extract provider part)
3. Keyword matching with precedence: Kimi > Qwen > DeepSeek
4. Fallback to FormatStandard

**Example**:
```go
DetectModelFormat("moonshot/kimi-k2")      // → FormatKimi
DetectModelFormat("qwen/qwen3-coder")      // → FormatQwen
DetectModelFormat("deepseek-chat")         // → FormatDeepSeek
DetectModelFormat("unknown-model")         // → FormatStandard (fallback)
DetectModelFormat("DeepSeek-R1")           // → FormatDeepSeek (case-insensitive)
```

**Complexity**: O(1) - simple string operations

---

### Kimi K2 Format Handling

#### parseKimiToolCalls

```go
// internal/transform/providers.go

func parseKimiToolCalls(content string) ([]ToolCall, error)
```

**Purpose**: Extracts tool calls from Kimi K2 special token format in OpenRouter responses.

**Input Format**:
```
<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Tokyo"}<|tool_call_end|>
<|tool_calls_section_end|>
```

**Output**:
```go
[]ToolCall{{
    ID:   "functions.get_weather:0",
    Type: "function",
    Function: Function{
        Name:      "get_weather",
        Arguments: `{"city": "Tokyo"}`,
    },
}}
```

**Algorithm**:
1. Check for `<|tool_calls_section_begin|>` presence
2. Regex extract tool_calls_section content
3. Regex extract individual tool_call blocks
4. Parse ID (`functions.{name}:{idx}`) and JSON arguments
5. Convert to ToolCall structs

**Error Handling**: Returns error if malformed special tokens detected.

#### handleKimiStreaming

```go
// internal/transform/streaming.go

func handleKimiStreaming(w http.ResponseWriter, flusher http.Flusher,
                         state *StreamState, chunk string) error
```

**Purpose**: Buffers Kimi K2 special tokens across SSE chunks until complete section received.

**Algorithm**:
1. Append chunk to `state.FormatContext.KimiBuffer`
2. Check buffer size < 10KB limit (return error if exceeded)
3. Check for `<|tool_calls_section_end|>` token
4. If complete: parse section, send Anthropic SSE events, clear buffer
5. If incomplete: continue buffering

**Example Flow**:
```
Chunk 1: "<|tool_calls_section_begin|>\n<|tool_call_begin|>fun"
         → Buffer, wait

Chunk 2: "ctions.get_weather:0<|tool_call_argument_begin|>{\"ci"
         → Buffer, wait

Chunk 3: "ty\": \"Tokyo\"}<|tool_call_end|>\n<|tool_calls_section_end|>"
         → Complete! Parse and emit events
```

---

### Qwen Format Handling

#### parseQwenToolCall

```go
// internal/transform/providers.go

func parseQwenToolCall(delta map[string]interface{}) []ToolCall
```

**Purpose**: Accepts both OpenAI `tool_calls` array AND Qwen-Agent `function_call` object from OpenRouter responses.

**Accepts Format 1** (vLLM with hermes parser):
```json
{
  "message": {
    "tool_calls": [{
      "id": "chatcmpl-tool-abc",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"city\": \"Tokyo\"}"
      }
    }]
  }
}
```

**Accepts Format 2** (Qwen-Agent):
```json
{
  "message": {
    "function_call": {
      "name": "get_weather",
      "arguments": "{\"city\": \"Tokyo\"}"
    }
  }
}
```

**Algorithm**:
1. Check for `tool_calls` array → return as-is
2. Check for `function_call` object → convert to ToolCall with synthetic ID
3. Return unified ToolCall array

**Complexity**: O(n) where n is number of tool calls

---

### Modified Existing Functions

#### transformMessage

```go
// internal/transform/transform.go

func transformMessage(msg Message, ctx *TransformContext) []OpenAIMessage
```

**Change**: Add `ctx *TransformContext` parameter
**Current Use**: Parameter currently unused (all formats accept standard OpenAI tool definitions in requests)
**Future**: Enables format-specific request transformations if needed

#### OpenAIToAnthropic

```go
// internal/transform/transform.go

func OpenAIToAnthropic(resp map[string]interface{}, modelName string,
                       format ModelFormat) map[string]interface{}
```

**Change**: Add `format ModelFormat` parameter
**Implementation**: Apply format-specific parsing before standard transformation

```go
// Pseudocode
func OpenAIToAnthropic(resp map[string]interface{}, modelName string,
                       format ModelFormat) map[string]interface{} {
    // Parse tool calls based on format
    switch format {
    case FormatKimi:
        if content, ok := resp["content"].(string); ok {
            toolCalls, err := parseKimiToolCalls(content)
            if err != nil {
                // Handle error
            }
            resp["tool_calls"] = toolCalls
        }
    case FormatQwen:
        toolCalls := parseQwenToolCall(resp)
        resp["tool_calls"] = toolCalls
    }

    // Continue with existing transformation logic
    return convertToAnthropicFormat(resp)
}
```

#### processStreamDelta

```go
// internal/transform/streaming.go

func processStreamDelta(w http.ResponseWriter, flusher http.Flusher,
                        delta map[string]interface{}, state *StreamState) error
```

**Changes**:
1. Replace 8 individual parameters with `state *StreamState`
2. Add format-specific routing at start

```go
func processStreamDelta(w http.ResponseWriter, flusher http.Flusher,
                        delta map[string]interface{}, state *StreamState) error {
    // Route to format-specific handler
    switch state.FormatContext.Format {
    case FormatKimi:
        if chunk, ok := delta["content"].(string); ok {
            return handleKimiStreaming(w, flusher, state, chunk)
        }
    case FormatQwen:
        toolCalls := parseQwenToolCall(delta)
        // Process Qwen tool calls...
    default:
        // Standard OpenAI processing (existing logic)
    }

    // Existing streaming logic continues...
}
```

#### sendStreamError

```go
// internal/transform/streaming.go (NEW)

func sendStreamError(w http.ResponseWriter, flusher http.Flusher,
                     errorType string, message string)
```

**Purpose**: Sends Anthropic-format error SSE event and gracefully terminates stream.

**Implementation**:
```go
func sendStreamError(w http.ResponseWriter, flusher http.Flusher,
                     errorType string, message string) {
    errorEvent := map[string]interface{}{
        "type": "error",
        "error": map[string]interface{}{
            "type":    errorType,
            "message": message,
        },
    }

    data, _ := json.Marshal(errorEvent)
    fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
    flusher.Flush()

    // Send stream end event
    fmt.Fprintf(w, "event: message_stop\ndata: {}\n\n")
    flusher.Flush()
}
```

---

## Component Architecture

### File Organization

Multi-file single-package architecture (maintains Athena's simplicity):

```
internal/transform/
├── transform.go    (~400 lines) - Core transformation logic
│   ├── AnthropicToOpenAI() - Modified: create TransformContext, detect format
│   ├── OpenAIToAnthropic() - Modified: add format parameter, call parsers
│   ├── transformMessage() - Modified: add ctx parameter
│   └── validateToolCalls() - Unchanged (existing validation)
│
├── providers.go    (~250 lines) - NEW: Format detection & parsing
│   ├── DetectModelFormat() - Format detection with precedence
│   ├── parseKimiToolCalls() - Kimi special token parser
│   └── parseQwenToolCall() - Qwen dual format parser
│
├── streaming.go    (~300 lines) - NEW: Streaming with format context
│   ├── HandleStreaming() - Modified: create StreamState
│   ├── processStreamDelta() - Modified: route to format handlers
│   ├── handleKimiStreaming() - Kimi streaming with buffering
│   └── sendStreamError() - Stream error helper
│
└── types.go        (~88 → ~188 lines) - Add format types
    ├── ModelFormat enum - NEW
    ├── TransformContext - NEW
    ├── StreamState - NEW
    └── FormatStreamContext - NEW
```

### Transformation Pipeline

#### Request Flow

```
┌─────────────────────┐
│ Anthropic Request   │
│ (with tools)        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ AnthropicToOpenAI() │
│ • Detect format     │ ◄── DetectModelFormat(mappedModel)
│ • Create context    │
│ • Transform tools   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ OpenRouter API      │
│ (standard OpenAI)   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ OpenRouter Response │
│ (format-specific)   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ OpenAIToAnthropic() │
│ • Parse by format   │ ◄── parseKimiToolCalls() / parseQwenToolCall()
│ • Convert to        │
│   Anthropic format  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Anthropic Response  │
│ (tool_use blocks)   │
└─────────────────────┘
```

#### Streaming Flow

```
┌────────────────────┐
│ OpenRouter SSE     │
│ (format-specific   │
│  chunks)           │
└─────────┬──────────┘
          │
          ▼
┌────────────────────┐
│ processStreamDelta │
│ • Check format     │
│ • Route to handler │
└─────────┬──────────┘
          │
          ├─── FormatKimi ──────► handleKimiStreaming()
          │                      • Buffer chunks
          │                      • Parse on complete
          │                      • Emit Anthropic events
          │
          ├─── FormatQwen ──────► parseQwenToolCall()
          │                      • Accept both formats
          │                      • Convert to standard
          │
          └─── FormatStandard ──► Standard processing
                                  • Existing logic
```

---

## Implementation Details

### Format Detection Strategy

**Precedence Order**: Kimi > Qwen > DeepSeek > Standard

**Rationale**: Most specific to least specific. Kimi has unique special tokens, Qwen has Hermes format, DeepSeek/Standard use same OpenAI format.

```go
func DetectModelFormat(modelID string) ModelFormat {
    normalized := strings.ToLower(modelID)

    // 1. OpenRouter format: provider/model
    if parts := strings.Split(normalized, "/"); len(parts) == 2 {
        switch parts[0] {
        case "moonshot":  // Kimi's provider on OpenRouter
            return FormatKimi
        case "qwen":
            return FormatQwen
        case "deepseek":
            return FormatDeepSeek
        }
    }

    // 2. Keyword matching with precedence
    if strings.Contains(normalized, "kimi") || strings.Contains(normalized, "k2") {
        return FormatKimi
    }
    if strings.Contains(normalized, "qwen") {
        return FormatQwen
    }
    if strings.Contains(normalized, "deepseek") {
        return FormatDeepSeek
    }

    // 3. Default fallback
    return FormatStandard
}
```

**Test Cases** (12 total):
- `"moonshot/kimi-k2"` → FormatKimi
- `"kimi-k2-instruct"` → FormatKimi
- `"qwen/qwen3-coder"` → FormatQwen
- `"qwen3-coder-plus"` → FormatQwen
- `"deepseek/deepseek-chat"` → FormatDeepSeek
- `"deepseek-r1"` → FormatDeepSeek
- `"DeepSeek-V3"` → FormatDeepSeek (case-insensitive)
- `"claude-3-opus"` → FormatStandard (fallback)
- `"gpt-4"` → FormatStandard (fallback)
- `"KIMI-K2"` → FormatKimi (case-insensitive)
- `"unknown/model"` → FormatStandard (fallback)
- `"qwen-deepseek-mix"` → FormatQwen (precedence: Qwen > DeepSeek)

---

### Format-Specific Handling

#### FormatKimi: Special Token Parsing

**Challenge**: OpenRouter returns Kimi responses with proprietary special tokens that may split across SSE chunks.

**Solution**: Regex-based parsing with streaming buffer.

**Non-Streaming**:
```go
func parseKimiToolCalls(content string) ([]ToolCall, error) {
    if !strings.Contains(content, "<|tool_calls_section_begin|>") {
        return nil, nil  // No tool calls
    }

    // Extract section
    sectionPattern := `<\|tool_calls_section_begin\|>(.*?)<\|tool_calls_section_end\|>`
    sections := regexp.MustCompile(sectionPattern).FindStringSubmatch(content)
    if len(sections) < 2 {
        return nil, fmt.Errorf("malformed tool calls section")
    }

    // Extract individual calls
    callPattern := `<\|tool_call_begin\|>\s*(?P<id>[\w\.]+:\d+)\s*` +
                   `<\|tool_call_argument_begin\|>\s*(?P<args>.*?)\s*` +
                   `<\|tool_call_end\|>`

    var toolCalls []ToolCall
    for _, match := range regexp.MustCompile(callPattern).FindAllStringSubmatch(sections[1], -1) {
        id := match[1]
        args := match[2]

        // Parse function name from ID: functions.get_weather:0 → get_weather
        parts := strings.Split(id, ".")
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid tool call ID: %s", id)
        }
        name := strings.Split(parts[1], ":")[0]

        toolCalls = append(toolCalls, ToolCall{
            ID:   id,
            Type: "function",
            Function: Function{
                Name:      name,
                Arguments: args,
            },
        })
    }

    return toolCalls, nil
}
```

**Streaming with Buffer**:
```go
const kimiBufferLimit = 10 * 1024  // 10KB

func handleKimiStreaming(w http.ResponseWriter, flusher http.Flusher,
                         state *StreamState, chunk string) error {
    fc := state.FormatContext

    // Append to buffer
    fc.KimiBuffer.WriteString(chunk)

    // Check limit
    if fc.KimiBuffer.Len() > fc.KimiBufferLimit {
        return fmt.Errorf("kimi tool call buffer exceeded %d bytes", fc.KimiBufferLimit)
    }

    content := fc.KimiBuffer.String()

    // Check for completion
    if strings.Contains(content, "<|tool_calls_section_end|>") {
        // Parse complete section
        toolCalls, err := parseKimiToolCalls(content)
        if err != nil {
            return err
        }

        // Send Anthropic SSE events
        for _, tc := range toolCalls {
            sendToolCallStartEvent(w, flusher, state, tc)
            sendToolCallDeltaEvent(w, flusher, state, tc)
            sendToolCallStopEvent(w, flusher, state)
            state.ContentBlockIndex++
        }

        // Clear buffer
        fc.KimiBuffer.Reset()
        fc.KimiInToolSection = false
    } else if strings.Contains(content, "<|tool_calls_section_begin|>") {
        fc.KimiInToolSection = true
    }

    return nil
}
```

#### FormatQwen: Dual Format Acceptance

**Challenge**: OpenRouter may return either OpenAI `tool_calls` array OR Qwen-Agent `function_call` object depending on backend configuration.

**Solution**: Accept both, normalize to OpenAI format.

```go
func parseQwenToolCall(delta map[string]interface{}) []ToolCall {
    var toolCalls []ToolCall

    // Format 1: OpenAI tool_calls array (vLLM with hermes parser)
    if tcArray, ok := delta["tool_calls"].([]interface{}); ok {
        for _, tc := range tcArray {
            if tcMap, ok := tc.(map[string]interface{}); ok {
                toolCalls = append(toolCalls, ToolCall{
                    ID:   getString(tcMap, "id"),
                    Type: "function",
                    Function: Function{
                        Name:      getString(tcMap, "function.name"),
                        Arguments: getString(tcMap, "function.arguments"),
                    },
                })
            }
        }
        return toolCalls
    }

    // Format 2: Qwen-Agent function_call object
    if fcObj, ok := delta["function_call"].(map[string]interface{}); ok {
        toolCalls = append(toolCalls, ToolCall{
            ID:   generateSyntheticID(),  // Generate ID
            Type: "function",
            Function: Function{
                Name:      getString(fcObj, "name"),
                Arguments: getString(fcObj, "arguments"),
            },
        })
        return toolCalls
    }

    return toolCalls
}

func generateSyntheticID() string {
    return fmt.Sprintf("qwen-tool-%d", time.Now().UnixNano())
}
```

#### FormatDeepSeek / FormatStandard: Passthrough

**No transformation needed** - existing code already handles standard OpenAI format.

```go
switch format {
case FormatKimi:
    // Parse Kimi special tokens
case FormatQwen:
    // Accept dual formats
case FormatDeepSeek, FormatStandard:
    // Existing logic - no changes needed
    return existingOpenAIProcessing(delta)
}
```

---

## Error Handling

### HTTP Status Code Mapping

| Error Type | HTTP Status | Scenario | Example |
|------------|-------------|----------|---------|
| **Client Error** | 400 | Invalid tool definition | "Tool parameter missing required 'type' field" |
| **Client Error** | 400 | Tool result without call | "Tool result references unknown tool_call_id: xyz" |
| **Client Error** | 400 | Schema validation failure | "Tool schema contains invalid JSON" |
| **Server Error** | 500 | Regex compilation error | "Failed to compile Kimi token pattern" |
| **Server Error** | 500 | Transformation panic | "Unexpected nil pointer in format conversion" |
| **Server Error** | 500 | JSON marshal failure | "Failed to marshal tool call to JSON" |
| **Gateway Error** | 502 | Malformed OpenRouter response | "OpenRouter returned incomplete tool_calls structure" |
| **Gateway Error** | 502 | Invalid JSON from OpenRouter | "Cannot parse tool arguments as JSON" |
| **Gateway Error** | 502 | Buffer exceeded | "Kimi tool call buffer exceeded 10KB limit" |
| **Gateway Error** | 502 | Missing end token | "Kimi response missing </tool_calls_section_end/>" |

### Streaming Error Handling

**Strategy**: Send error SSE event, then gracefully terminate.

```go
// When transformation error occurs during streaming
if err := handleKimiStreaming(w, flusher, state, chunk); err != nil {
    log.Error("Kimi streaming error", "error", err)

    sendStreamError(w, flusher, "format_transformation_error",
                    "Failed to parse Kimi tool call format")

    // Return to stop further processing
    return
}
```

**Error SSE Format** (Anthropic-compatible):
```
event: error
data: {"type":"error","error":{"type":"format_transformation_error","message":"Failed to parse Kimi tool call format"}}

event: message_stop
data: {}
```

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1)

**Goal**: Get format detection working, verify DeepSeek passthrough

**Tasks**:
1. Add `ModelFormat` enum to `types.go`
2. Add `TransformContext`, `StreamState`, `FormatStreamContext` structs to `types.go`
3. Create `providers.go`, implement `DetectModelFormat()`
4. Write 12 unit tests for `DetectModelFormat` (all patterns, precedence, fallback)
5. Modify `AnthropicToOpenAI` to create `TransformContext` and detect format
6. Verify DeepSeek passthrough still works (run existing test suite)

**Deliverable**: Format detection infrastructure in place, DeepSeek confirmed working.

**Test Example**:
```go
func TestDetectModelFormat(t *testing.T) {
    tests := []struct {
        modelID  string
        expected ModelFormat
    }{
        {"moonshot/kimi-k2", FormatKimi},
        {"kimi-k2-instruct", FormatKimi},
        {"qwen/qwen3-coder", FormatQwen},
        {"deepseek-chat", FormatDeepSeek},
        {"unknown-model", FormatStandard},
        // ... 7 more cases
    }

    for _, tt := range tests {
        got := DetectModelFormat(tt.modelID)
        assert.Equal(t, tt.expected, got)
    }
}
```

### Phase 2: Kimi K2 (Week 2)

**Goal**: Implement Kimi special token parsing (non-streaming + streaming)

**Tasks**:
7. Implement `parseKimiToolCalls()` for non-streaming in `providers.go`
8. Write 10 unit tests (single call, multiple calls, malformed tokens)
9. Modify `OpenAIToAnthropic` to call `parseKimiToolCalls` when `format == FormatKimi`
10. Create `streaming.go`, implement `handleKimiStreaming()` with buffering
11. Write 5 streaming tests (complete in one chunk, split across chunks, buffer exceeded)
12. Modify `processStreamDelta` to route to `handleKimiStreaming` for Kimi

**Deliverable**: Kimi tool calling works in both streaming and non-streaming modes.

**Test Example**:
```go
func TestParseKimiToolCalls(t *testing.T) {
    input := `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city":"Tokyo"}<|tool_call_end|>
<|tool_calls_section_end|>`

    calls, err := parseKimiToolCalls(input)
    require.NoError(t, err)
    assert.Len(t, calls, 1)
    assert.Equal(t, "get_weather", calls[0].Function.Name)
    assert.JSONEq(t, `{"city":"Tokyo"}`, calls[0].Function.Arguments)
}
```

### Phase 3: Qwen Hermes (Week 2-3)

**Goal**: Handle dual Qwen formats

**Tasks**:
13. Implement `parseQwenToolCall()` for dual format acceptance in `providers.go`
14. Write 8 unit tests (tool_calls array, function_call object, mixed, edge cases)
15. Modify `OpenAIToAnthropic` to call `parseQwenToolCall` when `format == FormatQwen`
16. Handle Qwen streaming deltas in `processStreamDelta`
17. Write 5 streaming tests for Qwen

**Deliverable**: All three formats working (DeepSeek, Kimi, Qwen).

**Test Example**:
```go
func TestParseQwenToolCall_BothFormats(t *testing.T) {
    // Test tool_calls array
    delta1 := map[string]interface{}{
        "tool_calls": []interface{}{
            map[string]interface{}{
                "id": "call-123",
                "function": map[string]interface{}{
                    "name": "get_weather",
                    "arguments": `{"city":"Tokyo"}`,
                },
            },
        },
    }
    calls := parseQwenToolCall(delta1)
    assert.Len(t, calls, 1)

    // Test function_call object
    delta2 := map[string]interface{}{
        "function_call": map[string]interface{}{
            "name": "get_weather",
            "arguments": `{"city":"Tokyo"}`,
        },
    }
    calls = parseQwenToolCall(delta2)
    assert.Len(t, calls, 1)
}
```

### Phase 4: Hardening (Week 3)

**Goal**: Production-ready with full error handling

**Tasks**:
18. Implement `sendStreamError()` helper in `streaming.go`
19. Add error handling to all transformation functions per error matrix
20. Add error handling to `server.go` after OpenRouter response received
21. Add format detection logging to `server.go` (log detected format with model)
22. Write integration tests (full request/response cycles for all 3 formats)
23. Performance benchmarks (target <1ms transformation overhead)
24. Update documentation (CLAUDE.md, examples)

**Deliverable**: Production-ready implementation with comprehensive testing.

**Integration Test Example**:
```go
func TestFullCycleKimi(t *testing.T) {
    // Mock OpenRouter response with Kimi special tokens
    mockResponse := `<|tool_calls_section_begin|>...`

    // Full transformation pipeline
    ctx := &TransformContext{
        Format: FormatKimi,
        Config: testConfig,
    }

    anthropicResp := OpenAIToAnthropic(mockResponse, "kimi-k2", FormatKimi)

    // Verify Anthropic format output
    assert.Contains(t, anthropicResp, "content")
    assert.Equal(t, "tool_use", anthropicResp["content"][0]["type"])
}
```

---

## Testing Strategy

### Unit Tests (80-100 total)

#### Format Detection (12 tests)
- OpenRouter ID format parsing
- Keyword matching (kimi, qwen, deepseek)
- Case insensitivity
- Precedence order (Kimi > Qwen > DeepSeek)
- Fallback to FormatStandard

#### Kimi Parsing (15 tests)
- Single tool call
- Multiple tool calls
- Nested JSON arguments
- Malformed special tokens (missing begin/end)
- Streaming: complete in one chunk
- Streaming: split across 2 chunks
- Streaming: split across 5 chunks
- Streaming: buffer limit exceeded
- Streaming: missing end token

#### Qwen Parsing (12 tests)
- tool_calls array format
- function_call object format
- Mixed formats in conversation
- Synthetic ID generation
- Streaming tool_calls deltas
- Streaming function_call deltas

#### Streaming (20 tests per format)
- Single tool call streamed
- Multiple tool calls simultaneously
- Tool call + text content mixed
- Error mid-stream
- Buffer edge cases (Kimi)
- State management across chunks

### Integration Tests

**Full Cycle Tests** (3 tests, one per format):
- Complete Anthropic request → transform → mock OpenRouter → parse → Anthropic response
- Verify tool definitions, tool_use blocks, tool_result blocks
- Multi-turn conversations with tools

**Error Scenario Tests** (12 tests from error matrix):
- Each error type (400, 500, 502)
- Streaming error handling
- Error message clarity

### Performance Tests

**Benchmarks**:
```go
func BenchmarkDetectModelFormat(b *testing.B) {
    for i := 0; i < b.N; i++ {
        DetectModelFormat("moonshot/kimi-k2")
    }
}
// Target: <10 ns/op

func BenchmarkParseKimiToolCalls(b *testing.B) {
    input := `<|tool_calls_section_begin|>...`
    for i := 0; i < b.N; i++ {
        parseKimiToolCalls(input)
    }
}
// Target: <100 μs/op

func BenchmarkFullTransformationPipeline(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Full transformation
    }
}
// Target: <1 ms/op
```

**Memory Profiling**:
- Profile Kimi streaming with maximum buffer usage (10KB)
- Verify no memory leaks in long-running streams
- Check GC pressure from string building

**Throughput**:
- Measure requests/second with format transformations enabled
- Compare to baseline (without format transformations)
- Target: <5% throughput reduction

---

## Performance Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Transformation Latency** | <1ms | Time from format detection to transformation complete |
| **Memory Allocation** | <100KB per request | Including 10KB Kimi buffer, context structs |
| **Streaming First Byte** | <50ms | Time to first SSE event (format detection cached) |
| **Buffer Limit** | 10KB hard limit | Kimi streaming buffer, error if exceeded |
| **Throughput Impact** | <5% reduction | Compared to baseline without format transformations |

---

## Dependencies

### Functions Being Modified

| Function | Location | Change |
|----------|----------|--------|
| `AnthropicToOpenAI` | `transform/transform.go` | Create TransformContext, call DetectModelFormat |
| `transformMessage` | `transform/transform.go` | Add ctx parameter (unused currently) |
| `OpenAIToAnthropic` | `transform/transform.go` | Add format parameter, call format parsers |
| `HandleStreaming` | `transform/streaming.go` | Create StreamState with FormatStreamContext |
| `processStreamDelta` | `transform/streaming.go` | Replace 8 params with StreamState, route by format |

### New Functions

| Function | Location | Purpose |
|----------|----------|---------|
| `DetectModelFormat` | `transform/providers.go` | Format detection from model ID |
| `parseKimiToolCalls` | `transform/providers.go` | Kimi special token parser |
| `parseQwenToolCall` | `transform/providers.go` | Qwen dual format parser |
| `handleKimiStreaming` | `transform/streaming.go` | Kimi streaming with buffering |
| `sendStreamError` | `transform/streaming.go` | Stream error event sender |

### New Types

| Type | Location | Purpose |
|------|----------|---------|
| `ModelFormat` enum | `transform/types.go` | Format identification |
| `TransformContext` | `transform/types.go` | Context propagation |
| `StreamState` | `transform/types.go` | Streaming state consolidation |
| `FormatStreamContext` | `transform/types.go` | Format-specific streaming state |

### External Dependencies

| Package | Usage |
|---------|-------|
| `regexp` | Kimi special token parsing (pattern matching) |
| `strings` | Format detection (case-insensitive, splitting) |
| `fmt` | Error messages, buffer limit errors |

**No new external dependencies** - all packages already used by Athena.

---

## References

- **Specification**: `docs/specs/toolcalling/spec.md`
- **Architecture Decisions**: `docs/specs/toolcalling/architecture.md`
- **Provider Formats**: `docs/specs/toolcalling/provider-formats.md`
- **Athena Standards**: `docs/standards/tech.md`, `docs/standards/practices.md`

---

**Next Step**: `/spec:plan toolcalling` to generate implementation tasks
