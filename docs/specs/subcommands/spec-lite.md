# CLI Subcommands - Lite Specification

**Feature:** cli-subcommands | **Version:** 1.0.0 | **Status:** Ready for Implementation

## Core Functionality

Add five CLI subcommands to transform openrouter-cc from a foreground HTTP proxy into a managed background service with Claude Code integration:

- **`start`** - Launch proxy as background daemon with PID tracking
- **`stop`** - Gracefully terminate daemon process  
- **`status`** - Show daemon state, PID, uptime, and configuration
- **`logs`** - Stream real-time log output with rotation support
- **`code`** - Launch Claude Code with proper environment variables

## Key Requirements

### Process Management
- Single daemon instance per port with PID file at `~/.openrouter-cc/openrouter-cc.pid`
- Graceful shutdown handling in-flight requests within 30 seconds
- Cross-platform signal handling (SIGTERM/SIGINT on Unix, os.Interrupt on Windows)

### Logging System
- File-based logging at `~/.openrouter-cc/openrouter-cc.log` for daemon mode
- 10MB log rotation with real-time streaming capability
- Restrictive file permissions (600) for security

### Claude Code Integration
- Environment setup: `ANTHROPIC_API_KEY=dummy`, `ANTHROPIC_BASE_URL=http://localhost:{port}`
- Process lifecycle management with inherited environment
- Graceful fallback if `claude` command not found in PATH

### Backward Compatibility
- All existing CLI flags (`-config`, `-port`, `-api-key`, etc.) must work identically
- Default behavior (no subcommand) starts server in foreground mode unchanged
- No changes to HTTP API endpoints or configuration file formats

## Technical Implementation

### Architecture Changes
- **CLI Framework:** Migrate from stdlib `flag` to `github.com/spf13/cobra ^1.8.0`
- **Process Model:** Add daemon mode with fork/detach logic per platform
- **Logging:** Switch from stdout to file-based logging in daemon mode

### Platform-Specific Handling
- **Unix (Linux/macOS):** Use `os/exec` forking, `flock` for PID file locking, `syscall.SIGTERM`
- **Windows:** Use `CREATE_NEW_PROCESS_GROUP`, file attributes for locking, `os.Interrupt`

### File Structure
```
~/.openrouter-cc/
├── openrouter-cc.pid    # Process ID tracking
└── openrouter-cc.log    # Daemon logs with rotation
```

## Dependencies

### Internal
- Configuration system must integrate with Cobra while preserving precedence order
- HTTP server requires refactoring to support both foreground/daemon modes
- Core proxy logic remains unchanged

### External
- **Cobra CLI Framework:** First external dependency, approved for scope
- **Claude Code:** Optional integration target, fails gracefully if unavailable

## Implementation Phases

1. **Cobra Migration** - Refactor CLI structure while maintaining compatibility
2. **Process Management** - Add start/stop with PID tracking
3. **Monitoring** - Implement status and logs subcommands
4. **Integration** - Add Claude Code environment setup
5. **Cross-Platform** - Test and verify all platforms

## Success Criteria

- 100% backward compatibility for existing CLI usage patterns
- Daemon reliability >99.9% for start/stop operations
- CLI responsiveness <100ms, daemon startup <5s
- Cross-platform functionality on Linux, macOS, Windows
- Successful Claude Code integration with proper environment

## Risk Mitigation

- **Zero-dependency principle:** Cobra explicitly approved, minimal impact
- **Cross-platform complexity:** Platform-specific code with extensive testing
- **Backward compatibility:** Comprehensive test coverage of existing patterns
- **Race conditions:** Proper file locking and PID validation