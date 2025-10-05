# Athena

<p align="center">
  <img src="Athena.png" alt="Athena" width="200"/>
</p>

> *Athena, ancient Greek goddess associated with wisdom, warfare, and handicraft.*

A proxy server that maps Anthropic's API format to OpenAI API format, allowing you to use Claude Code with OpenRouter's vast selection of AI models.

## Features

- 🔄 **API Translation**: Maps Anthropic API calls to OpenRouter format
- 🌊 **Streaming Support**: Full SSE streaming for real-time responses
- 🛠️ **Tool Calling**: Complete function/tool calling support
- 🎯 **Model Mapping**: Configurable mappings for Opus, Sonnet, and Haiku models
- 🔀 **Provider Routing**: Automatic Groq provider routing for Kimi K2 models
- ⚙️ **Flexible Configuration**: CLI flags, config files, environment variables, and .env files
- 🚀 **Minimal Dependencies**: Lightweight with only essential external packages (Cobra CLI, YAML parser)

## Quick Start

### One-line install:
```bash
curl -fsSL https://raw.githubusercontent.com/martinffx/athena/main/install.sh | bash
```

### Manual installation:
1. Download the latest release from [GitHub Releases](https://github.com/martinffx/athena/releases)
2. Extract and move to your PATH
3. Set up configuration (see below)

## Configuration

The proxy looks for configuration in this priority order (highest to lowest):
1. **Command line flags** - CLI arguments override everything
2. **Environment variables** - ATHENA_* prefixed env vars
3. **Local config file** - `./athena.yml` in current directory
4. **Global config file** - `~/.config/athena/athena.yml`
5. **Built-in defaults** - Hardcoded fallback values

### Config File Example (YAML):
```yaml
# ~/.config/athena/athena.yml
port: "12377"
api_key: "your-openrouter-api-key-here"
base_url: "https://openrouter.ai/api"
model: "moonshotai/kimi-k2-0905"
opus_model: "anthropic/claude-3-opus"
sonnet_model: "anthropic/claude-3.5-sonnet"
haiku_model: "anthropic/claude-3.5-haiku"
```

**Note:** The default model `moonshotai/kimi-k2-0905` automatically uses Groq provider routing for optimal performance.

### Advanced: Provider Routing

For fine-grained control over provider routing, add provider configurations to your YAML config:

```yaml
port: "12377"
api_key: "your-openrouter-api-key-here"
base_url: "https://openrouter.ai/api"
model: "moonshotai/kimi-k2-0905"
default_provider:
  order:
    - Groq
  allow_fallbacks: false
opus_model: "anthropic/claude-3-opus"
opus_provider:
  order:
    - Anthropic
  allow_fallbacks: true
sonnet_model: "anthropic/claude-3.5-sonnet"
haiku_model: "anthropic/claude-3.5-haiku"
```

Provider routing allows you to:
- Force requests through specific providers (e.g., Groq, Anthropic)
- Control fallback behavior when the primary provider is unavailable
- Configure different providers for different model tiers

### Environment Variables:
```bash
export OPENROUTER_API_KEY="your-key"
export OPUS_MODEL="anthropic/claude-3-opus"
export SONNET_MODEL="anthropic/claude-3.5-sonnet"
export HAIKU_MODEL="anthropic/claude-3.5-haiku"
export DEFAULT_MODEL="moonshotai/kimi-k2-0905"
export PORT="12377"
```

### .env File:
```bash
# ./.env
OPENROUTER_API_KEY=your-openrouter-api-key-here
OPUS_MODEL=anthropic/claude-3-opus
SONNET_MODEL=anthropic/claude-3.5-sonnet
HAIKU_MODEL=anthropic/claude-3.5-haiku
```

## Usage

### Daemon Management

```bash
# Run in foreground (default)
athena

# Run as background daemon
athena start

# Stop daemon
athena stop

# Check daemon status
athena status

# View logs (daemon mode)
tail -f ~/.athena/athena.log
```

### Custom Configuration
```bash
# Use specific models and port (foreground)
athena -port 9000 -api-key YOUR_KEY

# Or run as daemon with custom port
athena start -port 9000 -api-key YOUR_KEY

# Enable debug logging to see full request/response bodies
athena --log-level debug
```

### Using with Claude Code

```bash
# Start Athena daemon
athena start

# Configure Claude Code to use the proxy
export ANTHROPIC_BASE_URL=http://localhost:12377
export ANTHROPIC_API_KEY=your-openrouter-key

# Run Claude Code
claude
```

## How It Works

The proxy server:

1. **Receives** Anthropic API calls from Claude Code on `/v1/messages`
2. **Transforms** the request format to OpenAI-compatible format
3. **Forwards** to OpenRouter's `/v1/chat/completions` endpoint
4. **Converts** the response back to Anthropic format
5. **Streams** the response back to Claude Code

### Model Mapping

When Claude Code requests a model:
- `claude-3-opus*` → Your configured `opus_model`
- `claude-3.5-sonnet*` → Your configured `sonnet_model` 
- `claude-3.5-haiku*` → Your configured `haiku_model`
- Models with `/` (e.g., `openai/gpt-4`) → Passed through as-is
- Other models → Your configured `default_model`

## Building from Source

```bash
git clone https://github.com/martinffx/athena.git
cd athena
go build -o athena ./cmd/athena
```

## API Compatibility

The proxy provides a fully compatible Anthropic Messages API that supports:

- ✅ Text generation
- ✅ Streaming responses  
- ✅ System messages
- ✅ Tool/function calling
- ✅ Multi-turn conversations
- ✅ Content blocks (text, tool_use, tool_result)
- ✅ Usage tracking
- ✅ Stop reasons

## Endpoints

- `POST /v1/messages` - Anthropic Messages API (proxied to OpenRouter)
- `GET /health` - Health check endpoint

## Supported Platforms

- Linux (AMD64, ARM64)
- macOS (Intel, Apple Silicon)  
- Windows (AMD64, ARM64)

## Configuration Examples

### Use Claude Code with GPT-4:
```yaml
opus_model: "openai/gpt-4"
sonnet_model: "openai/gpt-4-turbo"
haiku_model: "openai/gpt-3.5-turbo"
```

### Use Claude Code with Gemini:
```yaml
opus_model: "google/gemini-pro"
sonnet_model: "google/gemini-pro"
haiku_model: "google/gemini-pro"
```

### Use Claude Code with Local Ollama:
```yaml
base_url: "http://localhost:12377/v1"
opus_model: "llama3:70b"
sonnet_model: "llama3:8b" 
haiku_model: "llama3:8b"
```

## Troubleshooting

### Port already in use:
```bash
# Use a different port
athena -port 9000
```

### API key issues:
```bash
# Check if key is set
echo $OPENROUTER_API_KEY

# Test the proxy directly
curl -X POST http://localhost:12377/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: your-key" \
  -d '{"model":"claude-3-sonnet","messages":[{"role":"user","content":"Hi"}]}'
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.