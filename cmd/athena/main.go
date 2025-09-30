package main

import (
	"flag"
	"log"

	"athena/internal/config"
	"athena/internal/server"
)

func main() {
	// Parse command line flags
	var configFile string
	var port string
	var apiKey string
	var baseURL string
	var model string
	var opusModel string
	var sonnetModel string
	var haikuModel string

	flag.StringVar(&configFile, "config", "", "Path to config file (JSON/YAML)")
	flag.StringVar(&port, "port", "", "Port to run the server on")
	flag.StringVar(&apiKey, "api-key", "", "OpenRouter API key")
	flag.StringVar(&baseURL, "base-url", "", "OpenRouter base URL")
	flag.StringVar(&model, "model", "", "Default model to use")
	flag.StringVar(&opusModel, "opus-model", "", "Model to map claude-opus requests to")
	flag.StringVar(&sonnetModel, "sonnet-model", "", "Model to map claude-sonnet requests to")
	flag.StringVar(&haikuModel, "haiku-model", "", "Model to map claude-haiku requests to")
	flag.Parse()

	// Load configuration
	cfg := config.Load(configFile)

	// Override config with command line flags if provided
	if port != "" {
		cfg.Port = port
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}
	if model != "" {
		cfg.Model = model
	}
	if opusModel != "" {
		cfg.OpusModel = opusModel
	}
	if sonnetModel != "" {
		cfg.SonnetModel = sonnetModel
	}
	if haikuModel != "" {
		cfg.HaikuModel = haikuModel
	}

	// Validate required config
	if cfg.APIKey == "" {
		log.Fatal("OpenRouter API key is required. Use -api-key flag, config file, or OPENROUTER_API_KEY env var")
	}

	// Create and start server
	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
