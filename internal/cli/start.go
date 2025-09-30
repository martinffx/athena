package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"athena/internal/config"
	"athena/internal/daemon"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Athena daemon in the background",
	Long: `Start the Athena proxy server as a background daemon process.
The daemon will continue running after you close the terminal.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Load configuration
		cfg := config.Load(configFile)

		// Apply flag overrides
		applyFlagOverrides(cfg)

		// Validate required config
		if cfg.APIKey == "" {
			return fmt.Errorf("OpenRouter API key is required. Use --api-key flag, config file, or OPENROUTER_API_KEY env var")
		}

		// Start daemon
		if err := daemon.StartDaemon(cfg); err != nil {
			return fmt.Errorf("failed to start daemon: %w", err)
		}

		// Get status to display PID
		status, err := daemon.GetStatus()
		if err != nil {
			return fmt.Errorf("daemon started but failed to get status: %w", err)
		}

		fmt.Printf("✓ Daemon started successfully\n")
		fmt.Printf("  PID: %d\n", status.PID)
		fmt.Printf("  Port: %d\n", status.Port)
		fmt.Printf("  Logs: ~/.athena/athena.log\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
