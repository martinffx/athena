package daemon

import (
	"os"
	"testing"
	"time"

	"athena/internal/config"
)

func TestStartDaemon_AlreadyRunning(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create a state file to simulate running daemon
	state := &ProcessState{
		PID:        os.Getpid(),
		Port:       11434,
		StartTime:  time.Now().Add(-1 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	cfg := &config.Config{
		Port:   "11434",
		APIKey: "test-key",
	}

	err := StartDaemon(cfg)
	if err == nil {
		t.Error("StartDaemon() expected error when daemon already running")
	}
	if err.Error() != "daemon already running" {
		t.Errorf("StartDaemon() error = %v, want 'daemon already running'", err)
	}
}

func TestStopDaemon_NotRunning(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	err := StopDaemon(1 * time.Second)
	if err == nil {
		t.Error("StopDaemon() expected error when daemon not running")
	}
}

func TestGetStatus_NotRunning(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	status, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.Running {
		t.Error("GetStatus() Running = true when daemon not running, want false")
	}
}

func TestGetStatus_Running(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create a state file
	state := &ProcessState{
		PID:        os.Getpid(),
		Port:       11434,
		StartTime:  time.Now().Add(-2 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	status, err := GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if !status.Running {
		t.Error("GetStatus() Running = false, want true")
	}
	if status.PID != os.Getpid() {
		t.Errorf("GetStatus() PID = %v, want %v", status.PID, os.Getpid())
	}
	if status.Port != 11434 {
		t.Errorf("GetStatus() Port = %v, want 11434", status.Port)
	}
	// Uptime should be approximately 2 hours
	if status.Uptime < 1*time.Hour || status.Uptime > 3*time.Hour {
		t.Errorf("GetStatus() Uptime = %v, want ~2h", status.Uptime)
	}
}

func TestIsRunning_NoDaemon(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// No PID file exists, should return false
	if IsRunning() {
		t.Error("IsRunning() = true when no daemon running, want false")
	}
}

func TestIsRunning_WithDaemon(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create a state file with current process PID
	state := &ProcessState{
		PID:        os.Getpid(),
		Port:       11434,
		StartTime:  time.Now().Add(-1 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Should return true
	if !IsRunning() {
		t.Error("IsRunning() = false when daemon state exists, want true")
	}
}

func TestIsRunning_DeadProcess(t *testing.T) {
	// Override GetDataDir to use temp directory
	tmpDir := t.TempDir()
	originalGetDataDir := GetDataDir
	GetDataDir = func() (string, error) {
		return tmpDir, nil
	}
	defer func() { GetDataDir = originalGetDataDir }()

	// Create state with non-existent PID
	state := &ProcessState{
		PID:        999999,
		Port:       11434,
		StartTime:  time.Now().Add(-1 * time.Hour),
		ConfigPath: "/test/config.yml",
	}
	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Should return false because process doesn't exist
	if IsRunning() {
		t.Error("IsRunning() = true for dead process, want false")
	}
}

func TestStatus_Fields(t *testing.T) {
	status := &Status{
		Running:    true,
		PID:        12345,
		Port:       11434,
		Uptime:     2 * time.Hour,
		StartTime:  time.Now().Add(-2 * time.Hour),
		ConfigPath: "/test/config.yml",
	}

	if !status.Running {
		t.Error("Status.Running = false, want true")
	}
	if status.PID != 12345 {
		t.Errorf("Status.PID = %v, want 12345", status.PID)
	}
	if status.Port != 11434 {
		t.Errorf("Status.Port = %v, want 11434", status.Port)
	}
	if status.Uptime != 2*time.Hour {
		t.Errorf("Status.Uptime = %v, want 2h", status.Uptime)
	}
}
