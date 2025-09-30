package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const (
	osWindows = "windows"
)

// GetDataDir returns the path to the data directory (~/.athena/)
func GetDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	dataDir := filepath.Join(homeDir, ".athena")
	return dataDir, nil
}

// EnsureDataDir creates the data directory if it doesn't exist with 0755 permissions
func EnsureDataDir() error {
	dataDir, err := GetDataDir()
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

// GetPidFilePath returns the path to the PID file (~/.athena/athena.pid)
func GetPidFilePath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	pidPath := filepath.Join(dataDir, "athena.pid")
	return pidPath, nil
}

// GetLogFilePath returns the path to the log file (~/.athena/athena.log)
func GetLogFilePath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	logPath := filepath.Join(dataDir, "athena.log")
	return logPath, nil
}

// IsProcessRunning returns true if the process with the given PID is running
func IsProcessRunning(pid int) bool {
	// Handle invalid PIDs
	if pid <= 0 {
		return false
	}

	// Platform-specific process detection
	if runtime.GOOS == osWindows {
		return isProcessRunningWindows(pid)
	}
	return isProcessRunningUnix(pid)
}

// isProcessRunningUnix checks if a process is running on Unix-like systems
func isProcessRunningUnix(pid int) bool {
	// Use kill with signal 0 to test if process exists
	// Signal 0 doesn't actually send a signal, just checks if we can send one
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, os.FindProcess always succeeds, so we need to actually test
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		return false // Process doesn't exist
	}

	// Process exists, but might be a zombie
	// Check /proc/[pid]/stat to see if it's a zombie (Linux)
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		// /proc not available (e.g., macOS), try ps command
		// Validate PID to prevent command injection
		if pid <= 0 {
			return false
		}
		pidStr := strconv.Itoa(pid)
		cmd := exec.Command("ps", "-p", pidStr, "-o", "stat=") // #nosec G204 - pidStr is validated integer from strconv.Itoa
		output, err := cmd.Output()
		if err != nil {
			// Can't determine state, assume running if signal(0) worked
			return true
		}

		// Check if output contains 'Z' for zombie
		state := strings.TrimSpace(string(output))
		return !strings.Contains(state, "Z")
	}

	// Parse /proc/[pid]/stat - state is the third field after the command in parentheses
	statStr := string(data)
	lastParen := strings.LastIndex(statStr, ")")
	if lastParen != -1 && len(statStr) > lastParen+2 {
		state := statStr[lastParen+2]
		// 'Z' means zombie
		return state != 'Z'
	}

	return true // Assume running if we can't determine state
}

// isProcessRunningWindows checks if a process is running on Windows
func isProcessRunningWindows(pid int) bool {
	// On Windows, we can use os.FindProcess and then try to send signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to send signal 0 - this works on Windows too
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// KillProcess terminates the process with the given PID
// graceful=true uses SIGTERM (Unix) or os.Interrupt (Windows)
// graceful=false uses SIGKILL (Unix) or process.Kill() (Windows)
func KillProcess(pid int, graceful bool) error {
	// Validate PID
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Check if process exists
	if !IsProcessRunning(pid) {
		return fmt.Errorf("process with PID %d is not running", pid)
	}

	// Get the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", pid, err)
	}

	// Platform-specific killing
	if runtime.GOOS == osWindows {
		return killProcessWindows(process, graceful)
	}
	return killProcessUnix(process, graceful)
}

// killProcessUnix terminates a process on Unix-like systems
func killProcessUnix(process *os.Process, graceful bool) error {
	if graceful {
		// Send SIGTERM for graceful shutdown
		err := process.Signal(syscall.SIGTERM)
		if err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}
	} else {
		// Send SIGKILL for forced termination
		err := process.Signal(syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("failed to send SIGKILL: %w", err)
		}
	}
	return nil
}

// killProcessWindows terminates a process on Windows
func killProcessWindows(process *os.Process, graceful bool) error {
	if graceful {
		// On Windows, we can try os.Interrupt, but it's not always reliable
		// For now, we'll use Kill() for both cases as Windows doesn't have
		// the same signal handling as Unix
		err := process.Signal(os.Interrupt)
		if err != nil {
			// If interrupt fails, fall back to Kill()
			return process.Kill()
		}
		return nil
	} else {
		// Forced termination
		return process.Kill()
	}
}
