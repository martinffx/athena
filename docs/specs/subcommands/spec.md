# CLI Subcommands Feature Specification

**Version:** 1.0.0  
**Created:** 2025-01-08  
**Status:** Ready for Implementation

## Feature Overview

### User Story
**As a** developer using openrouter-cc proxy  
**I want to** control the proxy server with CLI subcommands (start, stop, status, logs, code)  
**So that I can** manage the proxy as a background service and integrate seamlessly with Claude Code workflows

### Business Context
OpenRouter CC currently operates as a foreground HTTP proxy server. Users must manage the process manually using shell job control (`&`, `kill`, `ps`). This feature adds proper daemon management capabilities with five subcommands for process control, monitoring, and Claude Code integration.

## Detailed Acceptance Criteria

### AC-01: Start Command Creates Background Daemon
**GIVEN** the proxy is not currently running  
**WHEN** user runs `openrouter-cc start`  
**THEN:**
- Proxy starts as a background daemon process within 5 seconds
- PID file is created at `~/.openrouter-cc/openrouter-cc.pid`
- Log file is created at `~/.openrouter-cc/openrouter-cc.log`
- Command returns success exit code and daemon PID
- HTTP server becomes available on configured port

### AC-02: Stop Command Gracefully Terminates Daemon
**GIVEN** the proxy daemon is running  
**WHEN** user runs `openrouter-cc stop`  
**THEN:**
- Running proxy receives SIGTERM signal
- Proxy completes in-flight requests before stopping
- PID file is removed from `~/.openrouter-cc/`
- Process terminates within 30 seconds
- Command returns success exit code

### AC-03: Status Command Shows Daemon Information
**GIVEN** user wants to check proxy state  
**WHEN** user runs `openrouter-cc status`  
**THEN:**
- Shows running/stopped status
- Displays PID if running
- Shows port number and bind address
- Shows uptime duration if running
- Shows last startup time from log file

### AC-04: Logs Command Streams Real-Time Output
**GIVEN** the proxy daemon is generating logs  
**WHEN** user runs `openrouter-cc logs`  
**THEN:**
- Displays existing log content
- Streams new log entries in real-time
- Handles log rotation gracefully
- Exits cleanly on Ctrl+C

### AC-05: Code Command Launches Claude Code with Environment
**GIVEN** the proxy daemon is running  
**WHEN** user runs `openrouter-cc code`  
**THEN:**
- Sets `ANTHROPIC_API_KEY` environment variable to 'dummy'
- Sets `ANTHROPIC_BASE_URL` to proxy's local address
- Launches 'claude' command with inherited environment
- Proxy continues running in background
- Returns Claude Code's exit status

## Business Rules and Constraints

### BR-01: Process Management
Only one proxy instance can run per port to prevent port conflicts and ensure predictable behavior.

### BR-02: File Management
PID and log files stored in `~/.openrouter-cc/` directory following XDG Base Directory standards for user data.

### BR-03: Log Management
Log files rotate when they reach 10MB size to prevent unlimited disk space usage while preserving recent history.

### BR-04: Signal Handling
Daemon must handle SIGTERM for graceful shutdown, allowing clean termination of in-flight requests and resource cleanup.

### BR-05: Backward Compatibility
All existing CLI flags must remain functional to maintain compatibility with existing scripts and workflows.

### BR-06: Cross-Platform Support
All subcommands must work on Linux, macOS, and Windows to maintain existing platform support without regression.

## Scope Definition

### Included Features
- Cobra CLI framework integration
- Five subcommands: start, stop, status, logs, code
- Daemon process management with PID file tracking
- Log file management with 10MB rotation
- Cross-platform signal handling (SIGTERM/SIGINT)
- Environment variable setup for Claude Code integration
- Migration of existing flag-based CLI to Cobra structure
- Backward compatibility for all current CLI flags

### Excluded Features
- Breaking changes to existing HTTP API endpoints
- External dependencies beyond Cobra framework
- Persistent state storage beyond PID and log files
- Authentication or authorization for CLI commands
- Remote daemon control or multi-host management
- Service installation (systemd, launchd, Windows services)
- Configuration file changes or new config options

## Technical Architecture

### Architecture Changes

#### CLI Structure
**Current:** Single main() with flag.StringVar() calls  
**Proposed:** Cobra root command with subcommands, maintaining flag compatibility  
**Impact:** Major refactoring of CLI initialization code

#### Process Management
**Current:** Synchronous HTTP server startup  
**Proposed:** Daemon mode with PID file tracking and signal handling  
**Impact:** Need process forking/detachment logic for each platform

#### Logging
**Current:** Standard output logging  
**Proposed:** File-based logging with rotation in daemon mode  
**Impact:** Implement log rotation and file management

### Platform-Specific Considerations

#### Linux/macOS
- Use os.Signal with syscall.SIGTERM/SIGINT
- Fork process using os/exec for daemon mode
- PID file locking with flock system call

#### Windows
- Use os.Interrupt for graceful shutdown
- Use os/exec.Command with CREATE_NEW_PROCESS_GROUP
- File locking using Windows file attributes

### Performance Impact
- **Memory Usage:** Cobra adds ~2MB to binary size (acceptable for single binary distribution)
- **Startup Time:** Additional CLI parsing overhead ~10ms (negligible compared to HTTP server startup)

### Security Considerations
- **PID File Security:** Validate PID exists and belongs to openrouter-cc process
- **Log File Access:** Set restrictive file permissions (600) on log files to prevent sensitive data exposure

## Dependencies

### Internal Dependencies
1. **Configuration System:** Existing multi-source config loading must integrate with Cobra while maintaining flag precedence: CLI flags → config files → env vars → defaults
2. **HTTP Server:** Current server implementation must support daemon mode, requiring main() refactoring for foreground/background modes
3. **Request/Response Handling:** Core proxy functionality remains unchanged with no modifications to transformation logic

### External Dependencies
1. **github.com/spf13/cobra ^1.8.0:** Industry-standard CLI framework for Go applications (first external dependency, approved in scope)
2. **Claude Code CLI:** Target integration platform that must be available in PATH (code subcommand fails gracefully if not found)

## Backward Compatibility

### Requirements
- **CLI Flags:** All existing flags (-config, -port, -api-key, etc.) must work identically via Cobra persistent flags
- **Configuration Files:** Existing config file formats and locations must continue working unchanged
- **HTTP API:** All existing API endpoints must remain unchanged with core proxy functionality isolated
- **Binary Behavior:** Default behavior (no subcommand) should start server normally in foreground mode

### Migration Path
- **Current:** `./openrouter-cc -port 9000` → **New:** `./openrouter-cc -port 9000` (unchanged default behavior)
- **Current:** Background via shell `&` → **New:** `./openrouter-cc start -port 9000` (improved daemon management)

## Implementation Phases

### Phase 1: Cobra Integration
**Deliverable:** CLI structure refactored with existing behavior preserved
- Add Cobra dependency to go.mod
- Create root command structure
- Migrate existing flags to Cobra persistent flags
- Maintain backward compatibility for direct execution

### Phase 2: Basic Process Management
**Deliverable:** Start/stop subcommands functional on primary platform
- Implement start subcommand with daemon mode
- Implement stop subcommand with graceful shutdown
- Add PID file management
- Create ~/.openrouter-cc directory structure

### Phase 3: Status and Logging
**Deliverable:** Complete process monitoring and logging capabilities
- Implement status subcommand with process information
- Add file-based logging for daemon mode
- Implement logs subcommand with real-time tailing
- Add log rotation at 10MB threshold

### Phase 4: Claude Code Integration
**Deliverable:** Seamless Claude Code integration
- Implement code subcommand
- Set up environment variables for Claude Code
- Handle Claude Code process lifecycle
- Add error handling for missing claude command

### Phase 5: Cross-Platform Support
**Deliverable:** Full cross-platform compatibility verified
- Test and fix Windows-specific process management
- Verify signal handling across all platforms
- Test PID file locking on different filesystems
- Update build pipeline for new dependency

## Testing Requirements

### Unit Tests
- Cobra command parsing and flag mapping
- PID file creation, validation, and cleanup
- Log rotation logic and file management
- Signal handling for graceful shutdown
- Environment variable setup for Claude Code

### Integration Tests
- End-to-end daemon lifecycle (start → status → stop)
- Log streaming and real-time tailing functionality
- Claude Code process spawning and environment inheritance
- Cross-platform process management behavior
- Backward compatibility for existing CLI patterns

### Performance Tests
- CLI command response time benchmarks
- Daemon startup time validation
- Memory usage impact measurement
- HTTP proxy performance regression testing

## Success Metrics

1. **Backward Compatibility:** 100% of existing CLI usage patterns continue working (measured via automated tests)
2. **Daemon Reliability:** Daemon starts/stops successfully 99.9% of the time (measured via automated process management tests)
3. **Performance Impact:** CLI command response time <100ms, daemon startup <5s (measured via benchmark tests)
4. **Cross-Platform Support:** All subcommands work identically on Linux, macOS, Windows (measured via CI pipeline testing)
5. **Claude Code Integration:** Code subcommand successfully launches Claude Code with proper environment (measured via integration tests)

## Risk Assessment and Mitigation

### High Probability Risks
- **Breaking Zero-Dependency Principle** (Medium Impact): Cobra is explicitly approved in scope and widely used in Go ecosystem

### Medium Probability Risks
- **Cross-Platform Process Management Complexity** (High Impact): Implement platform-specific process handling with extensive testing
- **Backward Compatibility Issues** (High Impact): Comprehensive testing of existing CLI patterns and configuration loading

### Low Probability Risks
- **PID File Race Conditions** (Medium Impact): Implement proper file locking and PID validation

## Definition of Done

- [ ] All five subcommands (start, stop, status, logs, code) implemented and tested
- [ ] Backward compatibility maintained for all existing CLI flags and behavior
- [ ] Cross-platform functionality verified on Linux, macOS, and Windows
- [ ] Comprehensive test suite with >90% code coverage
- [ ] Performance benchmarks meet specified targets
- [ ] Documentation updated with subcommand usage examples
- [ ] CI/CD pipeline updated for Cobra dependency and cross-platform testing