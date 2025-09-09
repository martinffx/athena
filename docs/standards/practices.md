# OpenRouter CC - Development Practices

## Test-Driven Development (TDD)

### TDD Workflow for OpenRouter CC
Given the project's monolithic structure, TDD focuses on testing transformation functions and HTTP handlers:

1. **Red**: Write failing test for new functionality
2. **Green**: Implement minimal code to pass test
3. **Refactor**: Improve code while keeping tests passing

### Testing Strategy

#### Unit Testing Focus Areas
```go
// 1. Message transformation functions
func TestTransformAnthropicToOpenAI(t *testing.T) { ... }
func TestTransformOpenAIToAnthropic(t *testing.T) { ... }

// 2. Configuration loading
func TestLoadConfig(t *testing.T) { ... }
func TestLoadConfigFromFile(t *testing.T) { ... }

// 3. Model mapping logic
func TestResolveModel(t *testing.T) { ... }

// 4. Tool schema cleaning
func TestCleanToolSchema(t *testing.T) { ... }

// 5. Request validation
func TestValidateAnthropicRequest(t *testing.T) { ... }
```

#### Test Data Patterns
Create reusable test fixtures for complex JSON structures:

```go
var (
    sampleAnthropicRequest = AnthropicRequest{
        Model: "claude-3-sonnet",
        Messages: []Message{
            {Role: "user", Content: json.RawMessage(`"Hello"`)},
        },
        System: json.RawMessage(`"You are a helpful assistant"`),
        Stream: true,
    }
    
    expectedOpenAIRequest = OpenAIRequest{
        Model: "anthropic/claude-3-sonnet-20240229",
        Messages: []Message{
            {Role: "system", Content: json.RawMessage(`"You are a helpful assistant"`)},
            {Role: "user", Content: json.RawMessage(`"Hello"`)},
        },
        Stream: true,
    }
)
```

#### HTTP Handler Testing
Test HTTP handlers with httptest package:

```go
func TestHandleMessages(t *testing.T) {
    // Create request
    reqBody := `{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hello"}]}`
    req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    // Create response recorder
    rr := httptest.NewRecorder()
    
    // Call handler
    handleMessages(rr, req)
    
    // Assert response
    assert.Equal(t, http.StatusOK, rr.Code)
    assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
}
```

### Integration Testing

#### Mock OpenRouter Server
Create test server for integration testing:

```go
func createMockOpenRouterServer() *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate request format
        var req OpenAIRequest
        json.NewDecoder(r.Body).Decode(&req)
        
        // Return mock response
        resp := OpenAIResponse{
            Choices: []Choice{
                {Message: Message{Role: "assistant", Content: json.RawMessage(`"Hello back!"`)}},
            },
        }
        json.NewEncoder(w).Encode(resp)
    }))
}

func TestFullRequestFlow(t *testing.T) {
    mockServer := createMockOpenRouterServer()
    defer mockServer.Close()
    
    // Update config to use mock server
    config.BaseURL = mockServer.URL
    
    // Test full request flow
    // ...
}
```

## Development Workflow

### Local Development Setup
```bash
# 1. Clone and setup
git clone <repo>
cd openrouter-cc
make setup

# 2. Create local config
cp openrouter.example.yml openrouter.yml
# Edit with your OpenRouter API key

# 3. Run development checks
make check   # fmt + lint + vet + test

# 4. Start development server
make dev     # Runs with hot reload if available
```

### Code Quality Checks

#### Pre-commit Workflow
```bash
# Format code
make fmt
# or: go fmt ./...

# Lint code
make lint  
# or: golangci-lint run

# Vet code
make vet
# or: go vet ./...

# Run tests
make test
# or: go test -v ./...

# All checks together
make check
```

#### Continuous Integration
All code changes must pass:
1. **Formatting**: `go fmt` produces no changes
2. **Linting**: `golangci-lint` passes with zero warnings
3. **Vetting**: `go vet` finds no issues  
4. **Testing**: All tests pass with coverage >80%
5. **Building**: Cross-compilation succeeds for all platforms

### Git Workflow

#### Branch Strategy
- **main**: Production-ready code, protected branch
- **feature/**: Feature development branches
- **fix/**: Bug fix branches  
- **release/**: Release preparation branches

#### Commit Standards
```bash
# Format: <type>: <description>
# 
# Types: feat, fix, docs, style, refactor, test, build

git commit -m "feat: add model mapping configuration support

Add support for configurable model mappings in YAML config files.
Users can now specify custom mappings for claude-3-opus/sonnet/haiku
to any OpenRouter model ID.

- Add OpusModel, SonnetModel, HaikuModel config fields
- Update model resolution logic in resolveModel function
- Add validation for model mapping configuration
- Update example config with model mapping examples"
```

#### Pull Request Process
1. **Create feature branch** from main
2. **Implement changes** using TDD workflow
3. **Run quality checks** (`make check`)
4. **Push branch** and create pull request
5. **Code review** by at least one maintainer
6. **Merge** after approval and CI success

### Performance Testing

#### Benchmark Critical Paths
```go
func BenchmarkTransformAnthropicToOpenAI(b *testing.B) {
    req := sampleAnthropicRequest
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        transformAnthropicToOpenAI(req)
    }
}

func BenchmarkHandleMessages(b *testing.B) {
    reqBody := `{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hello"}]}`
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(reqBody))
        rr := httptest.NewRecorder()
        handleMessages(rr, req)
    }
}
```

#### Performance Targets
- **Transformation latency**: <1ms for typical requests
- **Memory allocation**: <100KB per request transformation  
- **Throughput**: Handle 1000+ req/sec on standard hardware
- **Streaming latency**: <50ms first byte time

### Error Handling Standards

#### Error Types and Responses
```go
// Client errors (4xx)
func handleBadRequest(w http.ResponseWriter, message string) {
    http.Error(w, fmt.Sprintf(`{"error":{"type":"invalid_request_error","message":"%s"}}`, message), 400)
}

// Server errors (5xx)
func handleInternalError(w http.ResponseWriter, err error) {
    log.Printf("Internal error: %v", err)
    http.Error(w, `{"error":{"type":"api_error","message":"Internal server error"}}`, 500)
}

// Upstream errors (pass through)
func handleUpstreamError(w http.ResponseWriter, statusCode int, body []byte) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    w.Write(body)
}
```

#### Logging Standards
```go
import "log"

// Structured logging approach
log.Printf("Request: method=%s path=%s remote=%s", r.Method, r.URL.Path, r.RemoteAddr)
log.Printf("Response: status=%d latency=%v", statusCode, time.Since(start))
log.Printf("Error: %v", err)
```

### Configuration Management

#### Environment-Specific Configs
```yaml
# Development config (openrouter.dev.yml)
port: "11434"
base_url: "http://localhost:11434/v1"  # Local Ollama
model: "llama3:8b"

# Production config (openrouter.prod.yml)  
port: "8080"
base_url: "https://openrouter.ai/api/v1"
model: "anthropic/claude-3-sonnet-20240229"
```

#### Configuration Validation
```go
func validateConfig(config Config) error {
    if config.Port == "" {
        return errors.New("port is required")
    }
    
    if config.BaseURL == "" {
        return errors.New("base_url is required")
    }
    
    // Validate URL format
    if _, err := url.Parse(config.BaseURL); err != nil {
        return fmt.Errorf("invalid base_url format: %v", err)
    }
    
    return nil
}
```

### Deployment Practices

#### Release Checklist
- [ ] All tests passing
- [ ] Performance benchmarks within targets
- [ ] Security review completed
- [ ] Documentation updated
- [ ] Example configs updated
- [ ] Wrapper scripts tested on all platforms
- [ ] GitHub release notes prepared

#### Binary Distribution
```bash
# Cross-compile for all supported platforms
make build-all

# Outputs:
# openrouter-cc-darwin-amd64
# openrouter-cc-darwin-arm64  
# openrouter-cc-linux-amd64
# openrouter-cc-linux-arm64
# openrouter-cc-windows-amd64.exe
# openrouter-cc-windows-arm64.exe
```

#### Health Monitoring
```go
func handleHealth(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status": "healthy",
        "version": Version,
        "uptime": time.Since(startTime),
        "requests": atomic.LoadInt64(&requestCount),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

These practices ensure consistent quality, maintainability, and reliability across all development activities while accommodating the unique constraints of the single-file architecture.