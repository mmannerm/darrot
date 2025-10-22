#!/bin/bash

# Script to install Google Container Structure Test tool for local development

set -euo pipefail

# Configuration
TOOL_NAME="container-structure-test"
INSTALL_DIR="/usr/local/bin"
TEMP_DIR="/tmp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if tool is already installed
check_existing_installation() {
    if command -v $TOOL_NAME &> /dev/null; then
        local version
        version=$($TOOL_NAME version 2>/dev/null || echo "unknown")
        log_info "$TOOL_NAME is already installed (version: $version)"
        read -p "Do you want to reinstall? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping installation"
            exit 0
        fi
    fi
}

# Detect OS and architecture
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *)          log_error "Unsupported operating system: $(uname -s)"; exit 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)              log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac
    
    echo "${os}-${arch}"
}

# Download and install the tool
install_tool() {
    local platform
    platform=$(detect_platform)
    
    local binary_name="${TOOL_NAME}-${platform}"
    local download_url="https://storage.googleapis.com/container-structure-test/latest/${binary_name}"
    local temp_file="${TEMP_DIR}/${binary_name}"
    
    log_info "Downloading $TOOL_NAME for platform: $platform"
    log_info "Download URL: $download_url"
    
    # Download the binary
    if ! curl -fsSL -o "$temp_file" "$download_url"; then
        log_error "Failed to download $TOOL_NAME"
        exit 1
    fi
    
    # Make it executable
    chmod +x "$temp_file"
    
    # Check if we need sudo for installation
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$temp_file" "$INSTALL_DIR/$TOOL_NAME"
    else
        log_info "Installing to $INSTALL_DIR (requires sudo)"
        sudo mv "$temp_file" "$INSTALL_DIR/$TOOL_NAME"
    fi
    
    log_info "$TOOL_NAME installed successfully to $INSTALL_DIR/$TOOL_NAME"
}

# Verify installation
verify_installation() {
    if command -v $TOOL_NAME &> /dev/null; then
        local version
        version=$($TOOL_NAME version 2>/dev/null || echo "unknown")
        log_info "Installation verified! Version: $version"
        
        # Test with a simple command
        log_info "Testing installation..."
        if $TOOL_NAME version &> /dev/null; then
            log_info "Installation test passed!"
        else
            log_warn "Installation test failed, but binary is accessible"
        fi
    else
        log_error "Installation verification failed - $TOOL_NAME not found in PATH"
        exit 1
    fi
}

# Main installation process
main() {
    log_info "Installing Google Container Structure Test tool..."
    
    check_existing_installation
    install_tool
    verify_installation
    
    log_info "Installation complete!"
    log_info "You can now run container structure tests with: make container-test"
    log_info "Or directly with: $TOOL_NAME test --image <image> --config <config>"
}

# Run main function
main "$@"