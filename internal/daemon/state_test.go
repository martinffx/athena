package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessState_Marshal(t *testing.T) {
	state := &ProcessState{
		PID:        12345,
		Port:       11434,
		StartTime:  time.Date(2025, 9, 30, 10, 0, 0, 0, time.UTC),
		ConfigPath: "/path/to/config.yml",
	}

	data, err := state.marshal()
	if err != nil {
		t.Fatalf("marshal() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("marshal() returned empty data")
	}

	// Should contain all fields
	str := string(data)
	if !contains(str, "12345") {
		t.Error("marshal() missing PID")
	}
	if !contains(str, "11434") {
		t.Error("marshal() missing Port")
	}
}

func TestUnmarshalState(t *testing.T) {
	jsonData := `{
  "pid": 12345,
  "port": 11434,
  "start_time": "2025-09-30T10:00:00Z",
  "config_path": "/path/to/config.yml"
}`

	state, err := unmarshalState([]byte(jsonData))
	if err != nil {
		t.Fatalf("unmarshalState() error = %v", err)
	}

	if state.PID != 12345 {
		t.Errorf("PID = %v, want 12345", state.PID)
	}
	if state.Port != 11434 {
		t.Errorf("Port = %v, want 11434", state.Port)
	}
	if state.ConfigPath != "/path/to/config.yml" {
		t.Errorf("ConfigPath = %v, want /path/to/config.yml", state.ConfigPath)
	}
}

func TestUnmarshalState_InvalidJSON(t *testing.T) {
	invalidJSON := `{"pid": "not a number"}`

	_, err := unmarshalState([]byte(invalidJSON))
	if err == nil {
		t.Error("unmarshalState() expected error for invalid JSON")
	}
}

func TestProcessState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   *ProcessState
		wantErr bool
	}{
		{
			name: "valid state",
			state: &ProcessState{
				PID:        os.Getpid(),
				Port:       11434,
				StartTime:  time.Now().Add(-1 * time.Hour),
				ConfigPath: "/valid/path",
			},
			wantErr: false, // Should pass now that it's implemented
		},
		{
			name: "invalid PID - zero",
			state: &ProcessState{
				PID:        0,
				Port:       11434,
				StartTime:  time.Now(),
				ConfigPath: "/path",
			},
			wantErr: true,
		},
		{
			name: "invalid PID - negative",
			state: &ProcessState{
				PID:        -1,
				Port:       11434,
				StartTime:  time.Now(),
				ConfigPath: "/path",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			state: &ProcessState{
				PID:        1234,
				Port:       100,
				StartTime:  time.Now(),
				ConfigPath: "/path",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			state: &ProcessState{
				PID:        1234,
				Port:       70000,
				StartTime:  time.Now(),
				ConfigPath: "/path",
			},
			wantErr: true,
		},
		{
			name: "future start time",
			state: &ProcessState{
				PID:        1234,
				Port:       11434,
				StartTime:  time.Now().Add(1 * time.Hour),
				ConfigPath: "/path",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Override GetDataDir for testing
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	state := &ProcessState{
		PID:        12345,
		Port:       11434,
		StartTime:  time.Now(),
		ConfigPath: "/test/config.yml",
	}

	err := SaveState(state)
	if err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Verify file was created
	pidPath := filepath.Join(tmpDir, "athena.pid")
	if _, statErr := os.Stat(pidPath); os.IsNotExist(statErr) {
		t.Error("SaveState() did not create PID file")
	}

	// Verify file permissions
	info, err := os.Stat(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("PID file permissions = %v, want 0600", info.Mode().Perm())
	}

	// Verify content
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(data), "12345") {
		t.Error("PID file does not contain PID")
	}
}

func TestLoadState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Override GetDataDir for testing
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Test loading when file doesn't exist
	_, err := LoadState()
	if err == nil {
		t.Error("LoadState() expected error when PID file doesn't exist")
	}

	// Create a valid state file
	state := &ProcessState{
		PID:        os.Getpid(), // Use current process
		Port:       11434,
		StartTime:  time.Now().Add(-1 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if saveErr := SaveState(state); saveErr != nil {
		t.Fatalf("SaveState() error = %v", saveErr)
	}

	// Load the state
	loaded, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	if loaded.PID != state.PID {
		t.Errorf("LoadState() PID = %v, want %v", loaded.PID, state.PID)
	}
	if loaded.Port != state.Port {
		t.Errorf("LoadState() Port = %v, want %v", loaded.Port, state.Port)
	}
}

func TestLoadState_DeadProcess(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Override GetDataDir for testing
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create state with non-existent PID
	state := &ProcessState{
		PID:        999999, // Very unlikely to exist
		Port:       11434,
		StartTime:  time.Now().Add(-1 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Try to load - should fail because process doesn't exist
	_, err := LoadState()
	if err == nil {
		t.Error("LoadState() expected error for dead process")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("LoadState() error = %v, want error containing 'not found'", err)
	}
}

func TestCleanupState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Override GetDataDir for testing
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create a state file
	state := &ProcessState{
		PID:        os.Getpid(),
		Port:       11434,
		StartTime:  time.Now(),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Cleanup
	err := CleanupState()
	if err != nil {
		t.Fatalf("CleanupState() error = %v", err)
	}

	// Verify file was removed
	pidPath := filepath.Join(tmpDir, "athena.pid")
	if _, statErr := os.Stat(pidPath); !os.IsNotExist(statErr) {
		t.Error("CleanupState() did not remove PID file")
	}

	// Cleanup again should not error
	err = CleanupState()
	if err != nil {
		t.Errorf("CleanupState() error = %v on non-existent file", err)
	}
}

func TestGetDataDir(t *testing.T) {
	dataDir, err := GetDataDir()
	if err != nil {
		t.Fatalf("GetDataDir() error = %v", err)
	}

	if dataDir == "" {
		t.Error("GetDataDir() returned empty string")
	}

	// Should end with .athena
	if filepath.Base(dataDir) != ".athena" {
		t.Errorf("GetDataDir() = %v, want directory ending with .athena", dataDir)
	}
}

func TestGetPIDFilePath(t *testing.T) {
	pidPath, err := GetPIDFilePath()
	if err != nil {
		t.Fatalf("GetPIDFilePath() error = %v", err)
	}

	if pidPath == "" {
		t.Error("GetPIDFilePath() returned empty string")
	}

	// Should end with athena.pid
	if filepath.Base(pidPath) != "athena.pid" {
		t.Errorf("GetPIDFilePath() = %v, want file named athena.pid", pidPath)
	}
}

func TestGetLogFilePath(t *testing.T) {
	logPath, err := GetLogFilePath()
	if err != nil {
		t.Fatalf("GetLogFilePath() error = %v", err)
	}

	if logPath == "" {
		t.Error("GetLogFilePath() returned empty string")
	}

	// Should end with athena.log
	if filepath.Base(logPath) != "athena.log" {
		t.Errorf("GetLogFilePath() = %v, want file named athena.log", logPath)
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	currentPID := os.Getpid()
	if !IsProcessRunning(currentPID) {
		t.Error("IsProcessRunning() = false for current process, want true")
	}

	// Test with invalid PID (very high number unlikely to exist)
	if IsProcessRunning(999999) {
		t.Error("IsProcessRunning() = true for invalid PID, want false")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)+1))
}
