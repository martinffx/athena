// Package daemon provides process management for the Athena daemon.
package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ProcessState represents daemon process state stored in PID file
type ProcessState struct {
	PID        int       `json:"pid"`
	Port       int       `json:"port"`
	StartTime  time.Time `json:"start_time"`
	ConfigPath string    `json:"config_path"`
}

// SaveState writes process state to PID file atomically
func SaveState(state *ProcessState) error {
	// Ensure data directory exists
	if err := ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Get PID file path
	pidPath, err := GetPIDFilePath()
	if err != nil {
		return err
	}

	// Marshal state to JSON
	data, err := state.marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write atomically: create temp file, write, then rename
	tmpPath := pidPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, pidPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}

// LoadState reads and validates process state from PID file
func LoadState() (*ProcessState, error) {
	pidPath, err := GetPIDFilePath()
	if err != nil {
		return nil, err
	}

	// Read PID file
	data, err := os.ReadFile(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("daemon not running: PID file not found")
		}
		return nil, fmt.Errorf("failed to read PID file: %w", err)
	}

	// Parse state
	state, err := unmarshalState(data)
	if err != nil {
		return nil, err
	}

	// Validate state
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	// Check if process is still running
	if !IsProcessRunning(state.PID) {
		return nil, fmt.Errorf("daemon not running: process %d not found", state.PID)
	}

	return state, nil
}

// CleanupState removes PID file and cleans up state
func CleanupState() error {
	pidPath, err := GetPIDFilePath()
	if err != nil {
		return err
	}

	// Remove PID file (ignore if not exists)
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

// Validate performs business rule validation on process state
func (s *ProcessState) Validate() error {
	// Validate PID
	if s.PID <= 0 {
		return fmt.Errorf("invalid PID: must be positive")
	}

	// Validate port range
	if s.Port < 1024 || s.Port > 65535 {
		return fmt.Errorf("invalid port: must be between 1024 and 65535")
	}

	// Validate start time is not in the future
	if s.StartTime.After(time.Now()) {
		return fmt.Errorf("invalid start time: cannot be in the future")
	}

	return nil
}

// getDataDirImpl is the actual implementation
func getDataDirImpl() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	dataDir := filepath.Join(home, ".athena")
	return dataDir, nil
}

// GetDataDir returns the directory where daemon state files are stored
// This is a variable to allow overriding in tests
var GetDataDir = getDataDirImpl

// GetPIDFilePath returns the path to the PID file
func GetPIDFilePath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "athena.pid"), nil
}

// GetLogFilePath returns the path to the log file
func GetLogFilePath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "athena.log"), nil
}

// ensureDataDir creates the data directory if it doesn't exist
func ensureDataDir() error {
	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dataDir, 0755)
}

// IsProcessRunning checks if a process with the given PID is running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	// Signal(syscall.Signal(0)) checks if process exists without actually sending a signal
	err = process.Signal(os.Signal(syscall.Signal(0)))
	return err == nil
}

// marshal converts ProcessState to JSON bytes
func (s *ProcessState) marshal() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// unmarshalState parses JSON bytes into ProcessState
func unmarshalState(data []byte) (*ProcessState, error) {
	var state ProcessState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}
	return &state, nil
}
