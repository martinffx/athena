package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"athena/internal/config"
	"athena/internal/transform"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Port:   "8080",
		APIKey: "test-key",
	}

	srv := New(cfg)

	if srv == nil {
		t.Fatal("New() returned nil")
	}

	if srv.cfg != cfg {
		t.Error("Server config not set correctly")
	}
}

func TestHandleHealth(t *testing.T) {
	cfg := &config.Config{}
	srv := New(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Health status = %q, expected %q", result["status"], "ok")
	}
}

func TestHandleMessages_InvalidMethod(t *testing.T) {
	cfg := &config.Config{}
	srv := New(cfg)

	req := httptest.NewRequest("GET", "/v1/messages", nil)
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestHandleMessages_InvalidJSON(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://test.api.com",
	}
	srv := New(cfg)

	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleMessages_NonStreaming(t *testing.T) {
	// Create a test server that will act as OpenRouter
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %q, expected %q", r.Header.Get("Authorization"), "Bearer test-key")
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, expected %q", r.Header.Get("Content-Type"), "application/json")
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var openAIReq map[string]interface{}
		if err := json.Unmarshal(body, &openAIReq); err != nil {
			t.Fatalf("Failed to parse OpenAI request: %v", err)
		}

		// Verify model mapping happened
		if openAIReq["model"] != "test/sonnet" {
			t.Errorf("Model = %q, expected %q", openAIReq["model"], "test/sonnet")
		}

		// Send OpenAI response
		response := map[string]interface{}{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "test/sonnet",
			"choices": []interface{}{
				map[string]interface{}{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello! How can I help you today?",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer openRouterServer.Close()

	cfg := &config.Config{
		APIKey:      "test-key",
		BaseURL:     openRouterServer.URL,
		SonnetModel: "test/sonnet",
	}
	srv := New(cfg)

	reqBody := transform.AnthropicRequest{
		Model: "claude-3-5-sonnet",
		Messages: []transform.Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
		Stream: false,
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Status code = %d, expected %d. Body: %s", resp.StatusCode, http.StatusOK, body)
	}

	var anthropicResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify Anthropic response format
	if anthropicResp["type"] != "message" {
		t.Errorf("Response type = %v, expected %q", anthropicResp["type"], "message")
	}

	if anthropicResp["role"] != "assistant" {
		t.Errorf("Response role = %v, expected %q", anthropicResp["role"], "assistant")
	}

	content, ok := anthropicResp["content"].([]interface{})
	if !ok {
		t.Fatalf("Response content is not an array")
	}

	if len(content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(content))
	}

	contentBlock := content[0].(map[string]interface{})
	if contentBlock["type"] != "text" {
		t.Errorf("Content type = %v, expected %q", contentBlock["type"], "text")
	}

	if contentBlock["text"] != "Hello! How can I help you today?" {
		t.Errorf("Content text = %v, expected %q", contentBlock["text"], "Hello! How can I help you today?")
	}
}

func TestHandleMessages_Streaming(t *testing.T) {
	// Create a test server that will act as OpenRouter with streaming
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("ResponseWriter doesn't support flushing")
		}

		// Send streaming chunks
		chunks := []string{
			`data: {"choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n\n"))
			flusher.Flush()
		}
	}))
	defer openRouterServer.Close()

	cfg := &config.Config{
		APIKey:      "test-key",
		BaseURL:     openRouterServer.URL,
		SonnetModel: "test/sonnet",
	}
	srv := New(cfg)

	reqBody := transform.AnthropicRequest{
		Model: "claude-3-5-sonnet",
		Messages: []transform.Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
		Stream: true,
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Status code = %d, expected %d. Body: %s", resp.StatusCode, http.StatusOK, body)
	}

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, expected %q", resp.Header.Get("Content-Type"), "text/event-stream")
	}

	// Read and verify streaming events
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should contain message_start event
	if !strings.Contains(bodyStr, "event: message_start") {
		t.Error("Response should contain message_start event")
	}

	// Should contain content_block_start event
	if !strings.Contains(bodyStr, "event: content_block_start") {
		t.Error("Response should contain content_block_start event")
	}

	// Should contain content_block_delta events
	if !strings.Contains(bodyStr, "event: content_block_delta") {
		t.Error("Response should contain content_block_delta events")
	}

	// Should contain message_stop event
	if !strings.Contains(bodyStr, "event: message_stop") {
		t.Error("Response should contain message_stop event")
	}
}

func TestHandleMessages_OpenRouterError(t *testing.T) {
	// Create a test server that returns an error
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Invalid API key"}`))
	}))
	defer openRouterServer.Close()

	cfg := &config.Config{
		APIKey:      "invalid-key",
		BaseURL:     openRouterServer.URL,
		SonnetModel: "test/sonnet",
	}
	srv := New(cfg)

	reqBody := transform.AnthropicRequest{
		Model: "claude-3-5-sonnet",
		Messages: []transform.Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleMessages_UserAgentForwarding(t *testing.T) {
	// Create a test server to verify User-Agent forwarding
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "TestClient/1.0" {
			t.Errorf("User-Agent = %q, expected %q", r.Header.Get("User-Agent"), "TestClient/1.0")
		}

		response := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": "OK",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer openRouterServer.Close()

	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: openRouterServer.URL,
		Model:   "test/model",
	}
	srv := New(cfg)

	reqBody := transform.AnthropicRequest{
		Model: "test-model",
		Messages: []transform.Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestClient/1.0")
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestHandleMessages_HeadersSet(t *testing.T) {
	// Create a test server to verify custom headers
	openRouterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HTTP-Referer") != "https://github.com/martinffx/athena" {
			t.Errorf("HTTP-Referer = %q, expected %q", r.Header.Get("HTTP-Referer"), "https://github.com/martinffx/athena")
		}

		if r.Header.Get("X-Title") != "Athena Proxy" {
			t.Errorf("X-Title = %q, expected %q", r.Header.Get("X-Title"), "Athena Proxy")
		}

		response := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": "OK",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer openRouterServer.Close()

	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: openRouterServer.URL,
		Model:   "test/model",
	}
	srv := New(cfg)

	reqBody := transform.AnthropicRequest{
		Model: "test-model",
		Messages: []transform.Message{
			{
				Role:    "user",
				Content: json.RawMessage(`"Hello"`),
			},
		},
	}

	reqJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.handleMessages(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status code = %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}
