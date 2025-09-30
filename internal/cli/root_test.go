package cli

import (
	"os"
	"testing"

	"athena/internal/config"
)

func TestApplyFlagOverrides(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		flags    map[string]string
		expected *config.Config
	}{
		{
			name: "port override",
			cfg: &config.Config{
				Port: config.DefaultPort,
			},
			flags: map[string]string{
				"port": "9000",
			},
			expected: &config.Config{
				Port: "9000",
			},
		},
		{
			name: "api-key override",
			cfg: &config.Config{
				APIKey: "original-key",
			},
			flags: map[string]string{
				"api-key": "new-key",
			},
			expected: &config.Config{
				APIKey: "new-key",
			},
		},
		{
			name: "base-url override",
			cfg: &config.Config{
				BaseURL: config.DefaultBaseURL,
			},
			flags: map[string]string{
				"base-url": "https://custom.url",
			},
			expected: &config.Config{
				BaseURL: "https://custom.url",
			},
		},
		{
			name: "model overrides",
			cfg: &config.Config{
				Model:       config.DefaultModelName,
				OpusModel:   config.DefaultOpusModel,
				SonnetModel: config.DefaultSonnetModel,
				HaikuModel:  config.DefaultHaikuModel,
			},
			flags: map[string]string{
				"model":        "custom/model",
				"model-opus":   "custom/opus",
				"model-sonnet": "custom/sonnet",
				"model-haiku":  "custom/haiku",
			},
			expected: &config.Config{
				Model:       "custom/model",
				OpusModel:   "custom/opus",
				SonnetModel: "custom/sonnet",
				HaikuModel:  "custom/haiku",
			},
		},
		{
			name: "empty flags don't override",
			cfg: &config.Config{
				Port:   "8080",
				APIKey: "test-key",
			},
			flags: map[string]string{},
			expected: &config.Config{
				Port:   "8080",
				APIKey: "test-key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flag variables
			port = ""
			apiKey = ""
			baseURL = ""
			model = ""
			opusModel = ""
			sonnetModel = ""
			haikuModel = ""

			// Set flag values
			if v, ok := tt.flags["port"]; ok {
				port = v
			}
			if v, ok := tt.flags["api-key"]; ok {
				apiKey = v
			}
			if v, ok := tt.flags["base-url"]; ok {
				baseURL = v
			}
			if v, ok := tt.flags["model"]; ok {
				model = v
			}
			if v, ok := tt.flags["model-opus"]; ok {
				opusModel = v
			}
			if v, ok := tt.flags["model-sonnet"]; ok {
				sonnetModel = v
			}
			if v, ok := tt.flags["model-haiku"]; ok {
				haikuModel = v
			}

			// Apply overrides
			applyFlagOverrides(tt.cfg)

			// Verify results
			if tt.cfg.Port != tt.expected.Port {
				t.Errorf("Port: got %v, want %v", tt.cfg.Port, tt.expected.Port)
			}
			if tt.cfg.APIKey != tt.expected.APIKey {
				t.Errorf("APIKey: got %v, want %v", tt.cfg.APIKey, tt.expected.APIKey)
			}
			if tt.cfg.BaseURL != tt.expected.BaseURL {
				t.Errorf("BaseURL: got %v, want %v", tt.cfg.BaseURL, tt.expected.BaseURL)
			}
			if tt.cfg.Model != tt.expected.Model {
				t.Errorf("Model: got %v, want %v", tt.cfg.Model, tt.expected.Model)
			}
			if tt.cfg.OpusModel != tt.expected.OpusModel {
				t.Errorf("OpusModel: got %v, want %v", tt.cfg.OpusModel, tt.expected.OpusModel)
			}
			if tt.cfg.SonnetModel != tt.expected.SonnetModel {
				t.Errorf("SonnetModel: got %v, want %v", tt.cfg.SonnetModel, tt.expected.SonnetModel)
			}
			if tt.cfg.HaikuModel != tt.expected.HaikuModel {
				t.Errorf("HaikuModel: got %v, want %v", tt.cfg.HaikuModel, tt.expected.HaikuModel)
			}
		})
	}
}

func TestRootCommandDefaultValues(t *testing.T) {
	// Verify that the root command has all expected persistent flags
	expectedFlags := []string{
		"config",
		"port",
		"api-key",
		"base-url",
		"model",
		"model-opus",
		"model-sonnet",
		"model-haiku",
	}

	for _, flagName := range expectedFlags {
		flag := rootCmd.PersistentFlags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected persistent flag %q not found", flagName)
		}
	}
}

func TestFlagPrecedence(t *testing.T) {
	// Test that CLI flags take precedence over config files and env vars
	// Set environment variable
	os.Setenv("PORT", "7777")
	defer os.Unsetenv("PORT")

	// Create temporary config file
	configContent := `port: "8888"`
	tmpfile, err := os.CreateTemp("", "athena-test-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load config with file
	cfg := config.Load(tmpfile.Name())

	// Set CLI flag
	port = "9999"
	defer func() { port = "" }()

	// Apply flag overrides
	applyFlagOverrides(cfg)

	// CLI flag should take precedence
	if cfg.Port != "9999" {
		t.Errorf("Expected CLI flag to override config file and env var: got %v, want 9999", cfg.Port)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that existing usage patterns work correctly

	t.Run("default config loading", func(t *testing.T) {
		cfg := config.Load("")
		if cfg == nil {
			t.Fatal("Expected config to be loaded with empty path")
		}
		// Should have default values
		if cfg.Port != config.DefaultPort {
			t.Errorf("Expected default port %v, got %v", config.DefaultPort, cfg.Port)
		}
	})

	t.Run("flag override on default config", func(t *testing.T) {
		cfg := config.Load("")

		// Simulate CLI flag
		port = "9000"
		defer func() { port = "" }()

		applyFlagOverrides(cfg)

		if cfg.Port != "9000" {
			t.Errorf("Expected port 9000, got %v", cfg.Port)
		}
	})
}
