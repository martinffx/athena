# CLI Subcommands Implementation Tasks

This document provides detailed task-by-task implementation guidance for adding CLI subcommands to openrouter-cc using a simplified function-based approach with Go standard library only.

## Overview

**Architecture Approach**: Function-based implementation within single main.go file
**Dependencies**: Go standard library only (no Cobra, no external frameworks)
**Testing**: TDD approach with comprehensive unit and integration tests
**Compatibility**: 100% backward compatibility with existing CLI usage

## Phase 1: Foundation Functions (9 hours)

### F001: Data Directory Management Functions (2 hours)

**Purpose**: Create cross-platform functions to manage the `~/.openrouter-cc/` directory structure.

**Implementation**:

```go
// Add to main.go after existing Config struct
const DataDirName = ".openrouter-cc"

func getDataDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("unable to determine home directory: %w", err)
    }
    return filepath.Join(home, DataDirName), nil
}

func ensureDataDir() error {
    dataDir, err := getDataDir()
    if err != nil {
        return err
    }
    
    return os.MkdirAll(dataDir, 0755)
}

func getPidFilePath() (string, error) {
    dataDir, err := getDataDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dataDir, "openrouter-cc.pid"), nil
}

func getLogFilePath() (string, error) {
    dataDir, err := getDataDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dataDir, "openrouter-cc.log"), nil
}
```

**Testing Requirements**:
- Test directory creation with various home directory scenarios
- Test permission handling across platforms
- Test path resolution works correctly
- Test error handling for permission denied cases

**Dependencies**: None

---

### F002: Process Management Utility Functions (3 hours)

**Purpose**: Create cross-platform process detection and management utilities.

**Implementation**:

```go
import (
    // Add platform-specific imports as needed
    "os/exec"
    "strconv"
    "syscall"
    "time"
)

func isProcessAlive(pid int) bool {
    if pid <= 0 {
        return false
    }
    
    // Platform-specific implementation
    if runtime.GOOS == "windows" {
        return isProcessAliveWindows(pid)
    }
    return isProcessAliveUnix(pid)
}

func isProcessAliveUnix(pid int) bool {
    process, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    err = process.Signal(syscall.Signal(0))
    return err == nil
}

func isProcessAliveWindows(pid int) bool {
    cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
    err := cmd.Run()
    return err == nil
}

func killProcess(pid int, graceful bool) error {
    if !isProcessAlive(pid) {
        return nil // Process already dead
    }
    
    process, err := os.FindProcess(pid)
    if err != nil {
        return fmt.Errorf("failed to find process %d: %w", pid, err)
    }
    
    if graceful {
        // Send graceful termination signal
        if runtime.GOOS == "windows" {
            return process.Signal(os.Interrupt)
        }
        return process.Signal(syscall.SIGTERM)
    }
    
    // Force kill
    return process.Kill()
}
```

**Testing Requirements**:
- Test process detection with current process (should return true)
- Test process detection with non-existent PID (should return false)
- Test graceful vs forceful process termination
- Test cross-platform compatibility

**Dependencies**: None

---

### F003: File Operations and Locking Functions (4 hours)

**Purpose**: Create atomic file operations with proper locking for state management.

**Implementation**:

```go
import (
    "encoding/json"
    "os"
    "path/filepath"
    "sync"
)

// Simple file locking using mutex (can be enhanced with flock later)
var fileLocks = make(map[string]*sync.Mutex)
var fileLocksMutex sync.Mutex

func lockFile(path string) (unlock func(), err error) {
    fileLocksMutex.Lock()
    if fileLocks[path] == nil {
        fileLocks[path] = &sync.Mutex{}
    }
    mutex := fileLocks[path]
    fileLocksMutex.Unlock()
    
    mutex.Lock()
    return func() { mutex.Unlock() }, nil
}

func writeJSONFile(path string, data interface{}) error {
    unlock, err := lockFile(path)
    if err != nil {
        return err
    }
    defer unlock()
    
    // Atomic write: write to temp file, then rename
    dir := filepath.Dir(path)
    tmpFile, err := os.CreateTemp(dir, "tmp-*.json")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %w", err)
    }
    tmpPath := tmpFile.Name()
    
    defer func() {
        tmpFile.Close()
        os.Remove(tmpPath) // Clean up on error
    }()
    
    encoder := json.NewEncoder(tmpFile)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(data); err != nil {
        return fmt.Errorf("failed to encode JSON: %w", err)
    }
    
    if err := tmpFile.Close(); err != nil {
        return fmt.Errorf("failed to close temp file: %w", err)
    }
    
    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("failed to rename temp file: %w", err)
    }
    
    return nil
}

func readJSONFile(path string, data interface{}) error {
    unlock, err := lockFile(path)
    if err != nil {
        return err
    }
    defer unlock()
    
    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()
    
    decoder := json.NewDecoder(file)
    if err := decoder.Decode(data); err != nil {
        return fmt.Errorf("failed to decode JSON: %w", err)
    }
    
    return nil
}
```

**Testing Requirements**:
- Test atomic writes don't leave partial files on error
- Test concurrent file access is properly serialized
- Test file locking prevents race conditions
- Test error handling for corrupt JSON files

**Dependencies**: F001 (for directory structure)

---

## Phase 2: CLI Functions (9 hours)

### C001: Command-line Argument Detection (2 hours)

**Purpose**: Parse and validate CLI arguments without external dependencies.

**Implementation**:

```go
type CommandInfo struct {
    Name  string
    Args  []string
    Flags map[string]string
}

var knownCommands = []string{"start", "stop", "status", "logs", "code"}

func isLegacyMode(args []string) bool {
    if len(args) == 0 {
        return true // No arguments = legacy direct server mode
    }
    
    // If first argument starts with -, it's a flag (legacy mode)
    if strings.HasPrefix(args[0], "-") {
        return true
    }
    
    // Check if first argument is a known command
    for _, cmd := range knownCommands {
        if args[0] == cmd {
            return false // New subcommand mode
        }
    }
    
    return true // Unknown first argument = legacy mode
}

func parseCommand(args []string) (CommandInfo, error) {
    if len(args) == 0 {
        return CommandInfo{}, fmt.Errorf("no command specified")
    }
    
    cmd := CommandInfo{
        Name:  args[0],
        Args:  []string{},
        Flags: make(map[string]string),
    }
    
    // Simple flag parsing (enhance as needed)
    for i := 1; i < len(args); i++ {
        arg := args[i]
        if strings.HasPrefix(arg, "--") {
            // Long flag: --flag=value or --flag value
            if strings.Contains(arg, "=") {
                parts := strings.SplitN(arg, "=", 2)
                cmd.Flags[strings.TrimPrefix(parts[0], "--")] = parts[1]
            } else {
                flagName := strings.TrimPrefix(arg, "--")
                if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
                    cmd.Flags[flagName] = args[i+1]
                    i++ // Skip next arg
                } else {
                    cmd.Flags[flagName] = "true" // Boolean flag
                }
            }
        } else if strings.HasPrefix(arg, "-") {
            // Short flag: -f value or -f
            flagName := strings.TrimPrefix(arg, "-")
            if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
                cmd.Flags[flagName] = args[i+1]
                i++ // Skip next arg
            } else {
                cmd.Flags[flagName] = "true" // Boolean flag
            }
        } else {
            cmd.Args = append(cmd.Args, arg)
        }
    }
    
    return cmd, nil
}
```

**Testing Requirements**:
- Test legacy mode detection with various argument patterns
- Test command parsing handles flags correctly
- Test help text generation works for all commands
- Test error handling for invalid command combinations

**Dependencies**: None

---

### C002: Command Handler Functions (5 hours)

**Purpose**: Implement individual handler functions for each subcommand.

**Implementation**:

```go
func handleStartCommand(args []string) error {
    cmd, err := parseCommand(append([]string{"start"}, args...))
    if err != nil {
        return fmt.Errorf("failed to parse start command: %w", err)
    }
    
    // Check if help requested
    if _, ok := cmd.Flags["help"]; ok {
        fmt.Println("Usage: openrouter-cc start [options]")
        fmt.Println("  --port PORT     Port to bind (default: 11434)")
        fmt.Println("  --config FILE   Configuration file path")
        fmt.Println("  --help          Show this help")
        return nil
    }
    
    // Load configuration (reuse existing loadConfig logic)
    // Check if daemon already running
    // Start daemon process
    // Save process state
    // Report success
    
    fmt.Println("Starting OpenRouter CC daemon...")
    return fmt.Errorf("start command not yet implemented")
}

func handleStopCommand(args []string) error {
    cmd, err := parseCommand(append([]string{"stop"}, args...))
    if err != nil {
        return fmt.Errorf("failed to parse stop command: %w", err)
    }
    
    if _, ok := cmd.Flags["help"]; ok {
        fmt.Println("Usage: openrouter-cc stop [options]")
        fmt.Println("  --force         Force kill if graceful shutdown fails")
        fmt.Println("  --timeout SEC   Graceful shutdown timeout (default: 30)")
        fmt.Println("  --help          Show this help")
        return nil
    }
    
    fmt.Println("Stopping OpenRouter CC daemon...")
    return fmt.Errorf("stop command not yet implemented")
}

func handleStatusCommand(args []string) error {
    cmd, err := parseCommand(append([]string{"status"}, args...))
    if err != nil {
        return fmt.Errorf("failed to parse status command: %w", err)
    }
    
    if _, ok := cmd.Flags["help"]; ok {
        fmt.Println("Usage: openrouter-cc status [options]")
        fmt.Println("  --json      Output status as JSON")
        fmt.Println("  --verbose   Show detailed information")
        fmt.Println("  --help      Show this help")
        return nil
    }
    
    fmt.Println("OpenRouter CC Status:")
    return fmt.Errorf("status command not yet implemented")
}

func handleLogsCommand(args []string) error {
    cmd, err := parseCommand(append([]string{"logs"}, args...))
    if err != nil {
        return fmt.Errorf("failed to parse logs command: %w", err)
    }
    
    if _, ok := cmd.Flags["help"]; ok {
        fmt.Println("Usage: openrouter-cc logs [options]")
        fmt.Println("  -f, --follow    Follow log output")
        fmt.Println("  --lines NUM     Number of lines to show (default: 50)")
        fmt.Println("  --help          Show this help")
        return nil
    }
    
    fmt.Println("OpenRouter CC Logs:")
    return fmt.Errorf("logs command not yet implemented")
}

func handleCodeCommand(args []string) error {
    cmd, err := parseCommand(append([]string{"code"}, args...))
    if err != nil {
        return fmt.Errorf("failed to parse code command: %w", err)
    }
    
    if _, ok := cmd.Flags["help"]; ok {
        fmt.Println("Usage: openrouter-cc code [options]")
        fmt.Println("Start daemon if needed and launch Claude Code")
        fmt.Println("  --help          Show this help")
        return nil
    }
    
    fmt.Println("Starting daemon and launching Claude Code...")
    return fmt.Errorf("code command not yet implemented")
}
```

**Testing Requirements**:
- Test each command handler with valid arguments
- Test error handling for invalid arguments
- Test help text is displayed correctly
- Test command handlers provide appropriate user feedback

**Dependencies**: C001 (for command parsing)

---

### C003: Command Routing and Execution (2 hours)

**Purpose**: Route commands to appropriate handlers and manage execution flow.

**Implementation**:

```go
func routeCommand(args []string) error {
    if len(args) == 0 || isLegacyMode(args) {
        // Run in legacy mode - use existing main() logic
        return runLegacyMode(args)
    }
    
    command := args[0]
    commandArgs := args[1:]
    
    switch command {
    case "start":
        return handleStartCommand(commandArgs)
    case "stop":
        return handleStopCommand(commandArgs)
    case "status":
        return handleStatusCommand(commandArgs)
    case "logs":
        return handleLogsCommand(commandArgs)
    case "code":
        return handleCodeCommand(commandArgs)
    case "help", "--help", "-h":
        return displayHelp()
    case "version", "--version", "-v":
        return displayVersion()
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
        displayHelp()
        return fmt.Errorf("unknown command: %s", command)
    }
}

func displayHelp() error {
    fmt.Println("OpenRouter CC - Claude Code proxy server")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  openrouter-cc [flags]              # Direct server mode (legacy)")
    fmt.Println("  openrouter-cc <command> [options]  # Daemon mode")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  start    Start daemon process")
    fmt.Println("  stop     Stop daemon process")
    fmt.Println("  status   Show daemon status")
    fmt.Println("  logs     Show/follow daemon logs")
    fmt.Println("  code     Start daemon and launch Claude Code")
    fmt.Println("  help     Show this help")
    fmt.Println("  version  Show version information")
    fmt.Println()
    fmt.Println("Use 'openrouter-cc <command> --help' for command-specific help.")
    return nil
}

func displayVersion() error {
    fmt.Println("OpenRouter CC version: development")
    return nil
}

func runLegacyMode(args []string) error {
    // This will call the existing main() logic
    // Implementation will be completed in integration phase
    return fmt.Errorf("legacy mode not yet implemented")
}
```

**Testing Requirements**:
- Test command routing dispatches to correct handlers
- Test unknown commands display helpful error messages
- Test help and version commands work correctly
- Test legacy mode routing preserves existing behavior

**Dependencies**: C002 (for command handlers)

---

## Phase 3: Daemon Functions (13 hours)

### D001: Process State Management Functions (3 hours)

**Purpose**: Manage daemon process state persistence and validation.

**Implementation**:

```go
type ProcessState struct {
    PID        int       `json:"pid"`
    Port       int       `json:"port"`
    StartTime  time.Time `json:"start_time"`
    ConfigPath string    `json:"config_path,omitempty"`
}

func (p ProcessState) IsValid() bool {
    if p.PID <= 0 || p.Port <= 0 {
        return false
    }
    if p.StartTime.IsZero() || p.StartTime.After(time.Now()) {
        return false
    }
    return isProcessAlive(p.PID)
}

func saveProcessState(state ProcessState) error {
    pidPath, err := getPidFilePath()
    if err != nil {
        return fmt.Errorf("failed to get PID file path: %w", err)
    }
    
    if err := ensureDataDir(); err != nil {
        return fmt.Errorf("failed to ensure data directory: %w", err)
    }
    
    return writeJSONFile(pidPath, state)
}

func loadProcessState() (ProcessState, error) {
    var state ProcessState
    
    pidPath, err := getPidFilePath()
    if err != nil {
        return state, fmt.Errorf("failed to get PID file path: %w", err)
    }
    
    if err := readJSONFile(pidPath, &state); err != nil {
        if os.IsNotExist(err) {
            return state, fmt.Errorf("no daemon process state found")
        }
        return state, fmt.Errorf("failed to read process state: %w", err)
    }
    
    if !state.IsValid() {
        // Clean up stale PID file
        cleanupProcessState()
        return state, fmt.Errorf("daemon process no longer running")
    }
    
    return state, nil
}

func cleanupProcessState() error {
    pidPath, err := getPidFilePath()
    if err != nil {
        return err
    }
    
    if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove PID file: %w", err)
    }
    
    return nil
}
```

**Testing Requirements**:
- Test state save/load round-trip works correctly
- Test state validation detects stale processes
- Test cleanup removes only stale files
- Test error handling for corrupt state files

**Dependencies**: F003 (for file operations)

---

### D002: Daemon Lifecycle Functions (6 hours)

**Purpose**: Implement daemon process creation, monitoring, and termination.

**Implementation**:

```go
type DaemonStatus struct {
    Running   bool      `json:"running"`
    PID       int       `json:"pid,omitempty"`
    Port      int       `json:"port,omitempty"`
    Uptime    string    `json:"uptime,omitempty"`
    StartTime time.Time `json:"start_time,omitempty"`
}

func isDaemonRunning() bool {
    state, err := loadProcessState()
    return err == nil && state.IsValid()
}

func getDaemonStatus() (DaemonStatus, error) {
    status := DaemonStatus{Running: false}
    
    state, err := loadProcessState()
    if err != nil {
        return status, nil // Daemon not running
    }
    
    status.Running = true
    status.PID = state.PID
    status.Port = state.Port
    status.StartTime = state.StartTime
    status.Uptime = time.Since(state.StartTime).Round(time.Second).String()
    
    return status, nil
}

func startDaemon(config Config) error {
    // Check if daemon already running
    if isDaemonRunning() {
        return fmt.Errorf("daemon is already running")
    }
    
    // Validate port availability
    if err := checkPortAvailable(config.Port); err != nil {
        return fmt.Errorf("port %s not available: %w", config.Port, err)
    }
    
    // Start daemon process
    cmd := exec.Command(os.Args[0], "--daemon-mode")
    
    // Set up daemon process attributes
    if err := setupDaemonProcess(cmd, config); err != nil {
        return fmt.Errorf("failed to setup daemon process: %w", err)
    }
    
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start daemon: %w", err)
    }
    
    // Save process state
    state := ProcessState{
        PID:        cmd.Process.Pid,
        Port:       parsePort(config.Port),
        StartTime:  time.Now(),
        ConfigPath: "", // Will be set by daemon process
    }
    
    if err := saveProcessState(state); err != nil {
        // Try to kill the process we just started
        cmd.Process.Kill()
        return fmt.Errorf("failed to save process state: %w", err)
    }
    
    // Verify daemon started successfully
    time.Sleep(100 * time.Millisecond) // Give it a moment
    if !isDaemonRunning() {
        return fmt.Errorf("daemon failed to start properly")
    }
    
    fmt.Printf("✓ Daemon started (PID: %d) on port %s\n", state.PID, config.Port)
    return nil
}

func stopDaemon(graceful bool) error {
    state, err := loadProcessState()
    if err != nil {
        return fmt.Errorf("no running daemon found: %w", err)
    }
    
    fmt.Printf("Stopping daemon (PID: %d)...\n", state.PID)
    
    // Try graceful shutdown first
    if err := killProcess(state.PID, true); err != nil {
        return fmt.Errorf("failed to send shutdown signal: %w", err)
    }
    
    // Wait for graceful shutdown
    timeout := 30 * time.Second
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        if !isProcessAlive(state.PID) {
            cleanupProcessState()
            fmt.Println("✓ Daemon stopped gracefully")
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
    
    // Force kill if graceful failed
    if !graceful {
        return fmt.Errorf("daemon did not stop gracefully within %s", timeout)
    }
    
    fmt.Println("Graceful shutdown timeout, forcing termination...")
    if err := killProcess(state.PID, false); err != nil {
        return fmt.Errorf("failed to force kill daemon: %w", err)
    }
    
    cleanupProcessState()
    fmt.Println("✓ Daemon stopped forcefully")
    return nil
}

func setupDaemonProcess(cmd *exec.Cmd, config Config) error {
    // Set up logging
    logPath, err := getLogFilePath()
    if err != nil {
        return err
    }
    
    logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("failed to open log file: %w", err)
    }
    
    cmd.Stdout = logFile
    cmd.Stderr = logFile
    cmd.Stdin = nil
    
    // Platform-specific daemon setup
    if runtime.GOOS != "windows" {
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Setpgid: true,
        }
    }
    
    return nil
}

func checkPortAvailable(port string) error {
    // Try to bind to the port temporarily
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        return err
    }
    listener.Close()
    return nil
}

func parsePort(portStr string) int {
    port, err := strconv.Atoi(portStr)
    if err != nil {
        return 0
    }
    return port
}
```

**Testing Requirements**:
- Test daemon starts successfully and detaches properly
- Test daemon stop with graceful and forceful modes
- Test status reporting works for running and stopped daemons
- Test multiple start attempts are handled correctly

**Dependencies**: D001 (for state management), F002 (for process management)

---

### D003: Logging and Output Management (4 hours)

**Purpose**: Implement logging functions for daemon mode with basic rotation.

**Implementation**:

```go
func setupDaemonLogging(logPath string) error {
    // Ensure log directory exists
    if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
        return fmt.Errorf("failed to create log directory: %w", err)
    }
    
    // Check if log rotation is needed
    if shouldRotateLogs(logPath) {
        if err := rotateLogs(logPath); err != nil {
            return fmt.Errorf("failed to rotate logs: %w", err)
        }
    }
    
    return nil
}

func shouldRotateLogs(logPath string) bool {
    info, err := os.Stat(logPath)
    if err != nil {
        return false // File doesn't exist or can't be accessed
    }
    
    const maxLogSize = 10 * 1024 * 1024 // 10MB
    return info.Size() > maxLogSize
}

func rotateLogs(logPath string) error {
    const maxBackups = 3
    
    // Rotate existing backups: .log.2 -> .log.3, .log.1 -> .log.2, etc.
    for i := maxBackups; i > 1; i-- {
        oldPath := fmt.Sprintf("%s.%d", logPath, i-1)
        newPath := fmt.Sprintf("%s.%d", logPath, i)
        
        if _, err := os.Stat(oldPath); err == nil {
            if err := os.Rename(oldPath, newPath); err != nil {
                return fmt.Errorf("failed to rotate %s to %s: %w", oldPath, newPath, err)
            }
        }
    }
    
    // Move current log to .1
    backupPath := logPath + ".1"
    if err := os.Rename(logPath, backupPath); err != nil {
        return fmt.Errorf("failed to backup current log: %w", err)
    }
    
    return nil
}

func followLogFile(logPath string, follow bool) error {
    file, err := os.Open(logPath)
    if err != nil {
        if os.IsNotExist(err) {
            fmt.Println("No log file found")
            return nil
        }
        return fmt.Errorf("failed to open log file: %w", err)
    }
    defer file.Close()
    
    if !follow {
        // Just dump the file and exit
        _, err := io.Copy(os.Stdout, file)
        return err
    }
    
    // Follow mode - read existing content first
    if _, err := io.Copy(os.Stdout, file); err != nil {
        return fmt.Errorf("failed to read log file: %w", err)
    }
    
    // Then watch for new content (simple polling approach)
    for {
        time.Sleep(100 * time.Millisecond)
        
        // Check if file still exists (in case it was rotated)
        if _, err := os.Stat(logPath); err != nil {
            // Try to reopen the file
            file.Close()
            file, err = os.Open(logPath)
            if err != nil {
                continue
            }
        }
        
        // Read any new content
        if _, err := io.Copy(os.Stdout, file); err != nil {
            break
        }
    }
    
    return nil
}
```

**Testing Requirements**:
- Test log rotation works at size threshold
- Test log following works correctly
- Test log cleanup maintains retention policy
- Test concurrent log writes don't corrupt files

**Dependencies**: D002 (for daemon lifecycle)

---

## Phase 4: Integration (13 hours)

### I001: Server Function Extraction (3 hours)

**Purpose**: Extract server creation logic from main() for reuse in daemon mode.

**Implementation**:

```go
func createHTTPServer(config Config) (*http.Server, error) {
    // Extract existing server creation logic from main()
    mux := http.NewServeMux()
    
    // Add existing handlers (extract from main)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"status": "ok", "version": "dev"}`)
    })
    
    mux.HandleFunc("/v1/messages", handleMessages) // Extract from main
    // ... add other handlers
    
    server := &http.Server{
        Addr:    ":" + config.Port,
        Handler: mux,
    }
    
    return server, nil
}

func startHTTPServer(server *http.Server) error {
    log.Printf("Starting server on %s", server.Addr)
    return server.ListenAndServe()
}

func stopHTTPServer(server *http.Server) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return server.Shutdown(ctx)
}

// Extract from existing main() function
func handleMessages(w http.ResponseWriter, r *http.Request) {
    // Move existing /v1/messages handler logic here
    // This preserves all existing proxy functionality
}
```

**Testing Requirements**:
- Test extracted server functions work identically to original
- Test server functions work in both daemon and direct modes
- Test graceful shutdown works correctly
- Test no regression in proxy functionality

**Dependencies**: D003 (daemon logging)

---

### I002: Configuration Enhancement (2 hours)

**Purpose**: Enhance configuration system for daemon-specific settings.

**Implementation**:

```go
// Extend existing Config struct
type Config struct {
    // Existing fields
    Port        string `json:"port"`
    APIKey      string `json:"api_key"`
    BaseURL     string `json:"base_url"`
    Model       string `json:"model"`
    OpusModel   string `json:"opus_model"`
    SonnetModel string `json:"sonnet_model"`
    HaikuModel  string `json:"haiku_model"`
    
    // New daemon-specific fields
    DaemonMode  bool   `json:"-"` // Runtime flag, not persisted
    LogLevel    string `json:"log_level,omitempty"`
}

func loadConfigForDaemon() (Config, error) {
    // Use existing loadConfig logic but add daemon-specific validation
    config := loadConfig() // Use existing function
    
    // Set daemon-specific defaults
    if config.LogLevel == "" {
        config.LogLevel = "info"
    }
    
    return config, nil
}

func displayConfigStatus(config Config) {
    fmt.Printf("Configuration:\n")
    fmt.Printf("  Port: %s\n", config.Port)
    fmt.Printf("  Base URL: %s\n", config.BaseURL)
    fmt.Printf("  Default Model: %s\n", config.Model)
    if config.LogLevel != "" {
        fmt.Printf("  Log Level: %s\n", config.LogLevel)
    }
}
```

**Testing Requirements**:
- Test enhanced config preserves all existing functionality
- Test daemon-specific configuration works correctly
- Test config validation catches invalid settings
- Test config file discovery works across platforms

**Dependencies**: I001 (server extraction)

---

### I003: Main Function Refactoring (3 hours)

**Purpose**: Refactor main() to serve as command dispatcher while preserving backward compatibility.

**Implementation**:

```go
func main() {
    args := os.Args[1:]
    
    // Check for special daemon mode flag (internal use)
    if len(args) > 0 && args[0] == "--daemon-mode" {
        if err := runDaemonMode(); err != nil {
            log.Fatalf("Daemon mode failed: %v", err)
        }
        return
    }
    
    // Route to appropriate handler
    if err := routeCommand(args); err != nil {
        if strings.Contains(err.Error(), "not yet implemented") {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        // For other errors, just log and exit
        log.Fatalf("Command failed: %v", err)
    }
}

func runDaemonMode() error {
    // This runs the actual daemon process (started by startDaemon)
    config, err := loadConfigForDaemon()
    if err != nil {
        return fmt.Errorf("failed to load daemon config: %w", err)
    }
    
    // Set up daemon logging
    logPath, err := getLogFilePath()
    if err != nil {
        return fmt.Errorf("failed to get log path: %w", err)
    }
    
    if err := setupDaemonLogging(logPath); err != nil {
        return fmt.Errorf("failed to setup logging: %w", err)
    }
    
    // Create and start HTTP server
    server, err := createHTTPServer(config)
    if err != nil {
        return fmt.Errorf("failed to create server: %w", err)
    }
    
    // Set up signal handling for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Println("Received shutdown signal, stopping server...")
        if err := stopHTTPServer(server); err != nil {
            log.Printf("Error stopping server: %v", err)
        }
    }()
    
    // Start server (this blocks until shutdown)
    log.Printf("Daemon starting on port %s", config.Port)
    if err := startHTTPServer(server); err != nil && err != http.ErrServerClosed {
        return fmt.Errorf("server failed: %w", err)
    }
    
    log.Println("Daemon stopped")
    return nil
}

func runLegacyMode(args []string) error {
    // Preserve original main() behavior exactly
    // Parse flags using existing flag parsing logic
    // Load config using existing loadConfig()
    // Create and start server directly (blocking)
    
    // For now, call existing logic (to be extracted)
    return runOriginalMain(args)
}

func runOriginalMain(args []string) error {
    // Move current main() logic here
    // This preserves 100% backward compatibility
    return fmt.Errorf("legacy mode extraction not yet implemented")
}
```

**Testing Requirements**:
- Test main() routes commands to correct handlers
- Test legacy mode works exactly as before refactoring
- Test daemon mode starts and stops correctly
- Test error handling and exit codes are appropriate

**Dependencies**: I002 (configuration), C003 (command routing)

---

### T001: End-to-End Testing (5 hours)

**Purpose**: Create comprehensive integration tests for complete workflows.

**Implementation**:

```go
// Add to main_test.go
func TestDaemonLifecycle(t *testing.T) {
    // Test complete start -> status -> stop workflow
    tests := []struct {
        name string
        steps []string
        expectSuccess bool
    }{
        {
            name: "basic lifecycle",
            steps: []string{"start", "status", "stop"},
            expectSuccess: true,
        },
        {
            name: "double start should fail", 
            steps: []string{"start", "start"},
            expectSuccess: false,
        },
        {
            name: "stop without start should fail",
            steps: []string{"stop"},
            expectSuccess: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set up clean test environment
            setupTestEnv(t)
            defer cleanupTestEnv(t)
            
            var lastErr error
            for _, step := range tt.steps {
                lastErr = runTestCommand(step)
                if lastErr != nil && tt.expectSuccess {
                    t.Fatalf("Step %s failed: %v", step, lastErr)
                }
            }
            
            if tt.expectSuccess && lastErr != nil {
                t.Fatalf("Expected success but got error: %v", lastErr)
            }
        })
    }
}

func TestBackwardCompatibility(t *testing.T) {
    // Test existing flag combinations work exactly as before
    legacyTests := [][]string{
        {},                                    // No args
        {"-port", "9000"},                    // Port flag
        {"-api-key", "test"},                 // API key flag
        {"-port", "9000", "-api-key", "test"}, // Multiple flags
    }
    
    for _, args := range legacyTests {
        t.Run(fmt.Sprintf("legacy_%v", args), func(t *testing.T) {
            // Test that legacy mode detection works
            assert.True(t, isLegacyMode(args))
            
            // Test that routing goes to legacy handler
            // (Implementation will be completed as functions are built)
        })
    }
}

func setupTestEnv(t *testing.T) {
    // Create temporary data directory for testing
    // Set environment variables to point to test directories
}

func cleanupTestEnv(t *testing.T) {
    // Clean up temporary files and processes
    // Ensure no test processes are left running
}

func runTestCommand(command string) error {
    // Helper to run commands in test environment
    return fmt.Errorf("test command execution not yet implemented")
}
```

**Testing Requirements**:
- Test all command workflows work end-to-end
- Test 100% backward compatibility with existing usage
- Test error scenarios provide helpful messages
- Test cross-platform compatibility

**Dependencies**: I003 (main refactoring)

---

## Implementation Notes

### Key Principles

1. **Single File Architecture**: All code remains in main.go with clear organization
2. **No External Dependencies**: Use only Go standard library
3. **Backward Compatibility**: Existing usage must work exactly as before
4. **Function-Based Design**: Simple functions instead of complex entity/service layers
5. **TDD Approach**: Write tests first, then implementation

### Testing Strategy

- **Unit Tests**: Test individual functions in isolation
- **Integration Tests**: Test command workflows end-to-end  
- **Compatibility Tests**: Verify legacy mode works exactly as before
- **Cross-Platform Tests**: Ensure functionality works on Linux, macOS, Windows

### Risk Mitigation

- **Incremental Development**: Implement phase by phase with testing
- **Feature Flags**: Use internal flags to enable/disable new functionality during development
- **Rollback Plan**: Legacy mode ensures instant rollback capability
- **Extensive Testing**: Comprehensive test coverage before deployment

### Success Metrics

- All existing CLI usage patterns work identically to current version
- New subcommands provide intuitive daemon management
- Command response times under 100ms
- Daemon startup under 2 seconds
- Zero regressions in HTTP proxy functionality