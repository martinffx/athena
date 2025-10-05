package transform

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sendStreamError sends an error SSE event and terminates the stream
func sendStreamError(w http.ResponseWriter, flusher http.Flusher, errorType string, message string) {
	errorEvent := map[string]interface{}{
		"type": "error",
		"error": map[string]interface{}{
			"type":    errorType,
			"message": message,
		},
	}
	emitSSEEvent(w, "error", errorEvent)

	// Send message_stop to terminate stream
	stopEvent := map[string]interface{}{
		"type": "message_stop",
	}
	emitSSEEvent(w, "message_stop", stopEvent)

	flusher.Flush()
}

// emitSSEEvent writes a Server-Sent Event to the response writer
func emitSSEEvent(w http.ResponseWriter, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Fallback error event if marshaling fails
		fmt.Fprintf(w, "event: error\ndata: {\"type\":\"error\",\"error\":{\"message\":\"JSON marshal error\"}}\n\n")
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(jsonData))
}
