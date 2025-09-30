package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
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

	log.Printf("Starting server on port %s", s.cfg.Port)

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
		log.Printf("Failed to encode health response: %v", err)
	}
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
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

	// Transform to OpenAI format
	openAIReq := transform.AnthropicToOpenAI(req, s.cfg)
	openAIBody, err := json.Marshal(openAIReq)
	if err != nil {
		http.Error(w, "Failed to marshal OpenAI request", http.StatusInternalServerError)
		return
	}

	// Forward to OpenRouter
	client := &http.Client{}
	url := s.cfg.BaseURL + "/v1/chat/completions"
	openRouterReq, err := http.NewRequest("POST", url, bytes.NewReader(openAIBody))
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

	// Handle response based on streaming
	if req.Stream {
		transform.HandleStreaming(w, resp, openAIReq.Model)
	} else {
		transform.HandleNonStreaming(w, resp, openAIReq.Model)
	}
}
