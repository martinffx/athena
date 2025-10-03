package config

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testProviderAnthropic = "Anthropic"
	testProviderGroq      = "Groq"
	testProviderOpenAI    = "OpenAI"
)

func TestNew_Defaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("ATHENA_PORT")
	os.Unsetenv("ATHENA_API_KEY")
	os.Unsetenv("ATHENA_BASE_URL")
	os.Unsetenv("ATHENA_MODEL")
	os.Unsetenv("ATHENA_OPUS_MODEL")
	os.Unsetenv("ATHENA_SONNET_MODEL")
	os.Unsetenv("ATHENA_HAIKU_MODEL")

	cfg, err := New("")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if cfg.Port != DefaultPort {
		t.Errorf("Default port = %q, expected %q", cfg.Port, DefaultPort)
	}
	if cfg.APIKey != "" {
		t.Errorf("Default API key should be empty, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != DefaultBaseURL {
		t.Errorf("Default base URL = %q, expected %q", cfg.BaseURL, DefaultBaseURL)
	}
	if cfg.Model != DefaultModelName {
		t.Errorf("Default model = %q, expected %q", cfg.Model, DefaultModelName)
	}
	if cfg.OpusModel != "" {
		t.Errorf("Default opus model should be empty, got %q", cfg.OpusModel)
	}
	if cfg.SonnetModel != "" {
		t.Errorf("Default sonnet model should be empty, got %q", cfg.SonnetModel)
	}
	if cfg.HaikuModel != "" {
		t.Errorf("Default haiku model should be empty, got %q", cfg.HaikuModel)
	}
}

func TestNew_EnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("ATHENA_PORT", "9000")
	os.Setenv("ATHENA_API_KEY", "test-key-123")
	os.Setenv("ATHENA_BASE_URL", "https://custom.api.com")
	os.Setenv("ATHENA_MODEL", "custom/model")
	os.Setenv("ATHENA_OPUS_MODEL", "custom/opus")
	os.Setenv("ATHENA_SONNET_MODEL", "custom/sonnet")
	os.Setenv("ATHENA_HAIKU_MODEL", "custom/haiku")

	defer func() {
		os.Unsetenv("ATHENA_PORT")
		os.Unsetenv("ATHENA_API_KEY")
		os.Unsetenv("ATHENA_BASE_URL")
		os.Unsetenv("ATHENA_MODEL")
		os.Unsetenv("ATHENA_OPUS_MODEL")
		os.Unsetenv("ATHENA_SONNET_MODEL")
		os.Unsetenv("ATHENA_HAIKU_MODEL")
	}()

	cfg, err := New("")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Env port = %q, expected %q", cfg.Port, "9000")
	}
	if cfg.APIKey != "test-key-123" {
		t.Errorf("Env API key = %q, expected %q", cfg.APIKey, "test-key-123")
	}
	if cfg.BaseURL != "https://custom.api.com" {
		t.Errorf("Env base URL = %q, expected %q", cfg.BaseURL, "https://custom.api.com")
	}
	if cfg.Model != "custom/model" {
		t.Errorf("Env model = %q, expected %q", cfg.Model, "custom/model")
	}
	if cfg.OpusModel != "custom/opus" {
		t.Errorf("Env opus model = %q, expected %q", cfg.OpusModel, "custom/opus")
	}
	if cfg.SonnetModel != "custom/sonnet" {
		t.Errorf("Env sonnet model = %q, expected %q", cfg.SonnetModel, "custom/sonnet")
	}
	if cfg.HaikuModel != "custom/haiku" {
		t.Errorf("Env haiku model = %q, expected %q", cfg.HaikuModel, "custom/haiku")
	}
}

func TestNew_YAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test.yml")

	yamlContent := `port: "8080"
api_key: "yaml-key"
base_url: "https://yaml.api.com"
model: "yaml/model"
opus_model: "yaml/opus"
sonnet_model: "yaml/sonnet"
haiku_model: "yaml/haiku"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	// Clear env vars to test file-only loading
	os.Unsetenv("ATHENA_PORT")
	os.Unsetenv("ATHENA_API_KEY")
	os.Unsetenv("ATHENA_BASE_URL")
	os.Unsetenv("ATHENA_MODEL")
	os.Unsetenv("ATHENA_OPUS_MODEL")
	os.Unsetenv("ATHENA_SONNET_MODEL")
	os.Unsetenv("ATHENA_HAIKU_MODEL")

	cfg, err := New(yamlPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("YAML port = %q, expected %q", cfg.Port, "8080")
	}
	if cfg.APIKey != "yaml-key" {
		t.Errorf("YAML API key = %q, expected %q", cfg.APIKey, "yaml-key")
	}
	if cfg.BaseURL != "https://yaml.api.com" {
		t.Errorf("YAML base URL = %q, expected %q", cfg.BaseURL, "https://yaml.api.com")
	}
	if cfg.Model != "yaml/model" {
		t.Errorf("YAML model = %q, expected %q", cfg.Model, "yaml/model")
	}
	if cfg.OpusModel != "yaml/opus" {
		t.Errorf("YAML opus model = %q, expected %q", cfg.OpusModel, "yaml/opus")
	}
	if cfg.SonnetModel != "yaml/sonnet" {
		t.Errorf("YAML sonnet model = %q, expected %q", cfg.SonnetModel, "yaml/sonnet")
	}
	if cfg.HaikuModel != "yaml/haiku" {
		t.Errorf("YAML haiku model = %q, expected %q", cfg.HaikuModel, "yaml/haiku")
	}
}

func TestNew_EnvOverridesFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yml")

	yamlContent := `port: "8080"
api_key: "file-key"
model: "file/model"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Set env vars (should override file)
	os.Setenv("ATHENA_PORT", "9999")
	os.Setenv("ATHENA_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("ATHENA_PORT")
		os.Unsetenv("ATHENA_API_KEY")
	}()

	cfg, err := New(yamlPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Env should override file
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, expected env value %q (env overrides file)", cfg.Port, "9999")
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, expected env value %q (env overrides file)", cfg.APIKey, "env-key")
	}
	// Model not set in env, should use file
	if cfg.Model != "file/model" {
		t.Errorf("Model = %q, expected file value %q", cfg.Model, "file/model")
	}
}

func TestNew_FileOverridesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yml")

	yamlContent := `port: "8080"
api_key: "file-key"
base_url: "https://file.api.com"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Clear all env vars
	os.Unsetenv("ATHENA_PORT")
	os.Unsetenv("ATHENA_API_KEY")
	os.Unsetenv("ATHENA_BASE_URL")

	cfg, err := New(yamlPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// File should override defaults
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, expected file value %q", cfg.Port, "8080")
	}
	if cfg.APIKey != "file-key" {
		t.Errorf("APIKey = %q, expected file value %q", cfg.APIKey, "file-key")
	}
	if cfg.BaseURL != "https://file.api.com" {
		t.Errorf("BaseURL = %q, expected file value %q", cfg.BaseURL, "https://file.api.com")
	}
}

func TestNew_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "invalid.yml")

	err := os.WriteFile(yamlPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML file: %v", err)
	}

	_, err = New(yamlPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestNew_NonExistentFile(t *testing.T) {
	_, err := New("/nonexistent/config.yml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestNew_YAMLWithProviderConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "providers.yml")

	yamlContent := `port: "8080"
api_key: "test-key"
model: "anthropic/claude-3.5-sonnet"
default_provider:
  order:
    - Anthropic
    - OpenAI
  allow_fallbacks: true
opus_provider:
  order:
    - Anthropic
  allow_fallbacks: false
sonnet_provider:
  order:
    - Groq
  allow_fallbacks: false
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	// Clear env vars
	os.Unsetenv("ATHENA_PORT")
	os.Unsetenv("ATHENA_API_KEY")
	os.Unsetenv("ATHENA_MODEL")

	cfg, err := New(yamlPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Check basic config
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, expected %q", cfg.Port, "8080")
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, expected %q", cfg.APIKey, "test-key")
	}

	// Check default provider
	if cfg.DefaultProvider == nil {
		t.Fatal("Expected DefaultProvider to be loaded, got nil")
	}
	if len(cfg.DefaultProvider.Order) != 2 {
		t.Errorf("Expected DefaultProvider.Order length = 2, got %d", len(cfg.DefaultProvider.Order))
	}
	if cfg.DefaultProvider.Order[0] != testProviderAnthropic || cfg.DefaultProvider.Order[1] != testProviderOpenAI {
		t.Errorf("Expected DefaultProvider.Order = [%s, %s], got %v", testProviderAnthropic, testProviderOpenAI, cfg.DefaultProvider.Order)
	}
	if !cfg.DefaultProvider.AllowFallbacks {
		t.Error("Expected DefaultProvider.AllowFallbacks = true")
	}

	// Check opus provider
	if cfg.OpusProvider == nil {
		t.Fatal("Expected OpusProvider to be loaded, got nil")
	}
	if len(cfg.OpusProvider.Order) != 1 || cfg.OpusProvider.Order[0] != testProviderAnthropic {
		t.Errorf("Expected OpusProvider.Order = [%s], got %v", testProviderAnthropic, cfg.OpusProvider.Order)
	}
	if cfg.OpusProvider.AllowFallbacks {
		t.Error("Expected OpusProvider.AllowFallbacks = false")
	}

	// Check sonnet provider
	if cfg.SonnetProvider == nil {
		t.Fatal("Expected SonnetProvider to be loaded, got nil")
	}
	if len(cfg.SonnetProvider.Order) != 1 || cfg.SonnetProvider.Order[0] != testProviderGroq {
		t.Errorf("Expected SonnetProvider.Order = [%s], got %v", testProviderGroq, cfg.SonnetProvider.Order)
	}
	if cfg.SonnetProvider.AllowFallbacks {
		t.Error("Expected SonnetProvider.AllowFallbacks = false")
	}

	// HaikuProvider should be nil (not configured)
	if cfg.HaikuProvider != nil {
		t.Errorf("Expected HaikuProvider = nil, got %+v", cfg.HaikuProvider)
	}
}

func TestNew_LogFormat(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yml")

	yamlContent := `log_format: "json"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	os.Unsetenv("ATHENA_LOG_FORMAT")

	cfg, err := New(yamlPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, expected %q", cfg.LogFormat, "json")
	}
}
