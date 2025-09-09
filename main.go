package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "Path to config file (JSON)")
	flag.StringVar(&config.Port, "port", "", "Port to run the server on")
	flag.StringVar(&config.APIKey, "api-key", "", "OpenRouter API key")
	flag.StringVar(&config.BaseURL, "base-url", "", "OpenRouter base URL")
	flag.StringVar(&config.Model, "model", "", "Default model to use")
	flag.StringVar(&config.OpusModel, "opus-model", "", "Model to map claude-opus requests to")
	flag.StringVar(&config.SonnetModel, "sonnet-model", "", "Model to map claude-sonnet requests to")
	flag.StringVar(&config.HaikuModel, "haiku-model", "", "Model to map claude-haiku requests to")
	flag.Parse()

	loadConfig(configFile)

	if config.APIKey == "" {
		log.Fatal("OpenRouter API key is required. Use -api-key flag, config file, or OPENROUTER_API_KEY env var")
	}

	http.HandleFunc("/v1/messages", handleMessages)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Printf("Starting server on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
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

	var req AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Transform to OpenAI format
	openAIReq := transformAnthropicToOpenAI(req)
	openAIBody, err := json.Marshal(openAIReq)
	if err != nil {
		http.Error(w, "Failed to marshal OpenAI request", http.StatusInternalServerError)
		return
	}

	// Forward to OpenRouter
	client := &http.Client{}
	url := config.BaseURL + "/v1/chat/completions"
	openRouterReq, err := http.NewRequest("POST", url, bytes.NewReader(openAIBody))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	openRouterReq.Header.Set("Content-Type", "application/json")
	openRouterReq.Header.Set("Authorization", "Bearer "+config.APIKey)
	openRouterReq.Header.Set("HTTP-Referer", "https://github.com/martinrichards23/openrouter-cc")
	openRouterReq.Header.Set("X-Title", "OpenRouter Claude Code Proxy")

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
		handleStreamingResponse(w, resp, openAIReq.Model)
	} else {
		handleNonStreamingResponse(w, resp, openAIReq.Model)
	}
}
