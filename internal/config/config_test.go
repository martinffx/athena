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

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_BASE_URL")
	os.Unsetenv("DEFAULT_MODEL")
	os.Unsetenv("OPUS_MODEL")
	os.Unsetenv("SONNET_MODEL")
	os.Unsetenv("HAIKU_MODEL")

	cfg := Load("")

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
	if cfg.OpusModel != DefaultOpusModel {
		t.Errorf("Default opus model = %q, expected %q", cfg.OpusModel, DefaultOpusModel)
	}
	if cfg.SonnetModel != DefaultSonnetModel {
		t.Errorf("Default sonnet model = %q, expected %q", cfg.SonnetModel, DefaultSonnetModel)
	}
	if cfg.HaikuModel != DefaultHaikuModel {
		t.Errorf("Default haiku model = %q, expected %q", cfg.HaikuModel, DefaultHaikuModel)
	}
}

func TestLoad_EnvVars(t *testing.T) {
	// Set env vars
	os.Setenv("PORT", "9000")
	os.Setenv("OPENROUTER_API_KEY", "test-key-123")
	os.Setenv("OPENROUTER_BASE_URL", "https://custom.api.com")
	os.Setenv("DEFAULT_MODEL", "custom/model")
	os.Setenv("OPUS_MODEL", "custom/opus")
	os.Setenv("SONNET_MODEL", "custom/sonnet")
	os.Setenv("HAIKU_MODEL", "custom/haiku")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("OPENROUTER_BASE_URL")
		os.Unsetenv("DEFAULT_MODEL")
		os.Unsetenv("OPUS_MODEL")
		os.Unsetenv("SONNET_MODEL")
		os.Unsetenv("HAIKU_MODEL")
	}()

	cfg := Load("")

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

func TestLoad_YAMLFile(t *testing.T) {
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
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_BASE_URL")
	os.Unsetenv("DEFAULT_MODEL")
	os.Unsetenv("OPUS_MODEL")
	os.Unsetenv("SONNET_MODEL")
	os.Unsetenv("HAIKU_MODEL")

	cfg := Load(yamlPath)

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

func TestLoad_JSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "test.json")

	jsonContent := `{
  "port": "7070",
  "api_key": "json-key",
  "base_url": "https://json.api.com",
  "model": "json/model",
  "opus_model": "json/opus",
  "sonnet_model": "json/sonnet",
  "haiku_model": "json/haiku"
}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	// Clear env vars
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_BASE_URL")
	os.Unsetenv("DEFAULT_MODEL")
	os.Unsetenv("OPUS_MODEL")
	os.Unsetenv("SONNET_MODEL")
	os.Unsetenv("HAIKU_MODEL")

	cfg := Load(jsonPath)

	if cfg.Port != "7070" {
		t.Errorf("JSON port = %q, expected %q", cfg.Port, "7070")
	}
	if cfg.APIKey != "json-key" {
		t.Errorf("JSON API key = %q, expected %q", cfg.APIKey, "json-key")
	}
	if cfg.BaseURL != "https://json.api.com" {
		t.Errorf("JSON base URL = %q, expected %q", cfg.BaseURL, "https://json.api.com")
	}
}

func TestLoad_EnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	envContent := `PORT=6060
OPENROUTER_API_KEY="env-file-key"
OPENROUTER_BASE_URL=https://envfile.api.com
DEFAULT_MODEL=envfile/model
`

	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// Change to temp dir so .env is found
	origDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	// Clear env vars
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_BASE_URL")
	os.Unsetenv("DEFAULT_MODEL")

	cfg := Load("")

	if cfg.Port != "6060" {
		t.Errorf(".env port = %q, expected %q", cfg.Port, "6060")
	}
	if cfg.APIKey != "env-file-key" {
		t.Errorf(".env API key = %q, expected %q", cfg.APIKey, "env-file-key")
	}
	if cfg.BaseURL != "https://envfile.api.com" {
		t.Errorf(".env base URL = %q, expected %q", cfg.BaseURL, "https://envfile.api.com")
	}
	if cfg.Model != "envfile/model" {
		t.Errorf(".env model = %q, expected %q", cfg.Model, "envfile/model")
	}
}

func TestParseYAML_WithQuotes(t *testing.T) {
	yamlData := []byte(`port: "8080"
api_key: 'single-quoted'
model: unquoted`)

	cfg := &Config{}
	err := parseYAML(yamlData, cfg)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, expected %q", cfg.Port, "8080")
	}
	if cfg.APIKey != "single-quoted" {
		t.Errorf("APIKey = %q, expected %q", cfg.APIKey, "single-quoted")
	}
	if cfg.Model != "unquoted" {
		t.Errorf("Model = %q, expected %q", cfg.Model, "unquoted")
	}
}

func TestParseYAML_WithComments(t *testing.T) {
	yamlData := []byte(`# This is a comment
port: "9090"
# Another comment
api_key: "test-key"

model: "test/model"`)

	cfg := &Config{}
	err := parseYAML(yamlData, cfg)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, expected %q", cfg.Port, "9090")
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, expected %q", cfg.APIKey, "test-key")
	}
	if cfg.Model != "test/model" {
		t.Errorf("Model = %q, expected %q", cfg.Model, "test/model")
	}
}

func TestParseYAML_AllFields(t *testing.T) {
	yamlData := []byte(`port: "3000"
api_key: "all-fields-key"
base_url: "https://all.fields.com"
model: "all/default"
opus_model: "all/opus"
sonnet_model: "all/sonnet"
haiku_model: "all/haiku"`)

	cfg := &Config{}
	err := parseYAML(yamlData, cfg)
	if err != nil {
		t.Fatalf("parseYAML failed: %v", err)
	}

	expected := &Config{
		Port:        "3000",
		APIKey:      "all-fields-key",
		BaseURL:     "https://all.fields.com",
		Model:       "all/default",
		OpusModel:   "all/opus",
		SonnetModel: "all/sonnet",
		HaikuModel:  "all/haiku",
	}

	if cfg.Port != expected.Port {
		t.Errorf("Port = %q, expected %q", cfg.Port, expected.Port)
	}
	if cfg.APIKey != expected.APIKey {
		t.Errorf("APIKey = %q, expected %q", cfg.APIKey, expected.APIKey)
	}
	if cfg.BaseURL != expected.BaseURL {
		t.Errorf("BaseURL = %q, expected %q", cfg.BaseURL, expected.BaseURL)
	}
	if cfg.Model != expected.Model {
		t.Errorf("Model = %q, expected %q", cfg.Model, expected.Model)
	}
	if cfg.OpusModel != expected.OpusModel {
		t.Errorf("OpusModel = %q, expected %q", cfg.OpusModel, expected.OpusModel)
	}
	if cfg.SonnetModel != expected.SonnetModel {
		t.Errorf("SonnetModel = %q, expected %q", cfg.SonnetModel, expected.SonnetModel)
	}
	if cfg.HaikuModel != expected.HaikuModel {
		t.Errorf("HaikuModel = %q, expected %q", cfg.HaikuModel, expected.HaikuModel)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "env var exists",
			key:          "TEST_KEY_EXISTS",
			defaultValue: "default",
			envValue:     "exists",
			expected:     "exists",
		},
		{
			name:         "env var does not exist",
			key:          "TEST_KEY_MISSING",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvWithDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnvWithDefault(%q, %q) = %q, expected %q", tt.key, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestLoadEnvFile_WithQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env.test")

	envContent := `TEST_VAR1="double quoted"
TEST_VAR2='single quoted'
TEST_VAR3=unquoted
# Comment line
TEST_VAR4="value with spaces"
`

	err := os.WriteFile(envPath, []byte(envContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	loadEnvFile(envPath)

	tests := []struct {
		key      string
		expected string
	}{
		{"TEST_VAR1", "double quoted"},
		{"TEST_VAR2", "single quoted"},
		{"TEST_VAR3", "unquoted"},
		{"TEST_VAR4", "value with spaces"},
	}

	for _, tt := range tests {
		if val := os.Getenv(tt.key); val != tt.expected {
			t.Errorf("Env var %q = %q, expected %q", tt.key, val, tt.expected)
		}
		os.Unsetenv(tt.key)
	}
}

func TestLoadEnvFile_NonExistent(_ *testing.T) {
	// Should not panic or error when file doesn't exist
	loadEnvFile("/nonexistent/path/to/.env")
}

func TestLoad_PriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	yamlPath := filepath.Join(tmpDir, "config.yml")
	yamlContent := `port: "8080"
api_key: "file-key"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Set env vars
	// Port: set to default so file can override
	// APIKey: set to non-empty so file cannot override
	os.Setenv("PORT", "11434")
	os.Setenv("OPENROUTER_API_KEY", "env-key")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("OPENROUTER_API_KEY")
	}()

	cfg := Load(yamlPath)

	// Port: env == default, so file overrides
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, expected file value %q (file overrides when env == default)", cfg.Port, "8080")
	}
	// APIKey: env is set, so file doesn't override (file only overrides if cfg.APIKey == "")
	if cfg.APIKey != "env-key" {
		t.Errorf("APIKey = %q, expected env value %q (env set, so file doesn't override)", cfg.APIKey, "env-key")
	}
}

func TestLoad_FileOverridesDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config file
	yamlPath := filepath.Join(tmpDir, "config.yml")
	yamlContent := `port: "8080"
api_key: "file-key"
base_url: "https://file.api.com"
`
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Clear all env vars so we test file overriding defaults
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("OPENROUTER_BASE_URL")

	cfg := Load(yamlPath)

	// File should override all defaults
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

func TestLoadConfigFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(jsonPath, []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	os.Unsetenv("PORT")
	cfg := Load(jsonPath)

	// Should fall back to defaults
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, expected default %q after invalid JSON", cfg.Port, DefaultPort)
	}
}

func TestLoadConfigFromFile_NonExistent(t *testing.T) {
	os.Unsetenv("PORT")
	cfg := Load("/nonexistent/config.yml")

	// Should use defaults
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, expected default %q with nonexistent file", cfg.Port, DefaultPort)
	}
}

func TestLoad_AutoGroqForKimiK2(t *testing.T) {
	// Clear env vars
	os.Unsetenv("DEFAULT_MODEL")

	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "kimi.yml")

	yamlContent := `model: "moonshotai/kimi-k2-0905"
api_key: "test-key"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	cfg := Load(yamlPath)

	// Should auto-configure Groq provider for kimi-k2 models
	if cfg.DefaultProvider == nil {
		t.Error("Expected DefaultProvider to be auto-configured for kimi-k2 model, got nil")
	} else {
		if len(cfg.DefaultProvider.Order) != 1 || cfg.DefaultProvider.Order[0] != "Groq" {
			t.Errorf("Expected DefaultProvider.Order = [Groq], got %v", cfg.DefaultProvider.Order)
		}
		if cfg.DefaultProvider.AllowFallbacks != false {
			t.Errorf("Expected DefaultProvider.AllowFallbacks = false, got %v", cfg.DefaultProvider.AllowFallbacks)
		}
	}
}

func TestLoad_AutoGroqForFileModelNotEnvModel(t *testing.T) {
	// This test verifies the timing bug fix:
	// Auto-config should check the FINAL model (after file merge), not the initial env/default model
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yml")

	// File specifies kimi-k2 model
	yamlContent := `model: "moonshotai/kimi-k2-0905"
api_key: "test-key"
`

	err := os.WriteFile(yamlPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test YAML file: %v", err)
	}

	// Set env to a NON-kimi model
	os.Setenv("DEFAULT_MODEL", "google/gemini-2.0-flash-exp:free")
	defer os.Unsetenv("DEFAULT_MODEL")

	cfg := Load(yamlPath)

	// Should auto-configure Groq because FILE has kimi-k2 (not env)
	if cfg.Model != "moonshotai/kimi-k2-0905" {
		t.Errorf("Expected model from file = %q, got %q", "moonshotai/kimi-k2-0905", cfg.Model)
	}

	if cfg.DefaultProvider == nil {
		t.Fatal("Expected DefaultProvider to be auto-configured for kimi-k2 model from FILE, got nil")
	}

	if len(cfg.DefaultProvider.Order) != 1 || cfg.DefaultProvider.Order[0] != testProviderGroq {
		t.Errorf("Expected DefaultProvider.Order = [%s], got %v", testProviderGroq, cfg.DefaultProvider.Order)
	}
}

func TestLoad_NoAutoGroqWhenProviderConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "config.json")

	jsonContent := `{
  "model": "moonshotai/kimi-k2-0905",
  "api_key": "test-key",
  "default_provider": {
    "order": ["` + testProviderAnthropic + `"],
    "allow_fallbacks": true
  }
}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	// Clear env vars
	os.Unsetenv("DEFAULT_MODEL")

	cfg := Load(jsonPath)

	// Should NOT auto-configure Groq since provider is explicitly set
	if cfg.DefaultProvider == nil {
		t.Error("Expected DefaultProvider to be loaded from config, got nil")
	} else {
		if len(cfg.DefaultProvider.Order) != 1 || cfg.DefaultProvider.Order[0] != testProviderAnthropic {
			t.Errorf("Expected DefaultProvider.Order = [%s], got %v", testProviderAnthropic, cfg.DefaultProvider.Order)
		}
		if cfg.DefaultProvider.AllowFallbacks != true {
			t.Errorf("Expected DefaultProvider.AllowFallbacks = true, got %v", cfg.DefaultProvider.AllowFallbacks)
		}
	}
}

func TestLoad_JSONWithProviderConfigs(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "providers.json")

	jsonContent := `{
  "port": "8080",
  "api_key": "test-key",
  "model": "anthropic/claude-3.5-sonnet",
  "default_provider": {
    "order": ["` + testProviderAnthropic + `", "` + testProviderOpenAI + `"],
    "allow_fallbacks": true
  },
  "opus_provider": {
    "order": ["` + testProviderAnthropic + `"],
    "allow_fallbacks": false
  },
  "sonnet_provider": {
    "order": ["` + testProviderGroq + `"],
    "allow_fallbacks": false
  }
}`

	err := os.WriteFile(jsonPath, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	// Clear env vars
	os.Unsetenv("PORT")
	os.Unsetenv("OPENROUTER_API_KEY")
	os.Unsetenv("DEFAULT_MODEL")

	cfg := Load(jsonPath)

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
