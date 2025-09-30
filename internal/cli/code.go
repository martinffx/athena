package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"athena/internal/config"
	"athena/internal/daemon"
)

var codeCmd = &cobra.Command{
	Use:   "code [args...]",
	Short: "Start daemon and launch Claude Code",
	Long: `Start the Athena daemon (if not running) and launch Claude Code with
the correct environment variables configured automatically.

Any additional arguments are passed through to the claude command.`,
	RunE: func(_ *cobra.Command, args []string) error {
		// Check if daemon is already running
		if !daemon.IsRunning() {
			fmt.Println("Starting Athena daemon...")

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

			fmt.Println("✓ Daemon started")
		}

		// Get daemon status to determine port
		status, err := daemon.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get daemon status: %w", err)
		}

		// Set environment variables for Claude Code
		baseURL := fmt.Sprintf("http://localhost:%d/v1", status.Port)
		os.Setenv("ANTHROPIC_BASE_URL", baseURL)
		os.Setenv("ANTHROPIC_API_KEY", "dummy") // Required but not used (we use X-Api-Key header)

		fmt.Printf("✓ Environment configured:\n")
		fmt.Printf("  ANTHROPIC_BASE_URL=%s\n", baseURL)
		fmt.Printf("  ANTHROPIC_API_KEY=dummy\n")
		fmt.Println()

		// Find claude executable
		claudePath, err := exec.LookPath("claude")
		if err != nil {
			return fmt.Errorf("claude command not found in PATH. Please install Claude Code: https://claude.ai/download")
		}

		// Launch Claude Code
		fmt.Println("Launching Claude Code...")
		cmd := exec.Command(claudePath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ() // Pass through all environment variables

		// Run claude and handle exit codes properly
		err = cmd.Run()
		if err != nil {
			// Check if it's an exit error (claude exited with non-zero status)
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Exit with the same code as claude
				os.Exit(exitErr.ExitCode())
			}
			// This was an actual execution error (not just non-zero exit)
			return fmt.Errorf("failed to run claude: %w", err)
		}

		// Claude exited successfully (exit code 0)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(codeCmd)
}
