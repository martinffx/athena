# Technical Design: CLI Subcommands

## Architecture Overview
Enhanced CLI interface with subcommands while maintaining single-file Go architecture and zero-dependency design. The existing HTTP proxy functionality remains unchanged - this adds a command layer on top.

## Domain Model

### Command Entities
- **CommandRouter**: Dispatches subcommands and maintains backward compatibility
- **ProcessManager**: Handles daemon lifecycle (start/stop/status/restart)  
- **StateManager**: Manages PID files, logs, and runtime state
- **ConfigValidator**: Validates and displays configuration information

### Process States
```go
type DaemonStatus int
const (
    StatusStopped DaemonStatus = iota
    StatusStarting
    StatusRunning 
    StatusStopping
    StatusError
)
```

## Data Persistence

### File System Layout
```
~/.local/share/openrouter-cc/          # Linux/macOS
%APPDATA%/openrouter-cc/               # Windows
├── openrouter-cc.pid                  # Process ID file
├── openrouter-cc.log                  # Daemon logs (rotating)
└── openrouter-cc.sock                 # Unix socket (future use)
```

### State Files
- **PID File**: Single integer, written atomically on daemon start
- **Log File**: Structured logs with timestamps, max 10MB with rotation
- **No Database**: All state is ephemeral or file-based

## API Specification

### CLI Interface
```bash
# Daemon management
openrouter-cc start [flags]           # Start daemon in background
openrouter-cc stop                    # Stop running daemon
openrouter-cc restart [flags]         # Stop and start daemon
openrouter-cc status                  # Show daemon status and health

# Configuration management  
openrouter-cc config validate        # Validate current configuration
openrouter-cc config show            # Display effective configuration
openrouter-cc models list            # Show available model mappings

# Utility commands
openrouter-cc health                  # Check proxy health (if running)
openrouter-cc logs                    # Tail daemon logs
openrouter-cc version                 # Show version information

# Backward compatibility (no subcommand)
openrouter-cc [flags]                 # Direct server mode (current behavior)
```

### Exit Codes
- `0`: Success
- `1`: General error 
- `2`: Process not running (for stop/status)
- `3`: Process already running (for start)
- `4`: Configuration error
- `5`: Permission error

## Components

### Router: Command Dispatch Layer
```go
type Command struct {
    Name        string
    Description string
    Handler     func(args []string) error
    Flags       *flag.FlagSet
}

func routeCommand(args []string) error {
    if len(args) == 0 || args[0][0] == '-' {
        // Backward compatibility: direct server mode
        return runServerMode(args)
    }
    
    cmd := args[0]
    switch cmd {
    case "start":
        return handleStart(args[1:])
    case "stop": 
        return handleStop(args[1:])
    // ... other commands
    }
}
```

### Service: Process Management
```go
type ProcessManager struct {
    pidFile    string
    logFile    string
    configPath string
}

func (pm *ProcessManager) Start(config Config) error {
    // Check if already running
    if pm.IsRunning() {
        return ErrAlreadyRunning
    }
    
    // Start daemon process
    cmd := exec.Command(os.Args[0], "--daemon", "--config", pm.configPath)
    cmd.Stdout = logWriter
    cmd.Stderr = logWriter
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // Write PID file
    return pm.writePIDFile(cmd.Process.Pid)
}

func (pm *ProcessManager) Stop() error {
    pid, err := pm.readPIDFile()
    if err != nil {
        return ErrNotRunning
    }
    
    // Send SIGTERM (Unix) or equivalent (Windows)
    return pm.terminateProcess(pid)
}
```

### Repository: State Persistence
```go
type StateManager struct {
    dataDir string
}

func (sm *StateManager) WritePIDFile(pid int) error {
    pidFile := filepath.Join(sm.dataDir, "openrouter-cc.pid")
    data := fmt.Sprintf("%d\n", pid)
    
    // Atomic write using temp file + rename
    return atomicWrite(pidFile, []byte(data), 0644)
}

func (sm *StateManager) ReadPIDFile() (int, error) {
    pidFile := filepath.Join(sm.dataDir, "openrouter-cc.pid")
    data, err := os.ReadFile(pidFile)
    if err != nil {
        return 0, err
    }
    
    return strconv.Atoi(strings.TrimSpace(string(data)))
}
```

### Entity: Configuration and Validation
```go
type ConfigEntity struct {
    Config
    ValidationErrors []string
}

func (ce *ConfigEntity) Validate() bool {
    ce.ValidationErrors = nil
    
    if ce.APIKey == "" {
        ce.ValidationErrors = append(ce.ValidationErrors, "API key is required")
    }
    
    if _, err := url.Parse(ce.BaseURL); err != nil {
        ce.ValidationErrors = append(ce.ValidationErrors, "Invalid base URL")
    }
    
    return len(ce.ValidationErrors) == 0
}

func (ce *ConfigEntity) Display() string {
    var buf bytes.Buffer
    fmt.Fprintf(&buf, "Configuration:\n")
    fmt.Fprintf(&buf, "  Port: %s\n", ce.Port)
    fmt.Fprintf(&buf, "  Base URL: %s\n", ce.BaseURL)
    fmt.Fprintf(&buf, "  API Key: %s\n", maskAPIKey(ce.APIKey))
    // ... other fields
    return buf.String()
}
```

## Platform-Specific Implementations

### Unix (Linux/macOS)
```go
//go:build unix

func (pm *ProcessManager) terminateProcess(pid int) error {
    process, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    
    // Send SIGTERM
    if err := process.Signal(syscall.SIGTERM); err != nil {
        return err
    }
    
    // Wait up to 10 seconds for graceful shutdown
    for i := 0; i < 100; i++ {
        if !pm.processExists(pid) {
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
    
    // Force kill if still running
    return process.Signal(syscall.SIGKILL)
}
```

### Windows
```go
//go:build windows

func (pm *ProcessManager) terminateProcess(pid int) error {
    // Use taskkill command for clean termination
    cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T")
    return cmd.Run()
}
```

## Integration with Existing Architecture

### Modified main() Function
```go
func main() {
    args := os.Args[1:]
    
    // Route to command handler
    if err := routeCommand(args); err != nil {
        log.Fatal(err)
    }
}

func runServerMode(args []string) error {
    // Existing main() logic moved here
    flag.Parse()
    loadConfig("")
    
    // Add daemon mode flag handling
    if *daemonFlag {
        setupLogging()
        setupSignalHandlers()
    }
    
    // Existing HTTP server startup
    return startHTTPServer()
}
```

### Signal Handling for Daemon Mode
```go
func setupSignalHandlers() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        log.Println("Received shutdown signal")
        cleanupAndExit()
    }()
}

func cleanupAndExit() {
    // Remove PID file
    sm := newStateManager()
    sm.RemovePIDFile()
    
    // Close HTTP server gracefully
    // (existing cleanup logic)
    
    os.Exit(0)
}
```

## Events

### Process Lifecycle Events
- `daemon.starting`: Before daemon process creation
- `daemon.started`: After successful daemon startup  
- `daemon.stopping`: Before termination signal sent
- `daemon.stopped`: After process termination confirmed
- `daemon.error`: On process management errors

### Configuration Events  
- `config.loaded`: After configuration parsing
- `config.validated`: After validation checks
- `config.error`: On configuration errors

## Dependencies

### External Dependencies
- **None**: Maintains zero-dependency design using only Go standard library
- `os/exec`: Process management
- `syscall`: Signal handling (Unix)
- `filepath`: Cross-platform path handling

### Internal Dependencies
- **Configuration System**: Reuse existing `Config` struct and loading logic
- **HTTP Server**: Reuse existing server and transformation code
- **Logging**: Extend existing log usage for daemon mode

## Backward Compatibility

### Existing Behavior Preserved
- Direct invocation: `openrouter-cc [flags]` continues to work
- All existing flags maintained with same behavior
- Wrapper script (`openrouter`) continues to function
- Configuration loading logic unchanged

### Migration Path
- No breaking changes to existing usage
- New subcommands are additive functionality
- Users can adopt subcommands incrementally

## Implementation Phases

### Phase 1: Core Subcommand Infrastructure
- Command routing and dispatch
- Basic `start`, `stop`, `status` commands
- PID file management
- Platform-specific process handling

### Phase 2: Enhanced Management
- `restart`, `logs`, `health` commands
- Configuration validation and display  
- Proper daemon logging and rotation

### Phase 3: Advanced Features
- `models list` with OpenRouter API integration
- Socket-based communication for status
- Enhanced error reporting and diagnostics