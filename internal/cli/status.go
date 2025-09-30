package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"athena/internal/daemon"
)

const (
	// UptimeRoundingPrecision is the precision for uptime display (1 second)
	UptimeRoundingPrecision = time.Second
)

var (
	statusJSON bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Athena daemon status",
	Long:  `Display the current status of the Athena daemon including PID, port, and uptime.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Get status
		status, err := daemon.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		if statusJSON {
			// Output as JSON
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(status)
		}

		// Human-readable output
		if !status.Running {
			fmt.Println("Daemon: Not running")
			return nil
		}

		fmt.Println("Athena Daemon Status")
		fmt.Println("====================")
		fmt.Printf("Status:  Running\n")
		fmt.Printf("PID:     %d\n", status.PID)
		fmt.Printf("Port:    %d\n", status.Port)
		fmt.Printf("Uptime:  %v\n", status.Uptime.Round(UptimeRoundingPrecision))
		fmt.Printf("Started: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
		if status.ConfigPath != "" {
			fmt.Printf("Config:  %s\n", status.ConfigPath)
		}
		fmt.Printf("Logs:    ~/.athena/athena.log\n")

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	rootCmd.AddCommand(statusCmd)
}
