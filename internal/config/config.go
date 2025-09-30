package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Default configuration values
const (
	DefaultModelName   = "moonshotai/kimi-k2-0905"
	DefaultPort        = "11434"
	DefaultBaseURL     = "https://openrouter.ai/api"
	DefaultOpusModel   = "anthropic/claude-3-opus"
	DefaultSonnetModel = "anthropic/claude-3.5-sonnet"
	DefaultHaikuModel  = "anthropic/claude-3.5-haiku"
)

// ProviderConfig holds provider routing configuration
type ProviderConfig struct {
	Order          []string `json:"order"`
	AllowFallbacks bool     `json:"allow_fallbacks"`
}

// Config holds the application configuration
type Config struct {
	Port            string          `json:"port"`
	APIKey          string          `json:"api_key"`
	BaseURL         string          `json:"base_url"`
	Model           string          `json:"model"`
	OpusModel       string          `json:"opus_model"`
	SonnetModel     string          `json:"sonnet_model"`
	HaikuModel      string          `json:"haiku_model"`
	DefaultProvider *ProviderConfig `json:"default_provider,omitempty"`
	OpusProvider    *ProviderConfig `json:"opus_provider,omitempty"`
	SonnetProvider  *ProviderConfig `json:"sonnet_provider,omitempty"`
	HaikuProvider   *ProviderConfig `json:"haiku_provider,omitempty"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) *Config {
	cfg := &Config{}

	// Load .env file if it exists
	loadEnvFile(".env")

	// Set defaults from environment variables
	cfg.Port = getEnvWithDefault("PORT", DefaultPort)
	cfg.APIKey = getEnvWithDefault("OPENROUTER_API_KEY", "")
	cfg.BaseURL = getEnvWithDefault("OPENROUTER_BASE_URL", DefaultBaseURL)
	cfg.Model = getEnvWithDefault("DEFAULT_MODEL", DefaultModelName)
	cfg.OpusModel = getEnvWithDefault("OPUS_MODEL", DefaultOpusModel)
	cfg.SonnetModel = getEnvWithDefault("SONNET_MODEL", DefaultSonnetModel)
	cfg.HaikuModel = getEnvWithDefault("HAIKU_MODEL", DefaultHaikuModel)

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

	// Set default provider for kimi-k2 models if not already configured
	// This runs AFTER file loading so it checks the final model value
	if cfg.DefaultProvider == nil && strings.Contains(cfg.Model, "kimi-k2") {
		log.Printf("Auto-configuring Groq provider for kimi-k2 model")
		cfg.DefaultProvider = &ProviderConfig{
			Order:          []string{"Groq"},
			AllowFallbacks: false,
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
		// YAML format only supports basic config fields
		// Provider routing requires JSON format
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
	mergeStringField(&cfg.Port, fileConfig.Port, "PORT", DefaultPort)
	mergeStringField(&cfg.BaseURL, fileConfig.BaseURL, "OPENROUTER_BASE_URL", DefaultBaseURL)
	mergeStringField(&cfg.Model, fileConfig.Model, "DEFAULT_MODEL", DefaultModelName)
	mergeStringField(&cfg.OpusModel, fileConfig.OpusModel, "OPUS_MODEL", DefaultOpusModel)
	mergeStringField(&cfg.SonnetModel, fileConfig.SonnetModel, "SONNET_MODEL", DefaultSonnetModel)
	mergeStringField(&cfg.HaikuModel, fileConfig.HaikuModel, "HAIKU_MODEL", DefaultHaikuModel)

	// APIKey special case: only merge if current is empty
	if fileConfig.APIKey != "" && cfg.APIKey == "" {
		cfg.APIKey = fileConfig.APIKey
	}
	// Override provider configs if present in file
	if fileConfig.DefaultProvider != nil {
		cfg.DefaultProvider = fileConfig.DefaultProvider
	}
	if fileConfig.OpusProvider != nil {
		cfg.OpusProvider = fileConfig.OpusProvider
	}
	if fileConfig.SonnetProvider != nil {
		cfg.SonnetProvider = fileConfig.SonnetProvider
	}
	if fileConfig.HaikuProvider != nil {
		cfg.HaikuProvider = fileConfig.HaikuProvider
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// mergeStringField merges a config field from file into current config
// Only overwrites if file value is non-empty and current value equals env/default
func mergeStringField(current *string, fileValue, envKey, defaultValue string) {
	envOrDefault := getEnvWithDefault(envKey, defaultValue)
	if fileValue != "" && *current == envOrDefault {
		*current = fileValue
	}
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
