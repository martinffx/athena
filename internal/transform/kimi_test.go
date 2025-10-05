package transform

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	testFuncGetWeather = "get_weather"
)

func TestParseKimiToolCalls(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCalls int
		wantErr   bool
		validate  func(*testing.T, []ToolCall)
	}{
		{
			name: "single tool call",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 1,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				if calls[0].Function.Name != testFuncGetWeather {
					t.Errorf("got name %q, want %q", calls[0].Function.Name, testFuncGetWeather)
				}
				if calls[0].ID != "functions.get_weather:0" {
					t.Errorf("got ID %q, want %q", calls[0].ID, "functions.get_weather:0")
				}
				if calls[0].Type != "function" {
					t.Errorf("got type %q, want %q", calls[0].Type, "function")
				}
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
					t.Errorf("failed to parse arguments: %v", err)
				}
				if args["city"] != "Beijing" {
					t.Errorf("got city %q, want %q", args["city"], "Beijing")
				}
			},
		},
		{
			name: "multiple tool calls",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
<|tool_call_begin|>functions.get_time:1<|tool_call_argument_begin|>{"timezone": "UTC"}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 2,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				if calls[0].Function.Name != testFuncGetWeather {
					t.Errorf("call 0: got name %q, want %q", calls[0].Function.Name, testFuncGetWeather)
				}
				if calls[1].Function.Name != "get_time" {
					t.Errorf("call 1: got name %q, want %q", calls[1].Function.Name, "get_time")
				}
			},
		},
		{
			name: "nested JSON arguments",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.search:0<|tool_call_argument_begin|>{"query": "test", "filters": {"category": "tech", "date": {"from": "2024-01-01", "to": "2024-12-31"}}}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 1,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
					t.Errorf("failed to parse nested JSON: %v", err)
				}
				filters, ok := args["filters"].(map[string]interface{})
				if !ok {
					t.Error("filters should be an object")
				}
				if filters["category"] != "tech" {
					t.Errorf("got category %q, want %q", filters["category"], "tech")
				}
			},
		},
		{
			name:      "missing section begin token",
			content:   `<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>`,
			wantCalls: 0,
			wantErr:   false, // Not an error, just no tool calls
		},
		{
			name: "missing section end token",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>`,
			wantCalls: 0,
			wantErr:   true,
		},
		{
			name:      "no tool calls present",
			content:   `This is just regular text without any tool calls.`,
			wantCalls: 0,
			wantErr:   false,
		},
		{
			name: "invalid ID format - missing colon",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 0,
			wantErr:   true,
		},
		{
			name: "empty arguments",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.no_args:0<|tool_call_argument_begin|>{}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 1,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				if calls[0].Function.Name != "no_args" {
					t.Errorf("got name %q, want %q", calls[0].Function.Name, "no_args")
				}
				if calls[0].Function.Arguments != "{}" {
					t.Errorf("got arguments %q, want %q", calls[0].Function.Arguments, "{}")
				}
			},
		},
		{
			name: "tool call with array arguments",
			content: `<|tool_calls_section_begin|>
<|tool_call_begin|>functions.batch_process:0<|tool_call_argument_begin|>{"items": ["a", "b", "c"]}<|tool_call_end|>
<|tool_calls_section_end|>`,
			wantCalls: 1,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
					t.Errorf("failed to parse array arguments: %v", err)
				}
				items, ok := args["items"].([]interface{})
				if !ok || len(items) != 3 {
					t.Errorf("items should be array of length 3, got %v", args["items"])
				}
			},
		},
		{
			name: "whitespace handling",
			content: `  <|tool_calls_section_begin|>
  <|tool_call_begin|>  functions.get_weather:0  <|tool_call_argument_begin|>  {"city": "Beijing"}  <|tool_call_end|>
  <|tool_calls_section_end|>  `,
			wantCalls: 1,
			wantErr:   false,
			validate: func(t *testing.T, calls []ToolCall) {
				if calls[0].Function.Name != testFuncGetWeather {
					t.Errorf("got name %q, want %q", calls[0].Function.Name, testFuncGetWeather)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, err := parseKimiToolCalls(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKimiToolCalls() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(calls) != tt.wantCalls {
				t.Errorf("got %d calls, want %d", len(calls), tt.wantCalls)
				return
			}
			if tt.validate != nil {
				tt.validate(t, calls)
			}
		})
	}
}

func TestHandleKimiStreaming(t *testing.T) {
	tests := []struct {
		name     string
		chunks   []string
		wantErr  bool
		validate func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "complete section in one chunk",
			chunks: []string{
				`<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>
<|tool_calls_section_end|>`,
			},
			wantErr: false,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				output := w.Body.String()
				if !strings.Contains(output, "event: content_block_start") {
					t.Error("expected content_block_start event")
				}
				if !strings.Contains(output, "get_weather") {
					t.Error("expected function name in output")
				}
				if !strings.Contains(output, "event: content_block_stop") {
					t.Error("expected content_block_stop event")
				}
			},
		},
		{
			name: "section split across 2 chunks",
			chunks: []string{
				`<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>`,
				`{"city": "Beijing"}<|tool_call_end|>
<|tool_calls_section_end|>`,
			},
			wantErr: false,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				output := w.Body.String()
				if !strings.Contains(output, "event: content_block_start") {
					t.Error("expected content_block_start event")
				}
				if !strings.Contains(output, "get_weather") {
					t.Error("expected function name in output")
				}
			},
		},
		{
			name: "section split across 5 chunks",
			chunks: []string{
				`<|tool_calls_section_begin|>`,
				`<|tool_call_begin|>functions.get_weather:0`,
				`<|tool_call_argument_begin|>{"city": `,
				`"Beijing"}<|tool_call_end|>`,
				`<|tool_calls_section_end|>`,
			},
			wantErr: false,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				output := w.Body.String()
				if !strings.Contains(output, "Beijing") {
					t.Error("expected complete arguments in output")
				}
			},
		},
		{
			name: "buffer limit exceeded",
			chunks: []string{
				`<|tool_calls_section_begin|>` + strings.Repeat("x", 11000),
			},
			wantErr: true,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should have error event
				output := w.Body.String()
				if !strings.Contains(output, "event: error") {
					t.Error("expected error event for buffer overflow")
				}
			},
		},
		{
			name: "missing end token",
			chunks: []string{
				`<|tool_calls_section_begin|>
<|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{"city": "Beijing"}<|tool_call_end|>`,
				// No end token provided, simulating incomplete stream
			},
			wantErr: false, // Not an error until we try to process
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				output := w.Body.String()
				// Should buffer but not emit events yet
				if strings.Contains(output, "event: content_block_start") {
					t.Error("should not emit events until section is complete")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test response writer
			w := httptest.NewRecorder()

			// Create a no-op flusher for testing
			flusher := &noopFlusher{}

			// Initialize stream state
			state := &StreamState{
				ContentBlockIndex:   0,
				HasStartedTextBlock: false,
				IsToolUse:           false,
				CurrentToolCallID:   "",
				ToolCallJSONMap:     make(map[string]string),
				FormatContext: &FormatStreamContext{
					Format:            FormatKimi,
					KimiBufferLimit:   10 * 1024, // 10KB
					KimiInToolSection: false,
				},
			}

			// Process chunks
			var err error
			for _, chunk := range tt.chunks {
				err = handleKimiStreaming(w, flusher, state, chunk)
				if err != nil && !tt.wantErr {
					t.Errorf("handleKimiStreaming() unexpected error = %v", err)
					return
				}
				if err != nil {
					break
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("handleKimiStreaming() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, w)
			}
		})
	}
}
