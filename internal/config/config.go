package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

// Default configuration values
const (
	DefaultModelName = "moonshotai/kimi-k2-0905"
	DefaultPort      = "12377"
	DefaultBaseURL   = "https://openrouter.ai/api"
)

// ProviderConfig holds provider routing configuration
type ProviderConfig struct {
	Order          []string `yaml:"order" json:"order"`
	AllowFallbacks bool     `yaml:"allow_fallbacks" json:"allow_fallbacks"`
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
	LogLevel        string          `yaml:"log_level,omitempty"`
	LogFile         string          `yaml:"log_file,omitempty"`
}

// New creates a new Config with precedence: env vars > ./athena.yml > ~/.config/athena/athena.yml > defaults
func New(configPath string) (*Config, error) {
	// 1. Start with hard-coded defaults
	cfg := &Config{
		Port:      DefaultPort,
		BaseURL:   DefaultBaseURL,
		Model:     DefaultModelName,
		LogFormat: "text",
		LogLevel:  "info",
	}

	// 2. Discover and load config files (if not explicitly provided)
	var paths []string
	if configPath != "" {
		// Explicit config path takes precedence over discovery
		paths = []string{configPath}
	} else {
		// Discover config files in priority order: global, then local
		paths = discoverConfigFiles()
	}

	// Load config files in reverse priority order (global first, local last)
	// This ensures local configs override global configs
	for _, path := range paths {
		if err := loadConfigFile(path, cfg); err != nil {
			if configPath != "" {
				// If explicit config was specified, fail on error
				return nil, err
			}
			// For discovered configs, skip if file doesn't exist or has errors
			continue
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
	if v := os.Getenv("ATHENA_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("ATHENA_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}

	return cfg, nil
}

// discoverConfigFiles returns a list of config file paths in priority order
// Priority: ~/.config/athena/athena.yml (global) → ./athena.yml (local)
func discoverConfigFiles() []string {
	var paths []string

	// 1. Global config in ~/.config/athena/athena.yml
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := home + "/.config/athena/athena.yml"
		if _, err := os.Stat(globalPath); err == nil {
			paths = append(paths, globalPath)
		}
	}

	// 2. Local config in ./athena.yml
	localPath := "./athena.yml"
	if _, err := os.Stat(localPath); err == nil {
		paths = append(paths, localPath)
	}

	return paths
}

// loadConfigFile loads and merges a YAML config file into the provided config
func loadConfigFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}
	return nil
}
