# OpenRouter CC - Code Style Standards

## Code Organization

### File Structure (Monolithic Design)
Given the project's single-file architecture, code organization follows a logical top-down structure:

```go
package main

// 1. IMPORTS - Standard library only
import (
    "bufio"
    "encoding/json" 
    "net/http"
    // ... other standard library imports
)

// 2. CONFIGURATION STRUCTURES
type Config struct { ... }

// 3. API DATA STRUCTURES  
type AnthropicRequest struct { ... }
type OpenAIRequest struct { ... }

// 4. TRANSFORMATION FUNCTIONS
func transformAnthropicToOpenAI() { ... }
func transformOpenAIToAnthropic() { ... }

// 5. HTTP HANDLERS
func handleMessages() { ... }
func handleHealth() { ... }

// 6. UTILITY FUNCTIONS
func loadConfig() { ... }
func validateToolCalls() { ... }

// 7. MAIN ENTRY POINT
func main() { ... }
```

### Naming Conventions

#### Functions
- **Public functions**: PascalCase (`TransformRequest`, `LoadConfig`)
- **Private functions**: camelCase (`transformAnthropicToOpenAI`, `validateToolCalls`)
- **Handler functions**: Prefix with `handle` (`handleMessages`, `handleHealth`)
- **Transform functions**: Descriptive direction (`transformAnthropicToOpenAI`)

#### Variables
- **Configuration**: Descriptive names (`config`, `baseURL`, `apiKey`)
- **HTTP objects**: Standard names (`req`, `resp`, `w`, `r`)
- **JSON data**: Contextual names (`anthropicReq`, `openaiResp`)
- **Iteration**: Single letters acceptable for short loops (`i`, `j`)

#### Constants
```go
const (
    DefaultPort    = "11434"
    DefaultBaseURL = "https://openrouter.ai/api/v1"
    HealthPath     = "/health"
    MessagesPath   = "/v1/messages"
)
```

#### Struct Fields
- **JSON tags**: Match API specifications exactly
- **YAML tags**: Use snake_case for configuration files
- **Optional fields**: Use `omitempty` for JSON serialization

```go
type Config struct {
    Port      string `yaml:"port" json:"port"`
    APIKey    string `yaml:"api_key" json:"api_key"`
    BaseURL   string `yaml:"base_url" json:"base_url"`
}

type ContentBlock struct {
    Type     string     `json:"type"`
    Text     string     `json:"text,omitempty"`
    ToolUse  *ToolUse   `json:"tool_use,omitempty"`
}
```

## Code Style Guidelines

### Error Handling
Always use explicit error checking with descriptive error messages:

```go
// GOOD: Explicit error handling
config, err := loadConfig()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// BAD: Ignoring errors
config, _ := loadConfig()
```

### HTTP Handler Pattern
Consistent pattern for all HTTP handlers:

```go
func handleMessages(w http.ResponseWriter, r *http.Request) {
    // 1. Method validation
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 2. Request parsing
    var req AnthropicRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // 3. Business logic
    transformedReq := transformAnthropicToOpenAI(req)
    
    // 4. Response handling
    resp, err := makeUpstreamRequest(transformedReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // 5. Response transformation and writing
    writeResponse(w, resp)
}
```

### JSON Handling
Use `json.RawMessage` for pass-through data and flexible content handling:

```go
// GOOD: Flexible JSON handling
type Message struct {
    Role    string          `json:"role"`
    Content json.RawMessage `json:"content"` // Can be string or []ContentBlock
}

// Parse content based on context
func parseContent(raw json.RawMessage) interface{} {
    // Try string first
    var str string
    if json.Unmarshal(raw, &str) == nil {
        return str
    }
    
    // Fall back to content blocks
    var blocks []ContentBlock
    if json.Unmarshal(raw, &blocks) == nil {
        return blocks
    }
    
    return nil
}
```

### Configuration Loading
Multi-source configuration with clear precedence:

```go
func loadConfig() (Config, error) {
    config := Config{
        Port:    DefaultPort,
        BaseURL: DefaultBaseURL,
    }
    
    // 1. Load from config files
    if err := loadConfigFiles(&config); err != nil {
        return config, err
    }
    
    // 2. Override with environment variables  
    loadEnvVars(&config)
    
    // 3. Override with CLI flags
    loadCLIFlags(&config)
    
    return config, nil
}
```

## Documentation Standards

### Function Comments
Document all public functions and complex private functions:

```go
// transformAnthropicToOpenAI converts an Anthropic Messages API request
// to OpenRouter/OpenAI Chat Completions format. It extracts system messages,
// normalizes content blocks, and cleans tool schemas for compatibility.
func transformAnthropicToOpenAI(req AnthropicRequest) OpenAIRequest {
    // Implementation...
}

// handleMessages processes Anthropic API requests, transforms them to OpenRouter
// format, forwards to the upstream provider, and transforms responses back.
func handleMessages(w http.ResponseWriter, r *http.Request) {
    // Implementation...
}
```

### Inline Comments
Use inline comments for complex logic and non-obvious transformations:

```go
// Extract system message from Anthropic format and prepend to messages array
if req.System != nil {
    systemMsg := Message{
        Role:    "system",
        Content: req.System, // Pass through as json.RawMessage
    }
    openaiReq.Messages = append([]Message{systemMsg}, openaiReq.Messages...)
}

// Clean tool schemas by removing unsupported "format" properties
for _, tool := range openaiReq.Tools {
    cleanToolSchema(tool.Function.Parameters)
}
```

### Code Structure Comments
Use section comments to organize the monolithic file:

```go
// ============================================================================
// CONFIGURATION AND DATA STRUCTURES
// ============================================================================

type Config struct { ... }

// ============================================================================  
// MESSAGE TRANSFORMATION FUNCTIONS
// ============================================================================

func transformAnthropicToOpenAI() { ... }

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func handleMessages() { ... }
```

## Performance Guidelines

### Memory Efficiency
- Use `json.RawMessage` to avoid unnecessary parsing
- Prefer streaming JSON processing for large requests
- Reuse buffers where possible in hot paths

### String Operations
- Use `strings.Builder` for string concatenation
- Avoid string concatenation in loops
- Use `strings.Contains` for simple substring checks

### HTTP Client Best Practices
```go
// Reuse HTTP client with appropriate timeouts
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        10,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     30 * time.Second,
    },
}

// Use context for request cancellation
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()
req = req.WithContext(ctx)
```

## Testing Standards

### Test Function Naming
- Test functions: `TestFunctionName`
- Benchmark functions: `BenchmarkFunctionName`
- Example functions: `ExampleFunctionName`

### Test Organization
```go
func TestTransformAnthropicToOpenAI(t *testing.T) {
    tests := []struct {
        name     string
        input    AnthropicRequest
        expected OpenAIRequest
    }{
        {
            name: "simple message transformation",
            input: AnthropicRequest{
                Model: "claude-3-sonnet",
                Messages: []Message{...},
            },
            expected: OpenAIRequest{...},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := transformAnthropicToOpenAI(tt.input)
            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %+v, want %+v", result, tt.expected)
            }
        })
    }
}
```

### Error Testing
Always test error conditions:

```go
func TestLoadConfig_InvalidFile(t *testing.T) {
    _, err := loadConfigFromFile("nonexistent.yml")
    if err == nil {
        t.Error("expected error for nonexistent file")
    }
}
```

## Build Standards

### Build Commands
```bash
# Development build
go build -o openrouter-cc main.go

# Production build (optimized)
go build -ldflags="-s -w" -o openrouter-cc main.go

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o openrouter-cc-linux-amd64 main.go
```

### Makefile Standards
- Use `.PHONY` for non-file targets
- Provide help target with descriptions
- Include common development commands (fmt, lint, test, build)

These standards ensure consistency, maintainability, and performance in the OpenRouter CC codebase while working within the constraints of a single-file architecture.