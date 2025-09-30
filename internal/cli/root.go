// Package cli provides the command-line interface for Athena using Cobra.
// It implements subcommands for daemon management (start, stop, status, logs)
// and Claude Code integration.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"athena/internal/config"
	"athena/internal/server"
)

var (
	// Persistent flags (available to all subcommands)
	configFile  string
	port        string
	apiKey      string
	baseURL     string
	model       string
	opusModel   string
	sonnetModel string
	haikuModel  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "athena",
	Short: "Athena - Anthropic to OpenRouter proxy",
	Long: `Athena is an HTTP proxy server that translates Anthropic API requests
to OpenRouter format, enabling Claude Code to work with OpenRouter's diverse
model selection.`,
	// Default behavior: Start server in foreground (backward compatibility)
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load configuration
		cfg := config.Load(configFile)

		// Apply flag overrides
		applyFlagOverrides(cfg)

		// Validate required config
		if cfg.APIKey == "" {
			return fmt.Errorf("OpenRouter API key is required. Use -api-key flag, config file, or OPENROUTER_API_KEY env var")
		}

		// Start server directly (blocking, legacy mode)
		srv := server.New(cfg)
		return srv.Start()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (JSON/YAML)")
	rootCmd.PersistentFlags().StringVar(&port, "port", "", "Port to run the server on")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "OpenRouter API key")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "OpenRouter base URL")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "Default model to use")
	rootCmd.PersistentFlags().StringVar(&opusModel, "model-opus", "", "Model to map claude-opus requests to")
	rootCmd.PersistentFlags().StringVar(&sonnetModel, "model-sonnet", "", "Model to map claude-sonnet requests to")
	rootCmd.PersistentFlags().StringVar(&haikuModel, "model-haiku", "", "Model to map claude-haiku requests to")
}

// applyFlagOverrides applies command-line flag overrides to the config
func applyFlagOverrides(cfg *config.Config) {
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
}
