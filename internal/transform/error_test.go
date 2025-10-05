package transform

import (
	"net/http"
	"strings"
	"testing"
)

// TestOpenAIToAnthropic_KimiMalformedToolCalls tests error handling for malformed Kimi tool calls
func TestOpenAIToAnthropic_KimiMalformedToolCalls(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing section end token",
			content:     "<|tool_calls_section_begin|>incomplete",
			wantErr:     true,
			errContains: "missing section end token",
		},
		{
			name:        "invalid tool call ID format",
			content:     "<|tool_calls_section_begin|><|tool_call_begin|>invalid-id<|tool_call_argument_begin|>{}<|tool_call_end|><|tool_calls_section_end|>",
			wantErr:     true,
			errContains: "invalid Kimi tool call ID format",
		},
		{
			name:        "invalid JSON arguments",
			content:     "<|tool_calls_section_begin|><|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{invalid json}<|tool_call_end|><|tool_calls_section_end|>",
			wantErr:     true,
			errContains: "invalid JSON arguments",
		},
		{
			name:    "valid tool call",
			content: "<|tool_calls_section_begin|><|tool_call_begin|>functions.get_weather:0<|tool_call_argument_begin|>{\"city\":\"Tokyo\"}<|tool_call_end|><|tool_calls_section_end|>",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build OpenAI response with Kimi content
			resp := map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": tt.content,
						},
						"finish_reason": "stop",
					},
				},
				"model": "kimi-k2",
			}

			result, err := OpenAIToAnthropic(resp, "kimi-k2", FormatKimi)

			if tt.wantErr {
				if err == nil {
					t.Errorf("OpenAIToAnthropic() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("OpenAIToAnthropic() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("OpenAIToAnthropic() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Error("OpenAIToAnthropic() returned nil result")
				}
			}
		})
	}
}

// TestOpenAIToAnthropic_InvalidResponseStructure tests error handling for malformed OpenRouter responses
func TestOpenAIToAnthropic_InvalidResponseStructure(t *testing.T) {
	tests := []struct {
		name        string
		resp        map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing choices",
			resp:        map[string]interface{}{},
			wantErr:     true,
			errContains: "invalid OpenRouter response: missing choices",
		},
		{
			name: "empty choices array",
			resp: map[string]interface{}{
				"choices": []interface{}{},
			},
			wantErr:     true,
			errContains: "invalid OpenRouter response: empty choices",
		},
		{
			name: "missing message in choice",
			resp: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"finish_reason": "stop",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid OpenRouter response: missing message",
		},
		{
			name: "valid response",
			resp: map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Hello",
						},
						"finish_reason": "stop",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := OpenAIToAnthropic(tt.resp, "test-model", FormatStandard)

			if tt.wantErr {
				if err == nil {
					t.Errorf("OpenAIToAnthropic() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("OpenAIToAnthropic() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("OpenAIToAnthropic() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Error("OpenAIToAnthropic() returned nil result")
				}
			}
		})
	}
}

// TestHandleKimiStreaming_BufferExceeded tests buffer limit error handling
func TestHandleKimiStreaming_BufferExceeded(t *testing.T) {
	// Create large chunk that exceeds 10KB buffer limit
	largeChunk := strings.Repeat("x", 11000)

	w := &mockResponseWriter{}
	flusher := &mockFlusher{}
	state := &StreamState{
		FormatContext: &FormatStreamContext{
			Format:            FormatKimi,
			KimiBufferLimit:   10240,
			KimiInToolSection: false,
		},
	}

	err := handleKimiStreaming(w, flusher, state, largeChunk)

	if err == nil {
		t.Error("handleKimiStreaming() expected error for buffer exceeded, got nil")
		return
	}

	if !strings.Contains(err.Error(), "buffer exceeded") {
		t.Errorf("handleKimiStreaming() error = %v, want error containing 'buffer exceeded'", err)
	}

	// Should have sent error event
	output := w.String()
	if !strings.Contains(output, "event: error") {
		t.Errorf("handleKimiStreaming() expected error event in output, got: %s", output)
	}

	if !strings.Contains(output, "overloaded") {
		t.Errorf("handleKimiStreaming() expected 'overloaded' error type, got: %s", output)
	}
}

// mockResponseWriter for testing
type mockResponseWriter struct {
	strings.Builder
	header http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) WriteHeader(_ int) {}
