# CLI Subcommands Implementation Status

This document tracks the implementation progress of CLI subcommands for openrouter-cc using a simplified function-based approach.

## Project Overview

**Implementation Approach**: Function-based architecture within single main.go file  
**Total Estimated Effort**: 44 hours across 4 phases  
**Dependencies**: Go standard library only  
**Compatibility**: 100% backward compatibility required

## Progress Summary

**Overall Progress**: 0% (0/15 tasks completed)  
**Phase 1**: 0% (0/3 tasks completed) - Foundation Functions  
**Phase 2**: 0% (0/3 tasks completed) - CLI Functions  
**Phase 3**: 0% (0/3 tasks completed) - Daemon Functions  
**Phase 4**: 0% (0/6 tasks completed) - Integration & Testing

---

## Phase 1: Foundation Functions (9 hours total)

### F001: Data Directory Management Functions
- **Status**: ⏳ Not Started
- **Effort**: 2 hours
- **Dependencies**: None
- **Deliverables**:
  - [ ] getDataDir() function with cross-platform home directory detection
  - [ ] ensureDataDir() function with proper permissions (0755)
  - [ ] getPidFilePath() and getLogFilePath() utility functions
  - [ ] Unit tests for all directory management functions

**Key Implementation Notes**:
- Use os.UserHomeDir() and filepath.Join() for cross-platform compatibility
- Create ~/.openrouter-cc/ directory with proper permissions
- Handle permission denied scenarios gracefully

**Testing Requirements**:
- [ ] Test directory creation across platforms
- [ ] Test permission handling for directories  
- [ ] Test path resolution works correctly
- [ ] Test error handling for permission denied scenarios

### F002: Process Management Utility Functions
- **Status**: ⏳ Not Started  
- **Effort**: 3 hours
- **Dependencies**: None
- **Deliverables**:
  - [ ] isProcessAlive(pid int) bool function with platform-specific implementations
  - [ ] getCurrentProcess() ProcessInfo function
  - [ ] killProcess(pid int, graceful bool) error function
  - [ ] Unit tests for process management utilities

**Key Implementation Notes**:
- Unix: Use kill(pid, 0) for detection, SIGTERM/SIGKILL for termination
- Windows: Use OpenProcess API and TerminateProcess
- Use build tags or runtime.GOOS for platform-specific code

**Testing Requirements**:
- [ ] Test process detection with current process (should return true)
- [ ] Test process detection with non-existent PID (should return false)
- [ ] Test graceful vs forceful process termination
- [ ] Test cross-platform compatibility

### F003: File Operations and Locking Functions
- **Status**: ⏳ Not Started
- **Effort**: 4 hours  
- **Dependencies**: F001
- **Deliverables**:
  - [ ] writeJSONFile(path string, data interface{}) error with atomic writes
  - [ ] readJSONFile(path string, data interface{}) error with validation
  - [ ] lockFile(path string) (unlock func(), error) for exclusive access
  - [ ] Unit tests for file operations with concurrent access scenarios

**Key Implementation Notes**:
- Use temp file + atomic rename pattern for writes
- Implement simple mutex-based locking (can enhance with flock later)
- Handle concurrent file access properly

**Testing Requirements**:
- [ ] Test atomic writes don't leave partial files on error
- [ ] Test concurrent file access is properly serialized
- [ ] Test file locking prevents race conditions
- [ ] Test error handling for corrupt JSON files

---

## Phase 2: CLI Functions (9 hours total)

### C001: Command-line Argument Detection
- **Status**: ⏳ Not Started
- **Effort**: 2 hours
- **Dependencies**: None
- **Deliverables**:
  - [ ] isLegacyMode(args []string) bool function
  - [ ] parseCommand(args []string) (command, flags, error) function
  - [ ] Command validation and help text generation functions
  - [ ] Unit tests for all argument parsing scenarios

**Key Implementation Notes**:
- No external dependencies (no Cobra) - use standard library only
- Legacy mode: no args, first arg starts with -, or unknown command
- Simple flag parsing for --flag=value and --flag value patterns

**Testing Requirements**:
- [ ] Test legacy mode detection with various argument patterns
- [ ] Test command parsing handles flags correctly
- [ ] Test help text generation works for all commands
- [ ] Test error handling for invalid command combinations

### C002: Command Handler Functions  
- **Status**: ⏳ Not Started
- **Effort**: 5 hours
- **Dependencies**: C001
- **Deliverables**:
  - [ ] handleStartCommand(args []string) error function
  - [ ] handleStopCommand(args []string) error function
  - [ ] handleStatusCommand(args []string) error function
  - [ ] handleLogsCommand(args []string) error function
  - [ ] handleCodeCommand(args []string) error function
  - [ ] Unit tests for each command handler

**Key Implementation Notes**:
- Each handler should be self-contained with clear error messages
- Implement help text for each command (--help flag)
- Start with basic structure, full implementation in daemon phase

**Testing Requirements**:
- [ ] Test each command handler with valid arguments
- [ ] Test error handling for invalid arguments
- [ ] Test help text is displayed correctly for each command
- [ ] Test command handlers provide appropriate user feedback

### C003: Command Routing and Execution
- **Status**: ⏳ Not Started
- **Effort**: 2 hours
- **Dependencies**: C002
- **Deliverables**:
  - [ ] routeCommand(args []string) error function as main dispatcher
  - [ ] displayHelp() and displayVersion() utility functions
  - [ ] Command completion and suggestion functions
  - [ ] Integration tests for command routing

**Key Implementation Notes**:
- Keep routing logic simple and maintainable within main.go
- Route unknown commands to helpful error messages
- Preserve legacy mode routing

**Testing Requirements**:
- [ ] Test command routing dispatches to correct handlers
- [ ] Test unknown commands display helpful error messages
- [ ] Test help and version commands work correctly
- [ ] Test legacy mode routing preserves existing behavior

---

## Phase 3: Daemon Functions (13 hours total)

### D001: Process State Management Functions
- **Status**: ⏳ Not Started
- **Effort**: 3 hours
- **Dependencies**: F003
- **Deliverables**:
  - [ ] ProcessState struct definition
  - [ ] saveProcessState(state ProcessState) error function  
  - [ ] loadProcessState() (ProcessState, error) function
  - [ ] cleanupProcessState() error function
  - [ ] Unit tests for state management

**Key Implementation Notes**:
- Simple ProcessState struct: PID, Port, StartTime, ConfigPath
- State validation checks if process is still alive
- Automatic cleanup of stale state files

**Testing Requirements**:
- [ ] Test state save/load round-trip works correctly
- [ ] Test state validation detects stale processes
- [ ] Test cleanup removes only stale files
- [ ] Test error handling for corrupt state files

### D002: Daemon Lifecycle Functions
- **Status**: ⏳ Not Started  
- **Effort**: 6 hours
- **Dependencies**: D001, F002
- **Deliverables**:
  - [ ] startDaemon(config Config) error function
  - [ ] stopDaemon(graceful bool) error function
  - [ ] getDaemonStatus() (DaemonStatus, error) function
  - [ ] isDaemonRunning() bool function
  - [ ] Integration tests for daemon lifecycle

**Key Implementation Notes**:
- Cross-platform daemon process creation with proper detachment
- Graceful shutdown with timeout (30s) and force kill fallback
- Process group management for proper signal handling

**Testing Requirements**:
- [ ] Test daemon starts successfully and detaches properly
- [ ] Test daemon stop with graceful and forceful modes
- [ ] Test status reporting works for running and stopped daemons
- [ ] Test multiple start attempts are handled correctly

### D003: Logging and Output Management
- **Status**: ⏳ Not Started
- **Effort**: 4 hours
- **Dependencies**: D002
- **Deliverables**:
  - [ ] setupDaemonLogging(logPath string) error function
  - [ ] rotateLogs(logPath string) error function
  - [ ] followLogFile(logPath string, follow bool) error function
  - [ ] Log rotation and cleanup utilities
  - [ ] Unit tests for logging functionality

**Key Implementation Notes**:
- Simple log rotation: 10MB threshold, keep 3 files
- Log following with basic polling approach
- Concurrent write safety

**Testing Requirements**:
- [ ] Test log rotation works at size threshold
- [ ] Test log following works correctly
- [ ] Test log cleanup maintains retention policy
- [ ] Test concurrent log writes don't corrupt files

---

## Phase 4: Integration & Testing (13 hours total)

### I001: Server Function Extraction
- **Status**: ⏳ Not Started
- **Effort**: 3 hours
- **Dependencies**: D003
- **Deliverables**:
  - [ ] createHTTPServer(config Config) (*http.Server, error) function
  - [ ] startHTTPServer(server *http.Server) error function
  - [ ] stopHTTPServer(server *http.Server) error function
  - [ ] Refactored main() function to use extracted functions
  - [ ] Unit tests for server function extraction

**Key Implementation Notes**:
- Carefully extract existing server logic without changing behavior
- Enable reuse in both daemon and direct modes
- Preserve all existing proxy functionality

**Testing Requirements**:
- [ ] Test extracted server functions work identically to original
- [ ] Test server functions work in both daemon and direct modes
- [ ] Test graceful shutdown works correctly
- [ ] Test no regression in proxy functionality

### I002: Configuration Enhancement
- **Status**: ⏳ Not Started
- **Effort**: 2 hours
- **Dependencies**: I001
- **Deliverables**:
  - [ ] Enhanced Config struct with daemon fields
  - [ ] loadConfigForDaemon() function with daemon-specific validation
  - [ ] Configuration display functions for status command
  - [ ] Backward compatibility preservation for all existing config options

**Key Implementation Notes**:
- Extend existing Config struct carefully to avoid breaking changes
- Add daemon-specific settings (LogLevel, etc.)
- Preserve all current configuration behavior

**Testing Requirements**:
- [ ] Test enhanced config preserves all existing functionality
- [ ] Test daemon-specific configuration works correctly
- [ ] Test config validation catches invalid settings
- [ ] Test config file discovery works across platforms

### I003: Main Function Refactoring
- **Status**: ⏳ Not Started
- **Effort**: 3 hours
- **Dependencies**: I002, C003
- **Deliverables**:
  - [ ] Refactored main() function as simple command dispatcher
  - [ ] Preserved legacy mode behavior exactly as before
  - [ ] Clean integration of all command handlers
  - [ ] Proper error handling and exit codes

**Key Implementation Notes**:
- This is the final integration step - ensure seamless operation
- main() becomes simple router: legacy mode vs command dispatcher
- Add --daemon-mode internal flag for actual daemon process

**Testing Requirements**:
- [ ] Test main() routes commands to correct handlers
- [ ] Test legacy mode works exactly as before refactoring
- [ ] Test all new commands work through main() dispatcher
- [ ] Test error handling and exit codes are appropriate

### T001: End-to-End Testing
- **Status**: ⏳ Not Started  
- **Effort**: 5 hours
- **Dependencies**: I003
- **Deliverables**:
  - [ ] Full workflow tests (start → status → stop)
  - [ ] Backward compatibility test suite with extensive flag combinations
  - [ ] Error scenario testing (port conflicts, permissions, etc.)
  - [ ] Cross-platform integration validation

**Key Implementation Notes**:
- Use table-driven tests and temporary directories for isolation
- Test all error scenarios with helpful error messages
- Validate cross-platform compatibility thoroughly

**Testing Requirements**:
- [ ] Test all command workflows work end-to-end
- [ ] Test 100% backward compatibility with existing usage
- [ ] Test error scenarios provide helpful messages  
- [ ] Test cross-platform compatibility on Linux, macOS, Windows

---

## Implementation Notes

### Current Blockers
- None identified - ready to begin Phase 1 implementation

### Risk Assessment
- **Low Risk**: Function-based approach is simpler than original entity/service design
- **Medium Risk**: Cross-platform compatibility requires careful testing
- **Mitigation**: Incremental implementation with extensive testing at each phase

### Next Steps
1. Start with F001: Data Directory Management Functions
2. Implement comprehensive unit tests for each function
3. Follow TDD approach: write tests first, then implementation
4. Maintain backward compatibility throughout

### Quality Gates
- [ ] All unit tests pass with >90% coverage
- [ ] Code follows existing single-file organization patterns  
- [ ] TDD workflow followed (test first, then implementation)
- [ ] Functions are self-contained with clear interfaces
- [ ] Error handling provides helpful user feedback
- [ ] Platform-specific code uses appropriate abstractions

### Success Criteria Tracking

#### Functionality
- [ ] All existing CLI usage works exactly as before
- [ ] All new subcommands work reliably across platforms
- [ ] Daemon lifecycle management works correctly
- [ ] State persistence survives system operations

#### Performance  
- [ ] Command response time <100ms (95th percentile)
- [ ] Daemon startup time <2 seconds
- [ ] Memory overhead <500KB for daemon management
- [ ] No regression in proxy performance

#### Reliability
- [ ] Zero data loss during daemon operations
- [ ] Graceful handling of all error conditions  
- [ ] Proper resource cleanup on exit
- [ ] Stable across system restarts

#### Usability
- [ ] Intuitive command structure
- [ ] Clear error messages with guidance
- [ ] Consistent output formatting
- [ ] Seamless legacy mode compatibility

---

*Status Updated*: 2025-01-08  
*Next Review*: After Phase 1 completion