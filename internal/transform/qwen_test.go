package transform

import (
	"encoding/json"
	"testing"
)

func TestParseQwenToolCall(t *testing.T) {
	tests := []struct {
		name     string
		delta    map[string]interface{}
		expected []ToolCall
		wantErr  bool
	}{
		{
			name: "tool_calls array format - single call",
			delta: map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id":   "call-123",
						"type": "function",
						"function": map[string]interface{}{
							"name":      "get_weather",
							"arguments": `{"city":"Tokyo"}`,
						},
					},
				},
			},
			expected: []ToolCall{
				{
					ID:   "call-123",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "get_weather",
						Arguments: `{"city":"Tokyo"}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function_call object format",
			delta: map[string]interface{}{
				"function_call": map[string]interface{}{
					"name":      "get_weather",
					"arguments": `{"city":"Beijing"}`,
				},
			},
			expected: []ToolCall{
				{
					// ID will be synthetic, we'll check it's not empty
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "get_weather",
						Arguments: `{"city":"Beijing"}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tool_calls array - multiple calls",
			delta: map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id":   "call-1",
						"type": "function",
						"function": map[string]interface{}{
							"name":      "get_weather",
							"arguments": `{"city":"Tokyo"}`,
						},
					},
					map[string]interface{}{
						"id":   "call-2",
						"type": "function",
						"function": map[string]interface{}{
							"name":      "get_time",
							"arguments": `{"timezone":"Asia/Tokyo"}`,
						},
					},
				},
			},
			expected: []ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "get_weather",
						Arguments: `{"city":"Tokyo"}`,
					},
				},
				{
					ID:   "call-2",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "get_time",
						Arguments: `{"timezone":"Asia/Tokyo"}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "empty delta",
			delta:    map[string]interface{}{},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "tool_calls array empty",
			delta: map[string]interface{}{
				"tool_calls": []interface{}{},
			},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "function_call with nested JSON arguments",
			delta: map[string]interface{}{
				"function_call": map[string]interface{}{
					"name":      "complex_function",
					"arguments": `{"nested":{"key":"value"},"array":[1,2,3]}`,
				},
			},
			expected: []ToolCall{
				{
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "complex_function",
						Arguments: `{"nested":{"key":"value"},"array":[1,2,3]}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tool_calls with missing id field",
			delta: map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"type": "function",
						"function": map[string]interface{}{
							"name":      "test_func",
							"arguments": `{}`,
						},
					},
				},
			},
			expected: []ToolCall{
				{
					ID:   "", // Missing ID should result in empty string
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "test_func",
						Arguments: `{}`,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function_call with missing arguments",
			delta: map[string]interface{}{
				"function_call": map[string]interface{}{
					"name": "test_func",
				},
			},
			expected: []ToolCall{
				{
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      "test_func",
						Arguments: "", // Missing arguments should result in empty string
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseQwenToolCall(tt.delta)

			if (got == nil) != (tt.expected == nil) {
				t.Errorf("parseQwenToolCall() returned nil = %v, want nil = %v",
					got == nil, tt.expected == nil)
				return
			}

			if got == nil {
				return
			}

			if len(got) != len(tt.expected) {
				t.Errorf("parseQwenToolCall() returned %d tool calls, want %d",
					len(got), len(tt.expected))
				return
			}

			for i := range got {
				// For function_call format, ID is synthetic, just check it's not empty
				if tt.name == "function_call object format" ||
					tt.name == "function_call with nested JSON arguments" ||
					tt.name == "function_call with missing arguments" {
					if got[i].ID == "" {
						t.Errorf("parseQwenToolCall()[%d].ID is empty, expected synthetic ID", i)
					}
				} else if got[i].ID != tt.expected[i].ID {
					t.Errorf("parseQwenToolCall()[%d].ID = %v, want %v",
						i, got[i].ID, tt.expected[i].ID)
				}

				if got[i].Type != tt.expected[i].Type {
					t.Errorf("parseQwenToolCall()[%d].Type = %v, want %v",
						i, got[i].Type, tt.expected[i].Type)
				}

				if got[i].Function.Name != tt.expected[i].Function.Name {
					t.Errorf("parseQwenToolCall()[%d].Function.Name = %v, want %v",
						i, got[i].Function.Name, tt.expected[i].Function.Name)
				}

				// Compare JSON arguments
				var gotArgs, expectedArgs interface{}
				if got[i].Function.Arguments != "" {
					if err := json.Unmarshal([]byte(got[i].Function.Arguments), &gotArgs); err != nil {
						t.Errorf("parseQwenToolCall()[%d].Function.Arguments is not valid JSON: %v", i, err)
					}
				}
				if tt.expected[i].Function.Arguments != "" {
					if err := json.Unmarshal([]byte(tt.expected[i].Function.Arguments), &expectedArgs); err != nil {
						t.Errorf("expected[%d].Function.Arguments is not valid JSON: %v", i, err)
					}
				}

				gotJSON, _ := json.Marshal(gotArgs)
				expectedJSON, _ := json.Marshal(expectedArgs)
				if string(gotJSON) != string(expectedJSON) && got[i].Function.Arguments != tt.expected[i].Function.Arguments {
					t.Errorf("parseQwenToolCall()[%d].Function.Arguments = %v, want %v",
						i, got[i].Function.Arguments, tt.expected[i].Function.Arguments)
				}
			}
		})
	}
}
