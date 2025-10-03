package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

// Default configuration values
const (
	DefaultModelName = "moonshotai/kimi-k2-0905"
	DefaultPort      = "11434"
	DefaultBaseURL   = "https://openrouter.ai/api"
)

// ProviderConfig holds provider routing configuration
type ProviderConfig struct {
	Order          []string `yaml:"order"`
	AllowFallbacks bool     `yaml:"allow_fallbacks"`
}

// Config holds the application configuration
type Config struct {
	Port            string          `yaml:"port"`
	APIKey          string          `yaml:"api_key"`
	BaseURL         string          `yaml:"base_url"`
	Model           string          `yaml:"model"`
	OpusModel       string          `yaml:"opus_model,omitempty"`
	SonnetModel     string          `yaml:"sonnet_model,omitempty"`
	HaikuModel      string          `yaml:"haiku_model,omitempty"`
	DefaultProvider *ProviderConfig `yaml:"default_provider,omitempty"`
	OpusProvider    *ProviderConfig `yaml:"opus_provider,omitempty"`
	SonnetProvider  *ProviderConfig `yaml:"sonnet_provider,omitempty"`
	HaikuProvider   *ProviderConfig `yaml:"haiku_provider,omitempty"`
	LogFormat       string          `yaml:"log_format"`
	LogFile         string          `yaml:"log_file,omitempty"`
}

// New creates a new Config with precedence: env vars > config file > defaults
func New(configPath string) (*Config, error) {
	// 1. Start with hard-coded defaults
	cfg := &Config{
		Port:      DefaultPort,
		BaseURL:   DefaultBaseURL,
		Model:     DefaultModelName,
		LogFormat: "text",
	}

	// 2. Load and merge YAML config file (if provided)
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// 3. Override with env vars (highest priority)
	if v := os.Getenv("ATHENA_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("ATHENA_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("ATHENA_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("ATHENA_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("ATHENA_OPUS_MODEL"); v != "" {
		cfg.OpusModel = v
	}
	if v := os.Getenv("ATHENA_SONNET_MODEL"); v != "" {
		cfg.SonnetModel = v
	}
	if v := os.Getenv("ATHENA_HAIKU_MODEL"); v != "" {
		cfg.HaikuModel = v
	}
	if v := os.Getenv("ATHENA_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("ATHENA_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}

	return cfg, nil
}
