// Package internal provides the command-line interface for Athena using Cobra.
// It implements subcommands for daemon management (start, stop, status).
package internal

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"athena/internal/config"
	"athena/internal/daemon"
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
	logFormat   string
	logLevel    string
	logFile     string

	// Command-specific flags
	statusJSON  bool
	stopTimeout time.Duration
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "athena",
	Short: "Athena - Anthropic to OpenRouter proxy",
	Long: `Athena is an HTTP proxy server that translates Anthropic API requests
to OpenRouter format, enabling Claude Code to work with OpenRouter's diverse
model selection.

By default, runs the HTTP server in foreground mode.
Use 'athena start' to run as a background daemon.`,
	// Initialize logger before running any command
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		// Try to load config, but don't fail if API key is missing
		// (some commands like stop/status don't need it)
		cfg, err := config.New(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		applyFlagOverrides(cfg)
		initLogger(cfg)
		return nil
	},
	// Default behavior: Run server in foreground
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := loadAndValidateConfig()
		if err != nil {
			return err
		}

		srv := server.New(cfg)
		return srv.Start()
	},
}

// startCmd starts the daemon in the background
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Athena daemon in the background",
	Long: `Start the Athena proxy server as a background daemon process.
The daemon will continue running after you close the terminal.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := loadAndValidateConfig()
		if err != nil {
			return err
		}

		status, err := daemon.StartWithConfig(cfg)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Daemon started successfully\n")
		fmt.Printf("  PID: %d\n", status.PID)
		fmt.Printf("  Port: %d\n", status.Port)
		fmt.Printf("  Logs: ~/.athena/athena.log\n")

		return nil
	},
}

// stopCmd stops the running daemon
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running Athena daemon",
	Long:  `Gracefully stop the Athena daemon process with a configurable timeout.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := daemon.StopDaemon(stopTimeout); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		fmt.Println("✓ Daemon stopped successfully")
		return nil
	},
}

// statusCmd shows daemon status
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Athena daemon status",
	Long:  `Display the current status of the Athena daemon including PID, port, and uptime.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		status, err := daemon.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		return daemon.DisplayStatus(status, statusJSON)
	},
}


func init() {
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (YAML)")
	rootCmd.PersistentFlags().StringVar(&port, "port", "", "Port to run the server on")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "OpenRouter API key")
	rootCmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "OpenRouter base URL")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "Default model to use")
	rootCmd.PersistentFlags().StringVar(&opusModel, "model-opus", "", "Model to map claude-opus requests to")
	rootCmd.PersistentFlags().StringVar(&sonnetModel, "model-sonnet", "", "Model to map claude-sonnet requests to")
	rootCmd.PersistentFlags().StringVar(&haikuModel, "model-haiku", "", "Model to map claude-haiku requests to")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "Log format: text or json")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level: debug, info, warn, error")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "Log file path (default: stdout)")

	// Command-specific flags
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	stopCmd.Flags().DurationVar(&stopTimeout, "timeout", 30*time.Second, "Graceful shutdown timeout")

	// Add subcommands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

// loadAndValidateConfig loads configuration and applies flag overrides
func loadAndValidateConfig() (*config.Config, error) {
	cfg, err := config.New(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	applyFlagOverrides(cfg)

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required. Use --api-key flag, config file, or OPENROUTER_API_KEY env var")
	}

	return cfg, nil
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
	if logFormat != "" {
		cfg.LogFormat = logFormat
	}
	if logLevel != "" {
		cfg.LogLevel = logLevel
	}
	if logFile != "" {
		cfg.LogFile = logFile
	}
}

// initLogger initializes the global slog logger with the configured level
func initLogger(cfg *config.Config) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
