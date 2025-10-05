package transform

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// mockFlusher implements http.Flusher for testing
type mockFlusher struct{}

func (m *mockFlusher) Flush() {}

// TestSendStreamError_BasicError tests basic error event emission
func TestSendStreamError_BasicError(t *testing.T) {
	w := httptest.NewRecorder()
	flusher := &mockFlusher{}

	sendStreamError(w, flusher, "invalid_request_error", "Invalid tool definition")

	output := w.Body.String()

	// Should contain error event
	if !strings.Contains(output, "event: error") {
		t.Errorf("Expected error event, got: %s", output)
	}

	// Should contain error type
	if !strings.Contains(output, "invalid_request_error") {
		t.Errorf("Expected error type 'invalid_request_error', got: %s", output)
	}

	// Should contain error message
	if !strings.Contains(output, "Invalid tool definition") {
		t.Errorf("Expected error message, got: %s", output)
	}

	// Should contain message_stop event
	if !strings.Contains(output, "event: message_stop") {
		t.Errorf("Expected message_stop event, got: %s", output)
	}
}

// TestSendStreamError_MultipleErrorTypes tests different error types
func TestSendStreamError_MultipleErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		message   string
	}{
		{
			name:      "invalid_request_error",
			errorType: "invalid_request_error",
			message:   "Malformed tool definition",
		},
		{
			name:      "internal_server_error",
			errorType: "internal_server_error",
			message:   "Regex compilation failed",
		},
		{
			name:      "upstream_error",
			errorType: "upstream_error",
			message:   "Malformed OpenRouter response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			flusher := &mockFlusher{}

			sendStreamError(w, flusher, tt.errorType, tt.message)

			output := w.Body.String()

			if !strings.Contains(output, tt.errorType) {
				t.Errorf("Expected error type '%s', got: %s", tt.errorType, output)
			}

			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected message '%s', got: %s", tt.message, output)
			}
		})
	}
}

// TestSendStreamError_EventFormat tests SSE format compliance
func TestSendStreamError_EventFormat(t *testing.T) {
	w := httptest.NewRecorder()
	flusher := &mockFlusher{}

	sendStreamError(w, flusher, "test_error", "Test message")

	output := w.Body.String()

	// Should have proper SSE format: "event: <type>\ndata: <json>\n\n"
	lines := strings.Split(output, "\n")

	// Find error event
	foundErrorEvent := false
	foundErrorData := false
	for i, line := range lines {
		if strings.HasPrefix(line, "event: error") {
			foundErrorEvent = true
			// Next line should be data
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "data: {") {
				foundErrorData = true
			}
		}
	}

	if !foundErrorEvent {
		t.Error("Expected 'event: error' line")
	}

	if !foundErrorData {
		t.Error("Expected 'data: {' line after error event")
	}

	// Find message_stop event
	foundStopEvent := false
	for _, line := range lines {
		if strings.HasPrefix(line, "event: message_stop") {
			foundStopEvent = true
		}
	}

	if !foundStopEvent {
		t.Error("Expected 'event: message_stop' line")
	}
}

// TestEmitSSEEvent_ValidJSON tests emitSSEEvent with valid data
func TestEmitSSEEvent_ValidJSON(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"type": "content_block_delta",
		"delta": map[string]string{
			"type": "text_delta",
			"text": "Hello",
		},
	}

	emitSSEEvent(w, "content_block_delta", data)

	output := w.Body.String()

	if !strings.HasPrefix(output, "event: content_block_delta\n") {
		t.Errorf("Expected event type 'content_block_delta', got: %s", output)
	}

	if !strings.Contains(output, "data: {") {
		t.Errorf("Expected JSON data, got: %s", output)
	}

	// Should end with double newline
	if !strings.HasSuffix(output, "\n\n") {
		t.Errorf("Expected double newline suffix, got: %s", output)
	}
}

// TestEmitSSEEvent_InvalidJSON tests emitSSEEvent with unmarshalable data
func TestEmitSSEEvent_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()

	// Create unmarshalable data (channels cannot be marshaled to JSON)
	data := make(chan int)

	emitSSEEvent(w, "test_event", data)

	output := w.Body.String()

	// Should emit fallback error event
	if !strings.Contains(output, "event: error") {
		t.Errorf("Expected fallback error event, got: %s", output)
	}

	if !strings.Contains(output, "JSON marshal error") {
		t.Errorf("Expected JSON marshal error message, got: %s", output)
	}
}
