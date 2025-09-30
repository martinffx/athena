# Implementation Tasks: CLI Subcommands

## Overview

This document provides a detailed task breakdown for implementing CLI subcommands in openrouter-cc. The implementation follows Test-Driven Development (TDD) with the pattern: **Stub → Test → Implement → Refactor** for each component.

The implementation is structured in 6 phases, introducing 2 new packages (`internal/cli/` and `internal/daemon/`) while maintaining full backward compatibility. Total estimated time: **30 hours**.

## Phase 1: Module Structure & Cobra Integration

**Estimated Time:** 6 hours
**Deliverable:** New package structure, Cobra integrated, backward compatibility preserved

### Task 1.1: Create Package Structure
**Files**:
- `internal/cli/root.go`
- `internal/daemon/daemon.go` (stub)
- `internal/daemon/state.go` (stub)

**Status**: ⬜ Pending

**Steps**:
1. [ ] Create directory structure: `internal/cli/` and `internal/daemon/`
2. [ ] Add Cobra dependency: `go get github.com/spf13/cobra@latest`
3. [ ] Update `go.mod` and verify dependency resolution

**Acceptance Criteria**:
- Directory structure created
- Cobra v1.8.0+ added to go.mod
- No compilation errors

**Estimated Time:** 30 minutes

---

### Task 1.2: Implement Root Command
**File**: `internal/cli/root.go`
**Test**: `internal/cli/root_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: RootCmd with persistent flags (port, config, api-key, base-url, model, opus-model, sonnet-model, haiku-model)
2. [ ] Write tests:
   - Test flag parsing matches current behavior
   - Test default values align with existing config
   - Test config file loading integration
   - Test backward compatibility (no subcommand runs server)
3. [ ] Implement: Complete root command with all persistent flags
4. [ ] Refactor: Extract flag definitions for reusability

**Acceptance Criteria**:
- RootCmd variable exported
- All existing CLI flags defined as persistent flags
- Default behavior (no subcommand) executes server in foreground
- Flag precedence: CLI flags → config files → env vars → defaults
- Tests verify backward compatibility

**Estimated Time:** 3 hours

---

### Task 1.3: Update Main Entry Point
**File**: `cmd/athena/main.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: main() calls cli.Execute()
2. [ ] Write integration test: Binary runs with existing flag patterns
3. [ ] Implement: Refactor main.go to use Cobra CLI
4. [ ] Refactor: Simplify to ~20 lines

**Acceptance Criteria**:
- main() calls cli.Execute()
- `./athena` starts server in foreground (unchanged behavior)
- `./athena -port 9000` works identically to current version
- All existing integration tests pass

**Estimated Time:** 2 hours

---

### Task 1.4: Backward Compatibility Testing
**File**: `internal/cli/compatibility_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write integration tests:
   - Test `./athena` starts server on default port
   - Test `./athena -port 9000` starts server on port 9000
   - Test `./athena -config athena.yml` loads config file
   - Test combined flags: `./athena -port 9000 -api-key test`
2. [ ] Verify all tests pass
3. [ ] Document any edge cases discovered

**Acceptance Criteria**:
- All existing CLI usage patterns tested
- 100% backward compatibility verified
- Tests automated in CI pipeline

**Estimated Time:** 30 minutes

---

## Phase 2: Daemon Package & Start Command

**Estimated Time:** 8 hours
**Deliverable:** Daemon process management with start command functional

### Task 2.1: Daemon State Management
**File**: `internal/daemon/state.go`
**Test**: `internal/daemon/state_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: ProcessState struct, SaveState/LoadState/CleanupState functions
2. [ ] Write tests:
   - Test SaveState creates PID file at `~/.openrouter-cc/openrouter-cc.pid`
   - Test LoadState reads and validates state
   - Test CleanupState removes PID file
   - Test state validation checks process is alive
   - Test concurrent access handling (file locking)
3. [ ] Implement: Complete all functions with proper error handling
4. [ ] Refactor: Extract path resolution, add platform-specific locking

**Acceptance Criteria**:
- ProcessState struct with: PID (int), Port (int), StartTime (time.Time), ConfigPath (string)
- SaveState() writes JSON atomically to `~/.openrouter-cc/openrouter-cc.pid`
- LoadState() reads state and validates PID still running
- CleanupState() safely removes PID file
- File permissions set to 600 (owner read/write only)
- Cross-platform file locking implemented

**Estimated Time:** 3 hours

---

### Task 2.2: Daemon Process Management
**File**: `internal/daemon/daemon.go`
**Test**: `internal/daemon/daemon_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: Start(), Stop(), IsRunning() functions
2. [ ] Write tests:
   - Test Start() forks process and detaches
   - Test Start() returns error if already running
   - Test IsRunning() validates PID existence
   - Test cross-platform process detachment
3. [ ] Implement: Complete daemon lifecycle functions
4. [ ] Refactor: Platform-specific implementations (Linux/macOS vs Windows)

**Acceptance Criteria**:
- Start() forks current process and detaches from terminal
- Process runs with log output redirected to file
- IsRunning() checks PID validity
- Cross-platform support (Linux/macOS/Windows)
- Proper error handling for all failure modes

**Estimated Time:** 3 hours

---

### Task 2.3: Start Command
**File**: `internal/cli/start.go`
**Test**: `internal/cli/start_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: startCmd with RunE function
2. [ ] Write tests:
   - Test start command checks if already running
   - Test start command creates daemon process
   - Test start command saves state file
   - Test start command returns PID on success
   - Test error handling when daemon already running
3. [ ] Implement: Complete start command logic
4. [ ] Refactor: Extract validation and startup logic

**Acceptance Criteria**:
- `openrouter-cc start` creates background daemon
- Daemon PID printed to stdout
- PID file created at `~/.openrouter-cc/openrouter-cc.pid`
- Error if daemon already running on same port
- All persistent flags (port, api-key, etc.) passed to daemon
- Daemon starts successfully within 5 seconds

**Estimated Time:** 2 hours

---

## Phase 3: Stop & Status Commands

**Estimated Time:** 5 hours
**Deliverable:** Process control and monitoring complete

### Task 3.1: Graceful Shutdown Support
**File**: `internal/server/shutdown.go` (extend existing server package)
**Test**: `internal/server/shutdown_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: ListenForShutdown() function
2. [ ] Write tests:
   - Test server responds to SIGTERM signal
   - Test server completes in-flight requests before stopping
   - Test server closes within 30 second timeout
   - Test server force-stops after timeout
3. [ ] Implement: Signal handling and graceful HTTP server shutdown
4. [ ] Refactor: Platform-specific signal handling (Unix vs Windows)

**Acceptance Criteria**:
- Server listens for SIGTERM and SIGINT signals
- http.Server.Shutdown() called with 30s context timeout
- In-flight requests complete before shutdown
- Force shutdown after timeout
- Cross-platform signal support

**Estimated Time:** 2 hours

---

### Task 3.2: Stop Command
**File**: `internal/cli/stop.go`
**Test**: `internal/cli/stop_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: stopCmd with RunE function, --force flag
2. [ ] Write tests:
   - Test stop command reads PID from state file
   - Test stop command sends SIGTERM to process
   - Test stop command waits for graceful shutdown
   - Test --force flag sends SIGKILL immediately
   - Test error handling when daemon not running
   - Test cleanup of PID file after stop
3. [ ] Implement: Complete stop command logic
4. [ ] Refactor: Extract process termination logic

**Acceptance Criteria**:
- `openrouter-cc stop` sends SIGTERM to daemon
- Waits up to 30 seconds for graceful shutdown
- `--force` flag sends SIGKILL immediately
- PID file removed after successful stop
- Helpful error if daemon not running
- Cross-platform signal support

**Estimated Time:** 2 hours

---

### Task 3.3: Status Command
**File**: `internal/cli/status.go`
**Test**: `internal/cli/status_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: statusCmd with RunE function, --json flag
2. [ ] Write tests:
   - Test status command reads state file
   - Test status shows "running" when daemon alive
   - Test status shows "stopped" when daemon not running
   - Test status displays: PID, port, uptime, start time
   - Test --json flag outputs machine-readable format
   - Test error handling for corrupted state file
3. [ ] Implement: Complete status display logic
4. [ ] Refactor: Extract formatting logic (text vs JSON)

**Acceptance Criteria**:
- `openrouter-cc status` shows running/stopped status
- Displays: PID, port, bind address, uptime, start time
- Human-readable output by default
- `--json` flag outputs structured JSON
- Validates PID still exists and is openrouter-cc process
- Returns exit code 0 if running, 1 if stopped

**Estimated Time:** 1 hour

---

## Phase 4: Logs Command

**Estimated Time:** 4 hours
**Deliverable:** Log management and streaming

### Task 4.1: File Logging Support
**File**: `internal/daemon/logging.go`
**Test**: `internal/daemon/logging_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: SetupFileLogging(), RotateLogs() functions
2. [ ] Write tests:
   - Test log output redirected to file in daemon mode
   - Test log rotation at 10MB size
   - Test rotation keeps last 3 log files
   - Test file permissions set to 600
   - Test concurrent write safety
3. [ ] Implement: File logging with rotation
4. [ ] Refactor: Extract rotation policy logic

**Acceptance Criteria**:
- Logs written to `~/.openrouter-cc/openrouter-cc.log` in daemon mode
- Stdout logging in foreground mode (backward compatible)
- Automatic rotation at 10MB
- Keeps athena.log, athena.log.1, athena.log.2
- Thread-safe logging
- File permissions: 600 (owner read/write only)

**Estimated Time:** 2 hours

---

### Task 4.2: Logs Command
**File**: `internal/cli/logs.go`
**Test**: `internal/cli/logs_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: logsCmd with RunE function, --lines and --follow flags
2. [ ] Write tests:
   - Test logs command displays last N lines (default 50)
   - Test --lines flag controls line count
   - Test --follow flag streams new log entries
   - Test handles log rotation gracefully
   - Test exits cleanly on Ctrl+C (SIGINT)
   - Test error handling when log file doesn't exist
3. [ ] Implement: Complete log display and streaming
4. [ ] Refactor: Extract tailing logic, handle file rotation

**Acceptance Criteria**:
- `openrouter-cc logs` displays last 50 lines by default
- `--lines N` displays last N lines
- `--follow` streams new entries in real-time
- Handles log rotation without interruption
- Clean exit on Ctrl+C
- Helpful error if daemon not running or no logs exist

**Estimated Time:** 2 hours

---

## Phase 5: Code Command & Polish

**Estimated Time:** 4 hours
**Deliverable:** Claude Code integration and final testing

### Task 5.1: Code Command
**File**: `internal/cli/code.go`
**Test**: `internal/cli/code_test.go`
**Status**: ⬜ Pending

**TDD Steps**:
1. [ ] Write stub: codeCmd with RunE function
2. [ ] Write tests:
   - Test code command checks daemon is running
   - Test environment variables set correctly
   - Test claude process spawned with inherited environment
   - Test exit code passed through from claude
   - Test error handling when daemon not running
   - Test error handling when claude not in PATH
3. [ ] Implement: Complete Claude Code integration
4. [ ] Refactor: Extract environment setup and process execution

**Acceptance Criteria**:
- `openrouter-cc code` checks daemon status first
- Sets `ANTHROPIC_API_KEY=dummy`
- Sets `ANTHROPIC_BASE_URL=http://localhost:{port}/v1`
- Executes `claude` command with inherited environment
- Returns claude's exit code
- Helpful error if daemon not running
- Helpful error if claude not found in PATH
- Daemon continues running after claude exits

**Estimated Time:** 2 hours

---

### Task 5.2: Error Handling Polish
**Files**: All command files
**Status**: ⬜ Pending

**Steps**:
1. [ ] Review all error messages for clarity
2. [ ] Add helpful suggestions for common failure modes:
   - "Daemon not running. Run 'openrouter-cc start' first."
   - "Port already in use. Check if another instance is running."
   - "Claude not found. Install Claude Code or add to PATH."
3. [ ] Ensure consistent error formatting across commands
4. [ ] Add debug output with --verbose flag

**Acceptance Criteria**:
- All error messages are clear and actionable
- Suggestions provided for common errors
- Consistent error formatting
- --verbose flag shows detailed debugging information

**Estimated Time:** 1 hour

---

### Task 5.3: Documentation Updates
**Files**: `README.md`, `CLAUDE.md`
**Status**: ⬜ Pending

**Steps**:
1. [ ] Update README.md with subcommand examples:
   - Getting started with daemon mode
   - All five subcommands with usage examples
   - Claude Code integration workflow
2. [ ] Add troubleshooting section:
   - Daemon won't start (port in use, permissions)
   - Stop command hangs (force flag usage)
   - Logs command shows nothing (daemon not started in background)
3. [ ] Update CLAUDE.md development commands:
   - Add subcommand testing instructions
   - Update architecture diagrams

**Acceptance Criteria**:
- README.md includes comprehensive subcommand documentation
- Troubleshooting section covers common issues
- CLAUDE.md reflects new architecture
- Examples tested and verified

**Estimated Time:** 1 hour

---

## Phase 6: Cross-Platform Testing & CI

**Estimated Time:** 3 hours
**Deliverable:** Verified cross-platform compatibility

### Task 6.1: Cross-Platform Manual Testing
**Status**: ⬜ Pending

**Steps**:
1. [ ] Test on Linux (Ubuntu 22.04):
   - All subcommands functional
   - Process management works correctly
   - Signal handling (SIGTERM/SIGINT)
   - File paths resolve correctly
2. [ ] Test on macOS (latest):
   - All subcommands functional
   - Process management works correctly
   - Signal handling
   - File paths resolve correctly
3. [ ] Test on Windows (Windows 11):
   - All subcommands functional
   - Process management with CREATE_NEW_PROCESS_GROUP
   - Signal handling (os.Interrupt)
   - File paths with backslashes and home directory

**Acceptance Criteria**:
- All subcommands work identically on Linux, macOS, Windows
- Process management functions correctly on all platforms
- Signal handling appropriate for each platform
- File paths resolve correctly (Unix forward slash vs Windows backslash)

**Estimated Time:** 2 hours

---

### Task 6.2: CI/CD Pipeline Updates
**File**: `.github/workflows/release.yml`, `.github/workflows/test.yml`
**Status**: ⬜ Pending

**Steps**:
1. [ ] Update workflows to install Cobra dependency
2. [ ] Add cross-platform test job:
   - Matrix build for Linux/macOS/Windows
   - Run all unit and integration tests
   - Verify backward compatibility tests
3. [ ] Add binary size check:
   - Ensure Cobra adds <5MB overhead
   - Alert if binary size exceeds threshold
4. [ ] Add performance benchmarks:
   - CLI command response time <100ms
   - Daemon startup time <5s

**Acceptance Criteria**:
- CI runs on all three platforms
- All tests pass on all platforms
- Binary size within acceptable limits
- Performance benchmarks pass
- Backward compatibility verified in CI

**Estimated Time:** 1 hour

---

## Task Dependencies

```
Phase 1 (Module Structure)
├── Task 1.1 (Package Structure) [MUST COMPLETE FIRST]
├── Task 1.2 (Root Command) [depends on 1.1]
├── Task 1.3 (Main Entry) [depends on 1.2]
└── Task 1.4 (Compatibility Tests) [depends on 1.3]

Phase 2 (Daemon & Start)
├── Task 2.1 (State Management) [depends on Phase 1]
├── Task 2.2 (Daemon Process) [depends on 2.1]
└── Task 2.3 (Start Command) [depends on 2.2]

Phase 3 (Stop & Status)
├── Task 3.1 (Graceful Shutdown) [depends on Phase 2]
├── Task 3.2 (Stop Command) [depends on 3.1]
└── Task 3.3 (Status Command) [depends on 2.1]

Phase 4 (Logs)
├── Task 4.1 (File Logging) [depends on Phase 2]
└── Task 4.2 (Logs Command) [depends on 4.1]

Phase 5 (Code & Polish)
├── Task 5.1 (Code Command) [depends on Phase 2]
├── Task 5.2 (Error Handling) [depends on all commands]
└── Task 5.3 (Documentation) [depends on 5.2]

Phase 6 (Testing & CI)
├── Task 6.1 (Manual Testing) [depends on Phase 5]
└── Task 6.2 (CI Updates) [depends on 6.1]
```

## Testing Strategy

### TDD Cycle for Each Task
1. **Stub**: Create minimal function signatures and types
2. **Test**: Write comprehensive unit tests covering success/failure cases
3. **Implement**: Write production code to pass all tests
4. **Refactor**: Improve code quality while maintaining test coverage

### Test Coverage Goals
- Unit test coverage: >90% for new packages (internal/cli/, internal/daemon/)
- Integration test coverage: All user-facing workflows
- Cross-platform tests: Automated via CI for Linux/macOS/Windows

### Critical Test Scenarios
1. **Backward Compatibility**: All existing CLI patterns work unchanged
2. **Daemon Lifecycle**: Full start → status → stop workflow
3. **Concurrent Start**: Multiple start attempts handled gracefully
4. **Process Validation**: PID validation detects stale PID files
5. **Log Rotation**: Rotation doesn't interrupt log streaming
6. **Claude Integration**: Environment variables set correctly
7. **Signal Handling**: Graceful shutdown completes in-flight requests

## Definition of Done

Each task is complete when:
- [ ] All TDD steps completed (Stub → Test → Implement → Refactor)
- [ ] Unit tests written and passing
- [ ] Integration tests written (where applicable) and passing
- [ ] Code reviewed for quality and consistency
- [ ] Documentation updated (inline comments, README if needed)
- [ ] Backward compatibility verified
- [ ] No regression in existing functionality

The feature is complete when:
- [ ] All 6 phases completed
- [ ] All 20 tasks marked complete
- [ ] Cross-platform testing passed
- [ ] CI/CD pipeline updated and green
- [ ] Documentation complete
- [ ] Performance benchmarks met
- [ ] 100% backward compatibility maintained
