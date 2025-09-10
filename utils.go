package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const (
	osWindows = "windows"
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

// F002: Process Management Functions

// isProcessRunning returns true if the process with the given PID is running
func isProcessRunning(pid int) bool {
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

// killProcess terminates the process with the given PID
// graceful=true uses SIGTERM (Unix) or os.Interrupt (Windows)
// graceful=false uses SIGKILL (Unix) or process.Kill() (Windows)
func killProcess(pid int, graceful bool) error {
	// Validate PID
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Check if process exists
	if !isProcessRunning(pid) {
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

// F003: File Operations and Locking Functions

// Global mutex map for file locking (mutex-based implementation)
var fileLocks = make(map[string]*sync.Mutex)
var fileLocksMapMutex sync.Mutex

// writeJSONFile atomically writes JSON data to a file using temp file + rename
func writeJSONFile(path string, data interface{}) error {
	// Validate input
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Marshal data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, mkdirErr)
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write JSON data to temporary file
	if _, err := tmpFile.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write data to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the temporary file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Set tmpFile to nil to prevent cleanup in defer
	tmpFile = nil

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up on rename failure
		return fmt.Errorf("failed to rename temporary file to target: %w", err)
	}

	return nil
}

// readJSONFile safely reads and unmarshals JSON data from a file
func readJSONFile(path string, data interface{}) error {
	// Validate input
	if data == nil {
		return fmt.Errorf("data target cannot be nil")
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Unmarshal JSON data
	if err := json.Unmarshal(content, data); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from file %s: %w", path, err)
	}

	return nil
}

// lockFile acquires an exclusive lock on a file path and returns unlock function
func lockFile(path string) (func(), error) {
	// Normalize path for consistent locking
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	// Get or create mutex for this file path
	fileLocksMapMutex.Lock()
	mutex, exists := fileLocks[absPath]
	if !exists {
		mutex = &sync.Mutex{}
		fileLocks[absPath] = mutex
	}
	fileLocksMapMutex.Unlock()

	// Acquire the lock
	mutex.Lock()

	// Return unlock function
	unlock := func() {
		mutex.Unlock()
	}

	return unlock, nil
}
