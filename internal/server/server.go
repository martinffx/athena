package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"athena/internal/config"
	"athena/internal/transform"
)

// Server represents the HTTP server
type Server struct {
	cfg *config.Config
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

// Start starts the HTTP server
func (s *Server) Start() error {

	http.HandleFunc("/v1/messages", s.handleMessages)
	http.HandleFunc("/health", s.handleHealth)

	slog.Info("starting server", "port", s.cfg.Port)

	// Create server with proper timeouts for security
	server := &http.Server{
		Addr:           ":" + s.cfg.Port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return server.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("failed to encode health response", "error", err)
	}
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req transform.AnthropicRequest
	if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	slog.Info("request received",
		"method", "POST",
		"path", "/v1/messages",
		"model", req.Model,
		"stream", req.Stream,
	)

	// Transform to OpenAI format
	openAIReq := transform.AnthropicToOpenAI(req, s.cfg)

	// Log provider routing if configured
	providerInfo := "default"
	if openAIReq.Provider != nil && len(openAIReq.Provider.Order) > 0 {
		providerInfo = strings.Join(openAIReq.Provider.Order, ",")
	}

	slog.Info("routing request",
		"from_model", req.Model,
		"to_model", openAIReq.Model,
		"provider", providerInfo,
	)

	openAIBody, err := json.Marshal(openAIReq)
	if err != nil {
		http.Error(w, "Failed to marshal OpenAI request", http.StatusInternalServerError)
		return
	}

	// Forward to OpenRouter
	client := &http.Client{}
	url := s.cfg.BaseURL + "/v1/chat/completions"
	openRouterReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(openAIBody))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	openRouterReq.Header.Set("Content-Type", "application/json")
	openRouterReq.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	openRouterReq.Header.Set("HTTP-Referer", "https://github.com/martinffx/athena")
	openRouterReq.Header.Set("X-Title", "Athena Proxy")

	if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
		openRouterReq.Header.Set("User-Agent", userAgent)
	}

	resp, err := client.Do(openRouterReq)
	if err != nil {
		http.Error(w, "Failed to connect to OpenRouter", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Extract actual provider from OpenRouter response headers
	actualProvider := resp.Header.Get("X-OpenRouter-Provider")
	if actualProvider == "" {
		actualProvider = "unknown"
	}

	slog.Info("response received",
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
		"actual_provider", actualProvider,
	)

	// Handle response based on streaming
	if req.Stream {
		transform.HandleStreaming(w, resp, openAIReq.Model)
	} else {
		transform.HandleNonStreaming(w, resp, openAIReq.Model)
	}
}
