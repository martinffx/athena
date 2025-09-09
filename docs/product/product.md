# OpenRouter CC - Product Definition

## Overview
OpenRouter CC is a zero-dependency Go proxy server that translates Anthropic API requests to OpenRouter format, enabling Claude Code users to access diverse AI models with cost savings and local development options.

## Target Users

### Primary Users
- **Claude Code developers** wanting access to more AI models than just Anthropic's
- **Cost-conscious developers** using OpenRouter's competitive pricing
- **Local development users** integrating with Ollama or self-hosted models
- **Enterprise teams** needing flexible model selection

### User Problems Solved
1. **Limited model access**: Claude Code only supports Anthropic models natively
2. **High costs**: Anthropic pricing can be expensive for heavy usage
3. **Local development**: No easy way to use local models with Claude Code
4. **Model experimentation**: Difficult to test different models for specific tasks

## Core Value Propositions
1. **Seamless Integration**: Drop-in replacement that works with existing Claude Code workflows
2. **Cost Optimization**: Access to OpenRouter's competitive model pricing
3. **Model Diversity**: Support for 100+ models from various providers
4. **Zero Dependencies**: Single Go binary with standard library only
5. **Local Development**: Built-in support for Ollama and self-hosted models

## Core Features

### 1. API Translation
- **Bidirectional conversion**: Anthropic Messages API ↔ OpenRouter Chat Completions
- **Format compatibility**: Maintains full compatibility with Claude Code expectations
- **Error handling**: Proper HTTP status codes and error message translation

### 2. Streaming Support
- **Real-time responses**: Server-Sent Events (SSE) with proper buffering
- **State management**: Tracks content blocks and tool calls during streaming
- **Line buffering**: Handles incomplete SSE data from upstream providers

### 3. Model Mapping
- **Intelligent routing**: claude-3-opus/sonnet/haiku → configurable OpenRouter models
- **Pass-through support**: Direct OpenRouter model IDs (provider/model format)
- **Default fallback**: Configurable default model for unmapped requests

### 4. Tool Calling
- **Complete support**: Function/tool calling with JSON schema validation
- **Schema cleaning**: Removes unsupported format properties for compatibility
- **Response validation**: Ensures tool calls have matching tool responses

### 5. Configuration System
- **Multi-source loading**: CLI flags → config files → env vars → defaults
- **Flexible formats**: YAML, JSON, and environment variable support
- **Runtime override**: Command-line flags take precedence for development

### 6. Claude Code Integration
- **Launcher wrapper**: Automatic proxy startup + Claude Code launch
- **Health checks**: Built-in endpoint monitoring for reliability
- **Graceful shutdown**: Clean process termination and resource cleanup

## Success Metrics
- **Adoption**: Number of active Claude Code users using the proxy
- **Cost savings**: Average monthly billing reduction for users
- **Model usage**: Distribution of model requests across providers
- **Reliability**: Uptime and error rates for proxy operations
- **Performance**: Request latency and streaming response times

## Competitive Advantages
1. **Zero configuration**: Works out-of-the-box with sensible defaults
2. **Single binary**: No runtime dependencies or complex installation
3. **Claude Code native**: Built specifically for Claude Code integration
4. **Model agnostic**: Supports any OpenRouter-compatible provider
5. **Local development**: Seamless Ollama and self-hosted model support