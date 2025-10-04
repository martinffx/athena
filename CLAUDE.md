# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Athena is a Go-based HTTP proxy server that translates Anthropic API requests to OpenRouter format, enabling Claude Code to work with OpenRouter's diverse model selection. The application uses minimal external dependencies (Cobra CLI framework, YAML parser) and follows standard Go project layout with `cmd/` and `internal/` packages.

**Status**: Production-ready with all core features implemented and tested.

## Documentation Structure

### Product Documentation (`docs/product/`)
- **product.md** - Product definition, target users, core features and value propositions
- **roadmap.md** - Current status (production-ready) and future enhancement priorities

### Technical Standards (`docs/standards/`)
- **tech.md** - Architecture overview, tech stack, component design, and performance characteristics
- **style.md** - Code organization, naming conventions, and documentation standards for single-file architecture
- **practices.md** - TDD workflow, development practices, error handling, and deployment standards

### Feature Specifications (`docs/spec/`)
- Directory for detailed implementation specifications for new features
- Use Spec-Driven Development approach for major enhancements

## Core Architecture

### Request Flow
1. **Anthropic API Input** (`/v1/messages`) → **Format Transformation** → **OpenRouter API** (`/v1/chat/completions`) → **Response Transformation** → **Anthropic Format Output**

### Key Components
- **Configuration System**: Multi-source config loading (CLI flags → config files → env vars → defaults)
- **Message Transformation**: Bidirectional conversion between Anthropic and OpenAI/OpenRouter formats
- **Streaming Handler**: Server-Sent Events (SSE) processing with proper buffering
- **Model Mapping**: Configurable mappings for claude-3-opus/sonnet/haiku to any OpenRouter model
- **Tool/Function Support**: Complete tool calling with JSON schema cleaning

### Data Structures
- `AnthropicRequest/Response` - Anthropic Messages API format
- `OpenAIRequest/Message` - OpenRouter/OpenAI chat completions format  
- `Config` - Unified configuration structure
- `ContentBlock` - Handles text, tool_use, and tool_result content types

## Development Commands

### Quick Setup
```bash
# Set up development environment
make setup

# Copy and edit config
cp openrouter.example.yml openrouter.yml
# Edit with your OpenRouter API key

# Build and run
make dev
```

### Development Workflow
```bash
# Format, lint, test - run before committing
make check

# Individual commands
make fmt      # Format code
make lint     # Run golangci-lint  
make vet      # Run go vet
make test     # Run tests
make build    # Build binary

# Cross-platform builds
make build-all         # Build for Linux, macOS, Windows
make release-test      # Test release build process
```

### Build and Run
```bash
# Build binary
make build
# or: go build -ldflags="-s -w" -o athena ./cmd/athena

# Run with default config
./athena

# Run with custom config
./athena -port 9000 -api-key YOUR_KEY
```

### CLI Commands

Athena provides a simple 4-command interface:

```bash
# Run server in foreground (default)
athena

# Start server as background daemon
athena start

# Stop the daemon
athena stop

# Show daemon status
athena status
```

Logs are written to `~/.athena/athena.log` in daemon mode. View with standard tools:
```bash
# Follow logs in real-time
tail -f ~/.athena/athena.log

# Search logs
grep "error" ~/.athena/athena.log

# View last 100 lines
tail -n 100 ~/.athena/athena.log
```

**Log Levels:**
- `info` (default): High-level request/response metadata without bodies
- `debug`: Full request/response bodies logged (useful for troubleshooting)
- `warn`: Warnings and errors only
- `error`: Errors only

```bash
# Enable debug logging
athena --log-level debug

# Or via config file
# athena.yml:
log_level: "debug"

# Or via environment variable
export ATHENA_LOG_LEVEL=debug
athena start
```

### Testing the Proxy
```bash
# Health check
curl http://localhost:12377/health

# Test message endpoint
curl -X POST http://localhost:12377/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: your-openrouter-key" \
  -d '{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hello"}]}'
```

### Configuration Management
The configuration system follows this priority (highest to lowest):
1. **CLI flags** (via --flag options)
2. **Environment variables** (ATHENA_* prefixed)
3. **Local config file** (./athena.yml in current directory)
4. **Global config file** (~/.config/athena/athena.yml)
5. **Defaults** (hardcoded in config.go)

Config file discovery:
- If `--config` flag is provided, only that file is loaded
- Otherwise, both global and local configs are discovered and merged (local overrides global)
- Runtime state (PID file, logs) stored in `~/.athena/`

## Key Implementation Details

### Message Transformation Logic
- **System messages**: Extracted from Anthropic format and prepended to OpenAI messages array
- **Content normalization**: Handles both string content and content block arrays
- **Tool validation**: Ensures tool_calls have matching tool responses via `validateToolCalls()`
- **JSON schema cleaning**: Removes unsupported `format: "uri"` properties from tool schemas

### Streaming Implementation  
- **Line-by-line processing**: Buffers incomplete SSE lines from OpenRouter
- **Event mapping**: Converts OpenAI delta events to Anthropic content block events
- **State management**: Tracks content block indices and tool call state during streaming

### Model Mapping Strategy
```go
// Built-in model detection
if strings.Contains(model, "opus") → config.OpusModel
if strings.Contains(model, "sonnet") → config.SonnetModel  
if strings.Contains(model, "haiku") → config.HaikuModel
if strings.Contains(model, "/") → pass-through (OpenRouter model ID)
else → config.Model (default)
```

## Development Standards & Practices

### Code Quality Standards
- **TDD Approach**: Write tests first for all new functionality (see `docs/standards/practices.md`)
- **Code Style**: Follow single-file organization patterns (see `docs/standards/style.md`)
- **Architecture**: Maintain separation of concerns within monolithic structure (see `docs/standards/tech.md`)

### Quality Gates
All code changes must pass:
1. **Formatting**: `make fmt` produces no changes
2. **Linting**: `make lint` passes with zero warnings  
3. **Testing**: `make test` passes with >80% coverage
4. **Building**: `make build-all` succeeds for all platforms

### Spec-Driven Development
For new features, follow the Spec-Driven Development approach:
1. **Business Context**: Document in `docs/product/` if needed
2. **Technical Specification**: Create detailed spec in `docs/spec/{feature}/`
3. **Implementation**: Follow TDD workflow with test-first development
4. **Documentation**: Update standards and practices as needed

## Release Process

### GitHub Actions Workflow
- **Trigger**: Git tags (`v*`) or manual dispatch
- **Cross-compilation**: 6 platforms (Linux/macOS/Windows × AMD64/ARM64)
- **Assets**: Binaries + wrapper scripts + example configs
- **Optimization**: Uses `-ldflags="-s -w"` for size reduction

### Manual Release
```bash
# Tag and push
git tag v1.0.0
git push --tags

# Local cross-compilation example
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o athena-linux-amd64 ./cmd/athena
```

## Configuration Examples

### Multi-provider setup
```yaml
# Use different providers for different model tiers
opus_model: "anthropic/claude-3-opus"      # High-end
sonnet_model: "openai/gpt-4"              # Mid-tier  
haiku_model: "google/gemini-pro"          # Fast/cheap
```

### Local development with Ollama
```yaml
base_url: "http://localhost:12377/v1"
opus_model: "llama3:70b"
sonnet_model: "llama3:8b"
```

## Code Patterns

### Error Handling
All HTTP handlers follow the pattern: validate input → transform → proxy request → transform response → handle errors with proper status codes.

### Configuration Loading
Multi-source configuration uses the `loadConfig()` function which processes sources in priority order and only overwrites empty values.

### JSON Processing
Heavy use of `json.RawMessage` for flexible content handling, especially for system messages and tool inputs that can be strings or complex objects.

## Performance & Monitoring

### Performance Targets (see `docs/standards/practices.md`)
- **Transformation latency**: <1ms for typical requests
- **Memory allocation**: <100KB per request transformation
- **Throughput**: Handle 1000+ req/sec on standard hardware
- **Streaming latency**: <50ms first byte time

### Health Monitoring
The `/health` endpoint provides service status, version, uptime, and request metrics for monitoring and alerting.