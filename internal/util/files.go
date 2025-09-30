package util

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Global mutex map for file locking (mutex-based implementation)
var fileLocks = make(map[string]*sync.Mutex)
var fileLocksMapMutex sync.Mutex

// WriteJSONFile atomically writes JSON data to a file using temp file + rename
func WriteJSONFile(path string, data interface{}) error {
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

// ReadJSONFile safely reads and unmarshals JSON data from a file
func ReadJSONFile(path string, data interface{}) error {
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

// LockFile acquires an exclusive lock on a file path and returns unlock function
func LockFile(path string) (func(), error) {
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
