package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"athena/internal/daemon"
)

var (
	stopTimeout time.Duration
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running Athena daemon",
	Long:  `Gracefully stop the Athena daemon process with a configurable timeout.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Stop daemon
		if err := daemon.StopDaemon(stopTimeout); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}

		fmt.Println("✓ Daemon stopped successfully")
		return nil
	},
}

func init() {
	stopCmd.Flags().DurationVar(&stopTimeout, "timeout", 30*time.Second, "Graceful shutdown timeout")
	rootCmd.AddCommand(stopCmd)
}
