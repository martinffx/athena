# Provider-Specific Tool Calling Formats

This document provides detailed specifications for tool calling formats across the three supported providers: Kimi K2, Qwen3-Coder, and DeepSeek.

## Overview

| Provider | Format Type | OpenAI Compatible | Special Handling Required |
|----------|-------------|-------------------|---------------------------|
| **DeepSeek** | Standard OpenAI | ✅ Yes | ❌ None |
| **Qwen3-Coder** | Hermes-style | ⚠️ Partial | ✅ Custom parser |
| **Kimi K2** | Special Tokens | ❌ No | ✅ Token wrapping |

---

## 1. DeepSeek (Standard OpenAI Format)

### Summary
DeepSeek uses **pure OpenAI-compatible tool calling** with no modifications required. The format follows the standard OpenAI API specification exactly.

### Tool Definition Format
```json
{
  "type": "function",
  "function": {
    "name": "get_weather",
    "description": "Get weather of a location",
    "parameters": {
      "type": "object",
      "properties": {
        "location": {
          "type": "string",
          "description": "The city and state, e.g. San Francisco, CA"
        }
      },
      "required": ["location"]
    }
  }
}
```

### Tool Call Response Format
```json
{
  "role": "assistant",
  "content": null,
  "tool_calls": [
    {
      "id": "call_abc123",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\": \"San Francisco, CA\"}"
      }
    }
  ]
}
```

### Tool Result Format
```json
{
  "role": "tool",
  "tool_call_id": "call_abc123",
  "content": "{\"temperature\": 72, \"condition\": \"sunny\"}"
}
```

### Implementation Notes
- **No transformation needed**: Existing Athena transform.go logic works as-is
- **Streaming**: Standard SSE format with `delta.tool_calls` chunks
- **Finish Reason**: Returns `"tool_calls"` when tools are invoked
- **API Endpoint**: `https://api.deepseek.com/v1/chat/completions`
- **Strict Mode**: Beta feature available at `https://api.deepseek.com/beta` with `strict: true`

### Examples

#### Example 1: Simple Tool Call
**Request:**
```json
{
  "model": "deepseek-chat",
  "messages": [
    {"role": "user", "content": "What's the weather in Tokyo?"}
  ],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_weather",
        "description": "Get current weather",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string"}
          },
          "required": ["location"]
        }
      }
    }
  ]
}
```

**Response:**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "tool_calls": [{
        "id": "call_xyz",
        "type": "function",
        "function": {
          "name": "get_weather",
          "arguments": "{\"location\": \"Tokyo, Japan\"}"
        }
      }]
    },
    "finish_reason": "tool_calls"
  }]
}
```

#### Example 2: Multiple Tool Calls
**Response with multiple tools:**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "tool_calls": [
        {
          "id": "call_1",
          "type": "function",
          "function": {
            "name": "get_weather",
            "arguments": "{\"location\": \"Tokyo\"}"
          }
        },
        {
          "id": "call_2",
          "type": "function",
          "function": {
            "name": "get_forecast",
            "arguments": "{\"location\": \"Tokyo\", \"days\": 3}"
          }
        }
      ]
    },
    "finish_reason": "tool_calls"
  }]
}
```

---

## 2. Qwen3-Coder (Hermes-Style Format)

### Summary
Qwen3-Coder uses **Hermes-style tool calling** which requires the `hermes` tool parser when using vLLM. The format differs from standard OpenAI in structure and field naming.

### Tool Definition Format (Same as OpenAI)
```json
{
  "type": "function",
  "function": {
    "name": "get_current_temperature",
    "description": "Get current temperature at a location",
    "parameters": {
      "type": "object",
      "properties": {
        "location": {
          "type": "string",
          "description": "City, State, Country format"
        },
        "unit": {
          "type": "string",
          "enum": ["celsius", "fahrenheit"]
        }
      },
      "required": ["location"]
    }
  }
}
```

### Tool Call Response Format (Hermes)
**Via vLLM with hermes parser:**
```json
{
  "role": "assistant",
  "content": null,
  "tool_calls": [
    {
      "id": "chatcmpl-tool-924d705a",
      "type": "function",
      "function": {
        "name": "get_current_temperature",
        "arguments": "{\"location\": \"San Francisco, CA, USA\"}"
      }
    }
  ]
}
```

**Note:** The Hermes format is **automatically parsed by vLLM** when using:
```bash
vllm serve Qwen/Qwen3-Coder-480B-A35B-Instruct \
  --enable-auto-tool-choice \
  --tool-call-parser hermes
```

### Tool Result Format
```json
{
  "role": "tool",
  "tool_call_id": "chatcmpl-tool-924d705a",
  "content": "{\"temperature\": 26.1, \"location\": \"San Francisco, CA, USA\", \"unit\": \"celsius\"}"
}
```

**Alternative (Qwen-Agent format):**
```json
{
  "role": "function",
  "name": "get_current_temperature",
  "content": "{\"temperature\": 26.1, \"unit\": \"celsius\"}"
}
```

### Streaming Format
Qwen3-Coder streaming follows OpenAI delta pattern:
```json
{"choices": [{"delta": {"tool_calls": [{"index": 0, "id": "chatcmpl-tool-", "type": "function", "function": {"name": ""}}]}}]}
{"choices": [{"delta": {"tool_calls": [{"index": 0, "function": {"name": "get_"}}]}}]}
{"choices": [{"delta": {"tool_calls": [{"index": 0, "function": {"name": "current_temperature"}}]}}]}
{"choices": [{"delta": {"tool_calls": [{"index": 0, "function": {"arguments": "{\"location\""}}]}}]}
```

### Implementation Notes
- **vLLM Required**: Use `--tool-call-parser hermes` parameter
- **Context Window**: 256K native, extendable to 1M (but tool use may reduce effective context to ~33K in some cases)
- **Finish Reason**: Returns `"tool_calls"` when tools invoked
- **Known Issues**:
  - Qwen2.5-Coder has unreliable tool calling (GitHub #180) - **avoid**
  - Qwen3/Qwen3-Coder has dramatically improved reliability
  - Context approaching limits may cause "nonsense" generation
- **API Endpoint**: DashScope `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`

### Examples

#### Example 1: Single Tool Call (No-Think Mode)
**Request:**
```json
{
  "model": "qwen3-coder-plus",
  "messages": [
    {"role": "user", "content": "What's the temperature in Beijing?"}
  ],
  "tools": [...],
  "extra_body": {
    "chat_template_kwargs": {"enable_thinking": false}
  }
}
```

**Response:**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "function_call": {
        "name": "get_current_temperature",
        "arguments": "{\"location\": \"Beijing, China\"}"
      }
    },
    "finish_reason": "tool_calls"
  }]
}
```

#### Example 2: Think Mode (With Reasoning)
**Response with reasoning:**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": null,
      "reasoning_content": "The user wants to know the temperature in Beijing. I should use the get_current_temperature function with location set to Beijing, China.",
      "function_call": {
        "name": "get_current_temperature",
        "arguments": "{\"location\": \"Beijing, China\"}"
      }
    },
    "finish_reason": "tool_calls"
  }]
}
```

#### Example 3: Multiple Tool Calls
**Response:**
```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "tool_calls": [
        {
          "id": "chatcmpl-tool-1",
          "function": {
            "name": "get_current_temperature",
            "arguments": "{\"location\": \"Beijing\"}"
          }
        },
        {
          "id": "chatcmpl-tool-2",
          "function": {
            "name": "get_temperature_date",
            "arguments": "{\"location\": \"Beijing\", \"date\": \"2025-10-05\"}"
          }
        }
      ]
    },
    "finish_reason": "tool_calls"
  }]
}
```

---

## 3. Kimi K2 (Special Token Format)

### Summary
Kimi K2 uses **proprietary special tokens** to wrap tool calls. This format is **NOT compatible** with standard OpenAI parsers and requires custom handling.

### Special Tokens
- `<|tool_calls_section_begin|>` - Start of tool calls section
- `<|tool_calls_section_end|>` - End of tool calls section
- `<|tool_call_begin|>` - Start of individual tool call
- `<|tool_call_end|>` - End of individual tool call
- `<|tool_call_argument_begin|>` - Separator between tool ID and arguments

### Tool Definition Format (Same as OpenAI)
```json
{
  "type": "function",
  "function": {
    "name": "get_weather",
    "description": "Get weather information",
    "parameters": {
      "type": "object",
      "required": ["city"],
      "properties": {
        "city": {
          "type": "string",
          "description": "City name"
        }
      }
    }
  }
}
```

### Raw Model Output Format
When Kimi K2 makes a tool call, the raw output looks like:
```
<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
<|tool_calls_section_end|>
```

### Tool ID Format
The tool ID follows the pattern: `functions.{func_name}:{idx}`
- **Prefix**: Always `functions.`
- **Function Name**: Extracted between `.` and `:`
- **Index**: Sequential number for multiple calls

Examples:
- `functions.get_weather:0` → function name: `get_weather`, index: `0`
- `functions.calculate:1` → function name: `calculate`, index: `1`

### Parsed Tool Call Format (OpenAI-compatible)
After parsing the special tokens, convert to:
```json
{
  "id": "functions.get_weather:0",
  "type": "function",
  "function": {
    "name": "get_weather",
    "arguments": "{\"city\": \"Beijing\"}"
  }
}
```

### Tool Result Format
```json
{
  "role": "tool",
  "tool_call_id": "functions.get_weather:0",
  "name": "get_weather",
  "content": "{\"temperature\": 24, \"condition\": \"sunny\"}"
}
```

### Parsing Logic (Python Reference)
```python
import re
import json

def extract_tool_call_info(tool_call_rsp: str):
    """Extract tool calls from Kimi K2 special token format."""
    if '<|tool_calls_section_begin|>' not in tool_call_rsp:
        return []

    # Extract tool calls section
    pattern = r"<\|tool_calls_section_begin\|>(.*?)<\|tool_calls_section_end\|>"
    tool_calls_sections = re.findall(pattern, tool_call_rsp, re.DOTALL)

    # Extract individual tool calls
    func_call_pattern = r"<\|tool_call_begin\|>\s*(?P<tool_call_id>[\w\.]+:\d+)\s*<\|tool_call_argument_begin\|>\s*(?P<function_arguments>.*?)\s*<\|tool_call_end\|>"

    tool_calls = []
    for match in re.findall(func_call_pattern, tool_calls_sections[0], re.DOTALL):
        function_id, function_args = match
        # Parse: functions.get_weather:0 → get_weather
        function_name = function_id.split('.')[1].split(':')[0]

        tool_calls.append({
            "id": function_id,
            "type": "function",
            "function": {
                "name": function_name,
                "arguments": function_args
            }
        })

    return tool_calls
```

### Streaming Format
In streaming mode, special tokens may be split across chunks:
```
Chunk 1: "<|tool_calls_section_begin|>\n<|tool_call_begin|>fun"
Chunk 2: "ctions.get_weather:0<|tool_call_argument_begin|>{\"ci"
Chunk 3: "ty\": \"Beijing\"}<|tool_call_end|>\n<|tool_calls_section_end|>"
```

**Buffering required:** Accumulate chunks until `<|tool_calls_section_end|>` is received.

### Implementation Notes
- **Provider-Specific**: Moonshot official API (https://api.moonshot.ai/v1) handles parsing automatically
- **Third-Party Failures**: Groq, OpenRouter, LiteLLM, etc. **do NOT parse** these tokens correctly
- **Manual Parsing Required**: When using non-Moonshot providers, must implement custom parsing
- **Finish Reason**: Returns `"tool_calls"` when tools invoked
- **Context Window**: 256K tokens (0905 version)
- **Performance**: Slower inference (~34 tokens/sec vs 91 for Claude)

### Examples

#### Example 1: Single Tool Call
**Raw Response:**
```
<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Tokyo", "unit": "celsius"}<|tool_call_end|>
<|tool_calls_section_end|>
```

**Parsed:**
```json
{
  "role": "assistant",
  "tool_calls": [{
    "id": "functions.get_weather:0",
    "type": "function",
    "function": {
      "name": "get_weather",
      "arguments": "{\"city\": \"Tokyo\", \"unit\": \"celsius\"}"
    }
  }],
  "finish_reason": "tool_calls"
}
```

#### Example 2: Multiple Tool Calls
**Raw Response:**
```
<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_current_temperature:0<|tool_call_argument_begin|>{"location": "San Francisco, CA, USA"}<|tool_call_end|>
<|tool_call_begin|>functions.get_temperature_date:1<|tool_call_argument_begin|>{"location": "San Francisco, CA, USA", "date": "2025-10-05"}<|tool_call_end|>
<|tool_calls_section_end|>
```

**Parsed:**
```json
{
  "role": "assistant",
  "tool_calls": [
    {
      "id": "functions.get_current_temperature:0",
      "type": "function",
      "function": {
        "name": "get_current_temperature",
        "arguments": "{\"location\": \"San Francisco, CA, USA\"}"
      }
    },
    {
      "id": "functions.get_temperature_date:1",
      "type": "function",
      "function": {
        "name": "get_temperature_date",
        "arguments": "{\"location\": \"San Francisco, CA, USA\", \"date\": \"2025-10-05\"}"
      }
    }
  ],
  "finish_reason": "tool_calls"
}
```

#### Example 3: No Tool Call (Plain Text)
**Raw Response:**
```
I'll help you check the weather, but I need to know which city you're interested in.
```

**Parsed:**
```json
{
  "role": "assistant",
  "content": "I'll help you check the weather, but I need to know which city you're interested in.",
  "finish_reason": "stop"
}
```

---

## Provider Detection Strategy

### Recommended Approach
```go
type Provider int

const (
    ProviderDeepSeek Provider = iota
    ProviderQwen
    ProviderKimi
    ProviderStandard  // Fallback for unknown models
)

func DetectProvider(modelID string) Provider {
    normalized := strings.ToLower(modelID)

    // 1. Check OpenRouter format: provider/model
    if parts := strings.Split(normalized, "/"); len(parts) == 2 {
        switch parts[0] {
        case "deepseek":
            return ProviderDeepSeek
        case "qwen":
            return ProviderQwen
        case "moonshot":  // Kimi's OpenRouter provider name
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

### Detection Examples
| Model ID | Detected Provider | Rationale |
|----------|-------------------|-----------|
| `deepseek-chat` | DeepSeek | Contains "deepseek" |
| `deepseek/deepseek-r1` | DeepSeek | OpenRouter format, provider = "deepseek" |
| `qwen3-coder-plus` | Qwen | Contains "qwen" |
| `qwen/qwen3-coder-480b` | Qwen | OpenRouter format, provider = "qwen" |
| `kimi-k2-instruct` | Kimi | Contains "kimi" |
| `moonshot/kimi-k2` | Kimi | OpenRouter format, provider = "moonshot" |
| `claude-3-opus` | Standard | No provider match → fallback |
| `gpt-4` | Standard | No provider match → fallback |

---

## Transformation Pipeline

### Request Flow
```
Anthropic Request
    ↓
Detect Provider (based on mapped model ID)
    ↓
Transform to OpenAI format
    ↓
Apply Provider-Specific Transformations:
    - DeepSeek: None (already OpenAI format)
    - Qwen: Ensure Hermes compatibility
    - Kimi: No pre-request transformation (special tokens in response only)
    ↓
Send to OpenRouter
    ↓
Receive Response
    ↓
Parse Provider-Specific Format:
    - DeepSeek: Standard OpenAI parsing
    - Qwen: Hermes parser (via vLLM or manual)
    - Kimi: Extract special tokens → convert to OpenAI format
    ↓
Transform back to Anthropic format
    ↓
Return to client
```

### Streaming Flow
```
Start Streaming
    ↓
Detect Provider (cached from request)
    ↓
For each SSE chunk:
    ↓
    Provider-Specific Chunk Processing:
        - DeepSeek: Standard delta.tool_calls processing
        - Qwen: Hermes delta processing
        - Kimi: Buffer until complete tool call section
    ↓
    Transform to Anthropic SSE events
    ↓
    Send to client
```

---

## Key Differences Summary

### Tool Definition
- **All providers**: Use standard OpenAI JSON schema format ✅

### Tool Call Response
- **DeepSeek**: Standard `tool_calls` array with `id`, `type`, `function`
- **Qwen**: `tool_calls` array (via vLLM parser) OR `function_call` object (via Qwen-Agent)
- **Kimi**: Raw special tokens requiring manual parsing

### Tool Result
- **DeepSeek**: `role: "tool"` with `tool_call_id`
- **Qwen**: `role: "tool"` with `tool_call_id` OR `role: "function"` with `name`
- **Kimi**: `role: "tool"` with `tool_call_id` (Kimi format ID like `functions.name:0`)

### Finish Reason
- **All providers**: `"tool_calls"` when tools invoked ✅

### Streaming
- **DeepSeek**: Standard OpenAI delta chunks
- **Qwen**: OpenAI-like delta chunks (parsed by vLLM/Qwen-Agent)
- **Kimi**: Special tokens may split across chunks → buffering required

### Context Window
- **DeepSeek**: 128K tokens
- **Qwen**: 256K native, 1M extendable (but effective ~33K with many tools)
- **Kimi**: 256K tokens

---

## Testing Checklist

### Per Provider
- [ ] Single tool call (simple arguments)
- [ ] Multiple tool calls in one response
- [ ] Tool call with complex nested objects
- [ ] Tool call with array arguments
- [ ] Tool result processing
- [ ] Multi-turn conversation with tools
- [ ] Streaming single tool call
- [ ] Streaming multiple tool calls
- [ ] Tool call + text content mixed
- [ ] Error: malformed tool call
- [ ] Error: unknown tool name
- [ ] Error: invalid arguments

### Cross-Provider
- [ ] Provider detection accuracy
- [ ] Fallback to standard format
- [ ] Format conversion fidelity (Anthropic ↔ Provider ↔ Anthropic)
- [ ] Streaming state management across providers
- [ ] Context window limits

---

## References

### Official Documentation
- **Kimi K2**: https://huggingface.co/moonshotai/Kimi-K2-Instruct/blob/main/docs/tool_call_guidance.md
- **Qwen3-Coder**: https://qwen.readthedocs.io/en/latest/framework/function_call.html
- **DeepSeek**: https://api-docs.deepseek.com/guides/function_calling

### Known Issues
- **Kimi K2**: GitHub issues #929, #1037 (SST OpenCode), #12679 (LiteLLM), #2450 (Avante.nvim) - third-party provider failures
- **Qwen2.5-Coder**: GitHub #180 - unreliable function calling (fixed in Qwen3)
- **Qwen3-Coder**: Context window "nonsense" generation when approaching limits

### Provider APIs
- **DeepSeek**: https://api.deepseek.com/v1
- **Qwen (DashScope)**: https://dashscope-intl.aliyuncs.com/compatible-mode/v1
- **Kimi (Moonshot)**: https://api.moonshot.ai/v1
