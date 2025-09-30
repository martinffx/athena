package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the application configuration
type Config struct {
	Port        string `json:"port"`
	APIKey      string `json:"api_key"`
	BaseURL     string `json:"base_url"`
	Model       string `json:"model"`
	OpusModel   string `json:"opus_model"`
	SonnetModel string `json:"sonnet_model"`
	HaikuModel  string `json:"haiku_model"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) *Config {
	cfg := &Config{}

	// Load .env file if it exists
	loadEnvFile(".env")

	// Set defaults from environment variables
	cfg.Port = getEnvWithDefault("PORT", "11434")
	cfg.APIKey = getEnvWithDefault("OPENROUTER_API_KEY", "")
	cfg.BaseURL = getEnvWithDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api")
	cfg.Model = getEnvWithDefault("DEFAULT_MODEL", "google/gemini-2.0-flash-exp:free")
	cfg.OpusModel = getEnvWithDefault("OPUS_MODEL", "anthropic/claude-3-opus")
	cfg.SonnetModel = getEnvWithDefault("SONNET_MODEL", "anthropic/claude-3.5-sonnet")
	cfg.HaikuModel = getEnvWithDefault("HAIKU_MODEL", "anthropic/claude-3.5-haiku")

	// Load config file if specified
	if configFile != "" {
		loadConfigFromFile(configFile, cfg)
	} else {
		// Try to load from standard locations
		configPaths := []string{
			filepath.Join(os.Getenv("HOME"), ".config", "athena", "athena.yml"),
			filepath.Join(os.Getenv("HOME"), ".config", "athena", "athena.json"),
			"./athena.yml",
			"./athena.json",
		}

		for _, path := range configPaths {
			if _, err := os.Stat(path); err == nil {
				loadConfigFromFile(path, cfg)
				break
			}
		}
	}

	return cfg
}

func loadConfigFromFile(filename string, cfg *Config) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v", filename, err)
		return
	}

	var fileConfig Config

	if strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml") {
		if err := parseYAML(data, &fileConfig); err != nil {
			log.Printf("Warning: Could not parse YAML config file %s: %v", filename, err)
			return
		}
	} else {
		if err := json.Unmarshal(data, &fileConfig); err != nil {
			log.Printf("Warning: Could not parse JSON config file %s: %v", filename, err)
			return
		}
	}

	log.Printf("Loaded config from %s", filename)

	// Override with config file values only if they're not empty
	defaultPort := getEnvWithDefault("PORT", "11434")
	if fileConfig.Port != "" && cfg.Port == defaultPort {
		cfg.Port = fileConfig.Port
	}
	if fileConfig.APIKey != "" && cfg.APIKey == "" {
		cfg.APIKey = fileConfig.APIKey
	}
	defaultBaseURL := getEnvWithDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api")
	if fileConfig.BaseURL != "" && cfg.BaseURL == defaultBaseURL {
		cfg.BaseURL = fileConfig.BaseURL
	}
	defaultModel := getEnvWithDefault("DEFAULT_MODEL", "google/gemini-2.0-flash-exp:free")
	if fileConfig.Model != "" && cfg.Model == defaultModel {
		cfg.Model = fileConfig.Model
	}
	defaultOpus := getEnvWithDefault("OPUS_MODEL", "anthropic/claude-3-opus")
	if fileConfig.OpusModel != "" && cfg.OpusModel == defaultOpus {
		cfg.OpusModel = fileConfig.OpusModel
	}
	defaultSonnet := getEnvWithDefault("SONNET_MODEL", "anthropic/claude-3.5-sonnet")
	if fileConfig.SonnetModel != "" && cfg.SonnetModel == defaultSonnet {
		cfg.SonnetModel = fileConfig.SonnetModel
	}
	defaultHaiku := getEnvWithDefault("HAIKU_MODEL", "anthropic/claude-3.5-haiku")
	if fileConfig.HaikuModel != "" && cfg.HaikuModel == defaultHaiku {
		cfg.HaikuModel = fileConfig.HaikuModel
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseYAML(data []byte, config *Config) error {
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		switch key {
		case "port":
			config.Port = value
		case "api_key":
			config.APIKey = value
		case "base_url":
			config.BaseURL = value
		case "model":
			config.Model = value
		case "opus_model":
			config.OpusModel = value
		case "sonnet_model":
			config.SonnetModel = value
		case "haiku_model":
			config.HaikuModel = value
		}
	}

	return nil
}

func loadEnvFile(filename string) {
	if _, err := os.Stat(filename); err != nil {
		return // File doesn't exist, skip silently
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		os.Setenv(key, value)
	}

	log.Printf("Loaded environment variables from %s", filename)
}
