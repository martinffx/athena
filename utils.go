package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// getDataDir returns the path to the data directory (~/.openrouter-cc/)
func getDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".openrouter-cc")
	return dataDir, nil
}

// ensureDataDir creates the data directory if it doesn't exist with 0755 permissions
func ensureDataDir() error {
	dataDir, err := getDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	// Check if directory already exists
	if info, err := os.Stat(dataDir); err == nil {
		if info.IsDir() {
			return nil // Directory already exists
		}
		return fmt.Errorf("path exists but is not a directory: %s", dataDir)
	}

	// Create directory with 0755 permissions
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	return nil
}

// getPidFilePath returns the path to the PID file (~/.openrouter-cc/openrouter-cc.pid)
func getPidFilePath() (string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	pidPath := filepath.Join(dataDir, "openrouter-cc.pid")
	return pidPath, nil
}

// getLogFilePath returns the path to the log file (~/.openrouter-cc/openrouter-cc.log)
func getLogFilePath() (string, error) {
	dataDir, err := getDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	logPath := filepath.Join(dataDir, "openrouter-cc.log")
	return logPath, nil
}
