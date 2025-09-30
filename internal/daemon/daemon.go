// Package daemon provides process management for the Athena daemon.
// It handles starting, stopping, and monitoring the background server process.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"athena/internal/config"
)

const (
	// DefaultPort is the default port for the Athena daemon
	DefaultPort = 11434
	// StopCheckInterval is how often to check if process has stopped
	StopCheckInterval = 100 * time.Millisecond
)

// Status represents daemon status information
type Status struct {
	Running    bool
	PID        int
	Port       int
	Uptime     time.Duration
	StartTime  time.Time
	ConfigPath string
}

// StartDaemon starts the proxy server as a background daemon
func StartDaemon(cfg *config.Config) error {
	// Check if daemon already running
	if IsRunning() {
		return fmt.Errorf("daemon already running")
	}

	// Ensure data directory exists
	if err := ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Get log file path
	logPath, err := GetLogFilePath()
	if err != nil {
		return fmt.Errorf("failed to get log file path: %w", err)
	}

	// Open log file for output
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build command arguments
	args := []string{}
	if cfg.Port != "" {
		args = append(args, "--port", cfg.Port)
	}
	if cfg.APIKey != "" {
		args = append(args, "--api-key", cfg.APIKey)
	}
	if cfg.BaseURL != "" {
		args = append(args, "--base-url", cfg.BaseURL)
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.OpusModel != "" {
		args = append(args, "--model-opus", cfg.OpusModel)
	}
	if cfg.SonnetModel != "" {
		args = append(args, "--model-sonnet", cfg.SonnetModel)
	}
	if cfg.HaikuModel != "" {
		args = append(args, "--model-haiku", cfg.HaikuModel)
	}

	// Create command
	cmd := exec.Command(execPath, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Get port as integer
	port, _ := strconv.Atoi(cfg.Port)
	if port == 0 {
		port = DefaultPort
	}

	// Save state
	state := &ProcessState{
		PID:        cmd.Process.Pid,
		Port:       port,
		StartTime:  time.Now(),
		ConfigPath: "",
	}
	if err := SaveState(state); err != nil {
		// Try to kill the process we just started and reap it
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait() // Reap zombie process
		}
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Release the process (don't wait for it)
	_ = cmd.Process.Release()

	return nil
}

// StopDaemon gracefully stops the running daemon
func StopDaemon(timeout time.Duration) error {
	// Load state
	state, err := LoadState()
	if err != nil {
		return fmt.Errorf("daemon not running: %w", err)
	}

	// Find the process
	process, err := os.FindProcess(state.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit with timeout
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsProcessRunning(state.PID) {
			// Process exited, cleanup state
			_ = CleanupState()
			return nil
		}
		time.Sleep(StopCheckInterval)
	}

	// Timeout reached, force kill
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	// Cleanup state
	_ = CleanupState()
	return nil
}

// GetStatus returns current daemon status
func GetStatus() (*Status, error) {
	// Load state
	state, err := LoadState()
	if err != nil {
		return &Status{Running: false}, nil
	}

	// Calculate uptime
	uptime := time.Since(state.StartTime)

	return &Status{
		Running:    true,
		PID:        state.PID,
		Port:       state.Port,
		Uptime:     uptime,
		StartTime:  state.StartTime,
		ConfigPath: state.ConfigPath,
	}, nil
}

// IsRunning checks if daemon is currently running
func IsRunning() bool {
	state, err := LoadState()
	if err != nil {
		return false
	}
	return IsProcessRunning(state.PID)
}
