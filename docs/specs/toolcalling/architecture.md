# Tool Calling Architecture Decisions

## Executive Summary

This document captures the key architectural decisions for implementing provider-specific tool calling support in Athena. The implementation enhances existing transformation logic to handle DeepSeek (standard OpenAI), Qwen3-Coder (Hermes-style), and Kimi K2 (special tokens) through OpenRouter.

**Key Decision**: Maintain single-package architecture with multi-file organization for maintainability.

---

## Decision 1: File Organization

### Context
Adding provider-specific logic will increase `internal/transform/transform.go` from ~640 lines to ~850+ lines, potentially impacting maintainability.

### Options Considered

#### Option A: Single File (Status Quo)
- **Pros**: Maintains current simplicity, no package changes
- **Cons**: File becomes unwieldy at 850+ lines, harder to navigate

#### Option B: Multi-Package Split
- **Pros**: Clean separation, scalable for future providers
- **Cons**: Violates Athena's single-file architecture principle, deployment complexity

#### Option C: Multi-File Single-Package ✅ **SELECTED**
- **Pros**: Maintains package simplicity, improves organization
- **Cons**: Slightly more files to manage

### Decision
**Split `internal/transform/` into multiple files within the same package:**

```
internal/transform/
├── transform.go         # Core transformation logic (~400 lines)
├── providers.go         # Provider detection & format handling (~250 lines)
├── streaming.go         # Streaming helpers (~300 lines)
└── types.go            # Existing types (~88 lines)
```

### Rationale
- Maintains Athena's single-package architecture principle
- Improves code organization and readability
- Easier to locate provider-specific logic
- No deployment or import complexity

---

## Decision 2: Provider Context Propagation

### Context
Provider-specific logic needs to be applied across multiple transformation functions. How do we pass provider information through the pipeline?

### Options Considered

#### Option A: Additional Parameter Everywhere
```go
func transformMessage(msg Message, provider Provider) []OpenAIMessage
func validateToolCalls(messages []Message, provider Provider) []Message
```
- **Pros**: Simple, explicit
- **Cons**: Changes many function signatures, parameter proliferation

#### Option B: Global State
```go
var currentProvider Provider  // Set once per request
```
- **Pros**: No signature changes
- **Cons**: Not thread-safe, violates functional principles

#### Option C: Context Struct ✅ **SELECTED**
```go
type TransformContext struct {
    Provider Provider
    Config   *config.Config
    // Future: add provider-specific settings
}

func AnthropicToOpenAI(req AnthropicRequest, cfg *config.Config) OpenAIRequest {
    ctx := &TransformContext{
        Provider: detectProvider(mappedModel),
        Config:   cfg,
    }
    messages := transformMessagesWithContext(req.Messages, ctx)
    // ...
}
```

### Decision
**Use `TransformContext` struct** to encapsulate provider information and configuration.

### Rationale
- Clean, thread-safe design
- Easy to extend with provider-specific settings
- Single parameter instead of multiple
- Future-proof for additional context needs

---

## Decision 3: Streaming State Management

### Context
Provider-specific logic in streaming requires maintaining state across SSE chunks (e.g., buffering Kimi K2 special tokens).

### Options Considered

#### Option A: Extend Existing Parameters
```go
func processStreamDelta(
    w, flusher, delta, contentBlockIndex,
    hasStartedTextBlock, isToolUse, currentToolCallID,
    toolCallJSONMap, provider, kimiBuffer, qwenState // More params!
)
```
- **Pros**: Minimal changes
- **Cons**: Function signature explosion (11+ parameters)

#### Option B: Map-Based State
```go
state := map[string]interface{}{
    "provider": provider,
    "kimiBuffer": buffer,
}
```
- **Pros**: Flexible
- **Cons**: Type-unsafe, error-prone

#### Option C: Structured State ✅ **SELECTED**
```go
type StreamState struct {
    ContentBlockIndex   int
    HasStartedTextBlock bool
    IsToolUse          bool
    CurrentToolCallID  string
    ToolCallJSONMap    map[string]string
    ProviderContext    *ProviderStreamContext
}

type ProviderStreamContext struct {
    Provider      Provider
    KimiBuffer    strings.Builder
    HermesState   *HermesToolState  // If needed
}

func processStreamDelta(w http.ResponseWriter, flusher http.Flusher,
                        delta OpenAIDelta, state *StreamState)
```

### Decision
**Encapsulate streaming state in `StreamState` struct** with nested `ProviderStreamContext`.

### Rationale
- Type-safe, testable
- Clear ownership of provider-specific state
- Easier to add provider-specific fields
- Reduces parameter count from 8+ to 4

---

## Decision 4: Provider Detection Strategy

### Context
Need to identify provider from model ID to apply correct transformations. Model IDs vary: `deepseek-chat`, `qwen/qwen3-coder`, `moonshot/kimi-k2`, etc.

### Approach

```go
type Provider int

const (
    ProviderDeepSeek Provider = iota
    ProviderQwen
    ProviderKimi
    ProviderStandard  // Fallback for unknown
)

func DetectProvider(modelID string) Provider {
    normalized := strings.ToLower(modelID)

    // 1. OpenRouter format: provider/model
    if parts := strings.Split(normalized, "/"); len(parts) == 2 {
        switch parts[0] {
        case "deepseek":
            return ProviderDeepSeek
        case "qwen":
            return ProviderQwen
        case "moonshot":  // Kimi's OpenRouter name
            return ProviderKimi
        }
    }

    // 2. Keyword matching with precedence: Kimi > Qwen > DeepSeek
    if strings.Contains(normalized, "kimi") || strings.Contains(normalized, "k2") {
        return ProviderKimi
    }
    if strings.Contains(normalized, "qwen") {
        return ProviderQwen
    }
    if strings.Contains(normalized, "deepseek") {
        return ProviderDeepSeek
    }

    // 3. Default to standard OpenAI format
    return ProviderStandard
}
```

### Decision Points

**Precedence Order**: Kimi > Qwen > DeepSeek > Standard
- **Rationale**: Most specific to least specific. If a model name contains multiple keywords (unlikely but possible), use the most specialized handler first.

**Fallback Behavior**: Unknown models → `ProviderStandard`
- **Rationale**: Standard OpenAI format is the most widely compatible. Better to attempt standard transformation than fail.

**Case Insensitivity**: Always normalize to lowercase
- **Rationale**: Model IDs may vary in casing across providers

---

## Decision 5: Provider-Specific Transformation Approach

### Kimi K2: Response-Side Parsing Only

**Decision**: Parse special tokens from OpenRouter **responses**, no request-side transformation.

```go
func parseKimiToolCalls(content string) []ToolCall {
    if !strings.Contains(content, "<|tool_calls_section_begin|>") {
        return nil
    }

    // Extract tool calls section
    pattern := `<\|tool_calls_section_begin\|>(.*?)<\|tool_calls_section_end\|>`
    sections := regexp.MustCompile(pattern).FindStringSubmatch(content)

    // Parse individual tool calls
    callPattern := `<\|tool_call_begin\|>\s*(?P<id>[\w\.]+:\d+)\s*<\|tool_call_argument_begin\|>\s*(?P<args>.*?)\s*<\|tool_call_end\|>`
    // ... parsing logic
}
```

**Rationale**:
- Kimi accepts standard OpenAI tool definitions in requests
- Special tokens only appear in responses from OpenRouter
- No need to modify outgoing requests

**Streaming Consideration**:
```go
type KimiStreamBuffer struct {
    buffer        strings.Builder
    maxSize       int  // 10KB limit
    inToolSection bool
}

func (b *KimiStreamBuffer) Append(chunk string) (complete bool, toolCalls []ToolCall, err error) {
    b.buffer.WriteString(chunk)

    if b.buffer.Len() > b.maxSize {
        return false, nil, errors.New("kimi tool call buffer exceeded 10KB limit")
    }

    if strings.Contains(b.buffer.String(), "<|tool_calls_section_end|>") {
        toolCalls := parseKimiToolCalls(b.buffer.String())
        return true, toolCalls, nil
    }

    return false, nil, nil
}
```

### Qwen3-Coder: Dual Format Acceptance

**Decision**: Accept both `tool_calls` array AND `function_call` object from OpenRouter responses.

```go
func parseQwenToolCall(response OpenAIResponse) []ToolCall {
    var toolCalls []ToolCall

    // Format 1: OpenAI-compatible tool_calls array (vLLM with hermes parser)
    if len(response.Message.ToolCalls) > 0 {
        toolCalls = response.Message.ToolCalls
    }

    // Format 2: Qwen-Agent function_call object
    if response.Message.FunctionCall != nil {
        toolCalls = []ToolCall{{
            ID:   generateID(),  // Generate synthetic ID
            Type: "function",
            Function: Function{
                Name:      response.Message.FunctionCall.Name,
                Arguments: response.Message.FunctionCall.Arguments,
            },
        }}
    }

    return toolCalls
}
```

**Rationale**:
- OpenRouter may return either format depending on backend configuration
- Accepting both ensures compatibility across OpenRouter's infrastructure changes
- Always output OpenAI-compatible format for consistency

### DeepSeek: Passthrough

**Decision**: No transformation needed, existing logic works as-is.

**Rationale**:
- DeepSeek via OpenRouter uses pure OpenAI format
- This becomes the default/fallback behavior
- Simplifies implementation

---

## Decision 6: Error Handling Strategy

### HTTP Status Code Mapping

| Error Type | Status Code | When | Example |
|------------|-------------|------|---------|
| **Client Error - Invalid Input** | 400 | Malformed tool definition, invalid schema | "Tool parameter 'location' missing required type" |
| **Client Error - Validation** | 400 | Tool result without matching call | "Tool result references unknown tool_call_id" |
| **Server Error - Transformation** | 500 | Provider parsing logic fails unexpectedly | "Failed to parse Kimi special tokens: regex error" |
| **Gateway Error - Provider Issue** | 502 | OpenRouter returns malformed response | "OpenRouter returned incomplete tool call" |
| **Gateway Timeout** | 504 | Kimi buffer exceeds timeout | "Tool call buffering exceeded 10KB limit" |

### Streaming Error Handling

**Decision**: Send error SSE event and gracefully terminate stream.

```go
func sendStreamError(w http.ResponseWriter, flusher http.Flusher, err error) {
    event := map[string]interface{}{
        "type": "error",
        "error": map[string]interface{}{
            "type":    "provider_transformation_error",
            "message": err.Error(),
        },
    }

    data, _ := json.Marshal(event)
    fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
    flusher.Flush()

    // Send stream end event
    fmt.Fprintf(w, "event: message_stop\ndata: {}\n\n")
    flusher.Flush()
}
```

**Rationale**:
- Anthropic SSE format supports error events
- Client can handle error gracefully instead of connection timeout
- Provides clear error message for debugging

---

## Decision 7: Buffer Management (Kimi K2 Streaming)

### Context
Kimi special tokens may be split across multiple SSE chunks. Need to buffer until complete section received.

### Approach

**Buffer Limit**: 10KB per tool call section
**Timeout**: Implicit via HTTP request timeout (no additional timeout needed)

```go
const kimiBufferLimit = 10 * 1024  // 10KB

func handleKimiStreaming(state *StreamState, chunk string) error {
    state.ProviderContext.KimiBuffer.WriteString(chunk)

    if state.ProviderContext.KimiBuffer.Len() > kimiBufferLimit {
        return fmt.Errorf("kimi tool call buffer exceeded %d bytes", kimiBufferLimit)
    }

    content := state.ProviderContext.KimiBuffer.String()
    if strings.Contains(content, "<|tool_calls_section_end|>") {
        // Parse complete tool calls
        toolCalls := parseKimiToolCalls(content)

        // Convert to Anthropic SSE events
        sendAnthropicToolCallEvents(toolCalls, state)

        // Clear buffer
        state.ProviderContext.KimiBuffer.Reset()
    }

    return nil
}
```

### Decision Points

**Why 10KB?**
- Typical tool call: ~500 bytes to 2KB
- 10KB allows for 5-20 tool calls buffered
- Large enough for complex scenarios
- Small enough to prevent memory issues

**No Separate Timeout**:
- HTTP request timeout (default 2 minutes) handles hung connections
- Additional timeout adds complexity without significant benefit
- 10KB limit provides implicit "reasonableness" check

---

## Decision 8: Configuration Design

### Optional Provider Override

**Decision**: Support manual provider override in config, but make it optional.

```yaml
# athena.yml
providers:
  # Override provider detection (optional)
  provider_override:
    "anthropic/claude-3-opus": "qwen"      # Force Qwen handler
    "custom-deepseek-model": "deepseek"    # Explicit mapping

  # Provider-specific settings (optional)
  kimi_k2:
    buffer_limit_kb: 10
    start_token: "<|tool_calls_section_begin|>"  # Override if format changes

  qwen_hermes:
    context_limit_kb: 100  # Warn when approaching limit
```

**Priority Order**:
1. `provider_override` (if configured)
2. Auto-detection from model ID
3. Fallback to Standard

**Rationale**:
- Most users won't need overrides (auto-detection works)
- Provides escape hatch for edge cases
- Allows adapting to provider API changes via config
- Low priority for initial implementation

---

## Decision 9: Testing Strategy

### Test Organization

```
internal/transform/
├── providers_test.go        # Provider detection tests
├── transform_kimi_test.go   # Kimi K2 parsing tests
├── transform_qwen_test.go   # Qwen Hermes tests
├── transform_test.go         # Core transformation tests
└── streaming_test.go         # Streaming scenarios
```

### Coverage Requirements

**Provider Detection** (12 test cases):
- OpenRouter ID format (`deepseek/deepseek-r1` → DeepSeek)
- Keyword detection (`kimi-k2-instruct` → Kimi)
- Case insensitivity (`DeepSeek` vs `deepseek`)
- Ambiguous names (multiple keywords → precedence)
- Unknown models → Standard fallback

**Kimi K2 Parsing** (15 test cases):
- Single tool call
- Multiple tool calls
- Streaming: complete in one chunk
- Streaming: split across chunks
- Streaming: buffer limit exceeded
- Malformed special tokens
- Missing end token

**Qwen Hermes** (12 test cases):
- `tool_calls` array format
- `function_call` object format
- Mixed formats in conversation
- Context limit scenarios

**Streaming** (20 test cases per provider):
- Single tool call streamed
- Multiple simultaneous tool calls
- Tool call + text content
- Error mid-stream
- Buffer edge cases

**Total**: 80-100 test cases

---

## Decision 10: Implementation Phases

### Phase 1: Foundation (Week 1)
**Goal**: Get DeepSeek working, establish patterns

- Implement provider detection (`DetectProvider`)
- Add `TransformContext` struct
- Verify DeepSeek passthrough works
- Unit tests for provider detection (all edge cases)
- **Deliverable**: DeepSeek tool calling confirmed working

### Phase 2: Kimi K2 (Week 2)
**Goal**: Implement special token parsing

- Implement `parseKimiToolCalls` for non-streaming
- Add `KimiStreamBuffer` for streaming
- Kimi-specific tests (15+ cases)
- **Deliverable**: Kimi tool calling works (streaming & non-streaming)

### Phase 3: Qwen Hermes (Week 2-3)
**Goal**: Handle dual format acceptance

- Implement `parseQwenToolCall` (both formats)
- Handle Qwen streaming deltas
- Qwen-specific tests (12+ cases)
- **Deliverable**: All three providers working

### Phase 4: Hardening (Week 3)
**Goal**: Production-ready

- Edge case handling (all error scenarios from matrix)
- Integration tests (full request/response cycles)
- Performance benchmarks (<1ms transformation overhead)
- Documentation updates (README, examples)
- **Deliverable**: Production-ready implementation

---

## Risks and Mitigations

### Risk 1: OpenRouter Format Changes
**Likelihood**: Medium (30% over 6 months)
**Impact**: High (breaks tool calling)

**Mitigation**:
- Configuration overrides for special tokens
- Graceful degradation to standard format
- Clear error messages for debugging
- Version detection (future enhancement)

### Risk 2: Provider Detection Ambiguity
**Likelihood**: Low (15%)
**Impact**: Medium (wrong transformation applied)

**Mitigation**:
- Well-defined precedence order
- Manual override configuration
- Extensive test coverage (12 detection tests)
- Logging of detected provider for debugging

### Risk 3: Buffer Memory Issues
**Likelihood**: Low (10%)
**Impact**: Low (single request affected)

**Mitigation**:
- Hard 10KB buffer limit
- Error on exceed (502 response)
- Per-request state (no global accumulation)

### Risk 4: Performance Degradation
**Likelihood**: Low (20%)
**Impact**: Low (<5ms added latency acceptable)

**Mitigation**:
- Benchmark early (target: <1ms overhead)
- Cache provider detection result
- Optimize hot paths (regex compilation)
- Performance tests in Phase 4

---

## Future Enhancements

### Not in Initial Scope, Consider Later

1. **Additional Providers**
   - LLaMA-based models
   - Mistral tool calling
   - Gemini function calling

2. **Provider Versioning**
   - Detect provider API version from response
   - Apply version-specific transformations
   - Example: `kimi-k2-v2` with different tokens

3. **Metrics & Monitoring**
   - Provider detection frequency (which providers used)
   - Transformation error rates by provider
   - Buffer usage statistics (Kimi streaming)

4. **Advanced Configuration**
   - Per-model timeout overrides
   - Custom regex patterns for special tokens
   - Debug mode (log all transformations)

---

## References

- **Specification**: `docs/specs/toolcalling/spec.md`
- **Provider Formats**: `docs/specs/toolcalling/provider-formats.md`
- **Athena Standards**: `docs/standards/tech.md`, `docs/standards/practices.md`
- **Implementation Target**: `internal/transform/` package

---

## Approval Required Before Implementation

**Questions for Product/Stakeholders**:

1. **Provider Priority**: If resource constraints require phasing, what's the priority? (Recommend: DeepSeek → Kimi → Qwen)

2. **Fallback Strategy**: Should unknown models attempt standard OpenAI format (current plan) or return error early?

3. **Breaking Changes**: If provider-specific transformations affect existing behavior, is that acceptable?

4. **Configuration Exposure**: Should provider override config be documented for users, or kept as internal escape hatch?

**Sign-Off Required From**:
- [ ] Technical Lead (architecture decisions)
- [ ] Product Owner (scope, priorities)
- [ ] Engineering Team (implementation feasibility)

---

**Document Version**: 1.0
**Last Updated**: 2025-10-04
**Status**: Ready for Review → Technical Design Phase
