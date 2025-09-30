#!/bin/bash

# Athena Install Script
# Downloads and installs the latest release of athena

set -e

# Configuration
REPO="martinffx/athena"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
LATEST_URL="https://api.github.com/repos/$REPO/releases/latest"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[install]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[install]${NC} $1"
}

error() {
    echo -e "${RED}[install]${NC} $1"
}

success() {
    echo -e "${GREEN}[install]${NC} $1"
}

# Detect platform
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            os="windows"
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release info
get_latest_release() {
    if command -v curl >/dev/null 2>&1; then
        curl -s "$LATEST_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O - "$LATEST_URL"
    else
        error "curl or wget is required to download releases"
        exit 1
    fi
}

# Download file
download_file() {
    local url="$1"
    local output="$2"
    
    log "Downloading: $url"
    
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$output" "$url"
    else
        error "curl or wget is required to download files"
        exit 1
    fi
}

# Extract download URL from release JSON
get_download_url() {
    local json="$1"
    local filename="$2"
    
    # Simple JSON parsing to find the browser_download_url for our file
    echo "$json" | grep -o "\"browser_download_url\": *\"[^\"]*$filename[^\"]*\"" | cut -d'"' -f4 | head -1
}

main() {
    log "Athena Installer"

    # Detect platform
    PLATFORM=$(detect_platform)
    log "Detected platform: $PLATFORM"

    # Determine filenames
    if [[ "$PLATFORM" == *"windows"* ]]; then
        BINARY_NAME="athena-${PLATFORM}.exe"
        WRAPPER_NAME="athena-wrapper-${PLATFORM}.bat"
    else
        BINARY_NAME="athena-${PLATFORM}"
        WRAPPER_NAME="athena-wrapper-${PLATFORM}"
    fi
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Get latest release info
    log "Fetching latest release information..."
    RELEASE_JSON=$(get_latest_release)
    
    if [[ -z "$RELEASE_JSON" ]]; then
        error "Failed to fetch release information"
        exit 1
    fi
    
    # Extract version
    VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
    log "Latest version: $VERSION"
    
    # Get download URLs
    BINARY_URL=$(get_download_url "$RELEASE_JSON" "$BINARY_NAME")
    WRAPPER_URL=$(get_download_url "$RELEASE_JSON" "$WRAPPER_NAME")
    
    if [[ -z "$BINARY_URL" ]]; then
        error "Could not find binary download URL for platform: $PLATFORM"
        error "Available files:"
        echo "$RELEASE_JSON" | grep -o '"name": *"[^"]*"' | cut -d'"' -f4 | grep -E "(athena-|athena-wrapper-)" | sort
        exit 1
    fi

    # Download binary
    BINARY_PATH="$INSTALL_DIR/athena"
    download_file "$BINARY_URL" "$BINARY_PATH"
    chmod +x "$BINARY_PATH"

    # Download wrapper script if available
    if [[ -n "$WRAPPER_URL" ]]; then
        if [[ "$PLATFORM" == *"windows"* ]]; then
            WRAPPER_PATH="$INSTALL_DIR/athena-wrapper.bat"
        else
            WRAPPER_PATH="$INSTALL_DIR/athena-wrapper"
        fi
        download_file "$WRAPPER_URL" "$WRAPPER_PATH"
        chmod +x "$WRAPPER_PATH" 2>/dev/null || true
        success "Installed wrapper script: $WRAPPER_PATH"
    fi
    
    success "Installed binary: $BINARY_PATH"
    
    # Download example configs
    CONFIG_URLS=(
        "$(get_download_url "$RELEASE_JSON" "athena.example.yml")"
        "$(get_download_url "$RELEASE_JSON" "athena.example.json")"
        "$(get_download_url "$RELEASE_JSON" ".env.example")"
    )

    CONFIG_DIR="$HOME/.config/athena"
    mkdir -p "$CONFIG_DIR"

    for url in "${CONFIG_URLS[@]}"; do
        if [[ -n "$url" ]]; then
            filename=$(basename "$url")
            download_file "$url" "$CONFIG_DIR/$filename"
            log "Downloaded example config: $CONFIG_DIR/$filename"
        fi
    done
    
    # Check if install dir is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "Install directory ($INSTALL_DIR) is not in your PATH"
        warn "Add this line to your shell profile (.bashrc, .zshrc, etc.):"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\""
        warn "Or use the full path: $BINARY_PATH"
    fi
    
    success "Installation complete!"
    echo
    log "Next steps:"
    echo "1. Copy example config: cp $CONFIG_DIR/athena.example.yml $CONFIG_DIR/athena.yml"
    echo "2. Edit config with your OpenRouter API key"
    echo "3. Run: athena (server only) or athena-wrapper (server + Claude Code)"
    echo
    log "For more information, see: https://github.com/$REPO"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help|-h)
            echo "Athena Install Script"
            echo
            echo "Usage: $0 [options]"
            echo
            echo "Options:"
            echo "  --install-dir DIR    Install to DIR (default: $HOME/.local/bin)"
            echo "  --help, -h           Show this help message"
            echo
            echo "Environment variables:"
            echo "  INSTALL_DIR          Same as --install-dir"
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            exit 1
            ;;
    esac
done

main