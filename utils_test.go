package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// F001: Data Directory Management Functions Tests

func TestGetDataDir(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "returns home directory based path",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getDataDir()

			if tt.expectError && err == nil {
				t.Error("getDataDir() expected error but got nil")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("getDataDir() unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				// Should end with .openrouter-cc
				if !strings.HasSuffix(result, ".openrouter-cc") {
					t.Errorf("getDataDir() = %q, should end with '.openrouter-cc'", result)
				}

				// Should be an absolute path
				if !filepath.IsAbs(result) {
					t.Errorf("getDataDir() = %q, should be absolute path", result)
				}

				// Should contain home directory
				homeDir, _ := os.UserHomeDir()
				if homeDir != "" && !strings.Contains(result, homeDir) {
					t.Errorf("getDataDir() = %q, should contain home directory %q", result, homeDir)
				}
			}
		})
	}
}

func TestEnsureDataDir(t *testing.T) {
	// Test using the real function - it should work with any home directory
	err := ensureDataDir()

	if err != nil {
		t.Errorf("ensureDataDir() unexpected error: %v", err)
		return
	}

	// Verify directory was created
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("getDataDir() failed: %v", err)
	}

	if info, err := os.Stat(dataDir); err != nil {
		t.Errorf("ensureDataDir() directory not created: %v", err)
	} else if !info.IsDir() {
		t.Error("ensureDataDir() created path is not a directory")
	}
}

func TestGetPidFilePath(t *testing.T) {
	result, err := getPidFilePath()

	if err != nil {
		t.Errorf("getPidFilePath() unexpected error: %v", err)
		return
	}

	// Should end with .pid
	if !strings.HasSuffix(result, ".pid") {
		t.Errorf("getPidFilePath() = %q, should end with '.pid'", result)
	}

	// Should contain .openrouter-cc
	if !strings.Contains(result, ".openrouter-cc") {
		t.Errorf("getPidFilePath() = %q, should contain '.openrouter-cc'", result)
	}

	// Should be absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("getPidFilePath() = %q, should be absolute path", result)
	}
}

func TestGetLogFilePath(t *testing.T) {
	result, err := getLogFilePath()

	if err != nil {
		t.Errorf("getLogFilePath() unexpected error: %v", err)
		return
	}

	// Should end with .log
	if !strings.HasSuffix(result, ".log") {
		t.Errorf("getLogFilePath() = %q, should end with '.log'", result)
	}

	// Should contain .openrouter-cc
	if !strings.Contains(result, ".openrouter-cc") {
		t.Errorf("getLogFilePath() = %q, should contain '.openrouter-cc'", result)
	}

	// Should be absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("getLogFilePath() = %q, should be absolute path", result)
	}
}

func TestDataDirectoryIntegration(t *testing.T) {
	// Test that all directory functions work together

	// Test getDataDir
	dataDir, err := getDataDir()
	if err != nil {
		t.Fatalf("getDataDir() failed: %v", err)
	}

	// Test ensureDataDir
	err = ensureDataDir()
	if err != nil {
		t.Fatalf("ensureDataDir() failed: %v", err)
	}

	// Test getPidFilePath
	pidPath, err := getPidFilePath()
	if err != nil {
		t.Fatalf("getPidFilePath() failed: %v", err)
	}

	expectedPidPath := filepath.Join(dataDir, "openrouter-cc.pid")
	if pidPath != expectedPidPath {
		t.Errorf("getPidFilePath() = %q, expected %q", pidPath, expectedPidPath)
	}

	// Test getLogFilePath
	logPath, err := getLogFilePath()
	if err != nil {
		t.Fatalf("getLogFilePath() failed: %v", err)
	}

	expectedLogPath := filepath.Join(dataDir, "openrouter-cc.log")
	if logPath != expectedLogPath {
		t.Errorf("getLogFilePath() = %q, expected %q", logPath, expectedLogPath)
	}

	// Verify directory exists
	if info, err := os.Stat(dataDir); err != nil {
		t.Errorf("Data directory not created: %v", err)
	} else if !info.IsDir() {
		t.Error("Data path is not a directory")
	}
}
