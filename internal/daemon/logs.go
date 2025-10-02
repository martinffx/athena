package daemon

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

const (
	// MaxLogLineLength is the maximum allowed length for a log line (1MB)
	MaxLogLineLength = 1024 * 1024
	// LogPollInterval is how often to check for new log entries
	LogPollInterval = 100 * time.Millisecond
)

// ShowLastLines displays the last N lines from the log file
func ShowLastLines(logPath string, n int) error {
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Read all lines into memory (simple approach for small log files)
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	// Show last N lines
	start := len(lines) - n
	if start < 0 {
		start = 0
	}

	for i := start; i < len(lines); i++ {
		fmt.Println(lines[i])
	}

	return nil
}

// FollowLogs tails the log file and streams new entries
func FollowLogs(logPath string) error {
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Seek to end of file
	_, err = file.Seek(0, 2)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Set buffer size limit to prevent unbounded memory growth
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, MaxLogLineLength)

	for {
		// Read available lines
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading log file: %w", err)
		}

		// Wait a bit before checking for more lines
		time.Sleep(LogPollInterval)

		// Check if daemon is still running
		if !IsRunning() {
			fmt.Println("\n[Daemon stopped]")
			return nil
		}
	}
}
