#!/bin/bash

# IPTV Aggregator Service Installation Script
# This script automates the installation of IPTV Aggregator as a system service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_PATH="${1:-.}"
CONFIG_PATH="${2:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Functions
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root"
        echo "Please run: sudo $0"
        exit 1
    fi
}

check_binary() {
    if [[ ! -f "$BINARY_PATH/iptv-aggregator" ]]; then
        print_error "Binary not found at $BINARY_PATH/iptv-aggregator"
        echo "Please build the project first:"
        echo "  make build-linux"
        exit 1
    fi
    print_success "Binary found: $BINARY_PATH/iptv-aggregator"
}

check_linux() {
    if [[ "$OSTYPE" != "linux-gnu"* ]]; then
        print_error "This script only works on Linux"
        exit 1
    fi
    print_success "Running on Linux"
}

build_binary() {
    print_info "Building binary..."
    cd "$PROJECT_ROOT"
    make build-linux
    print_success "Binary built successfully"
}

install_service() {
    print_header "Installing IPTV Aggregator Service"
    
    if [[ -z "$CONFIG_PATH" ]]; then
        print_info "No custom config specified, using default"
        "$BINARY_PATH/iptv-aggregator" -s install
    else
        print_info "Using custom config: $CONFIG_PATH"
        "$BINARY_PATH/iptv-aggregator" -s install -config "$CONFIG_PATH"
    fi
    
    print_success "Service installed"
}

start_service() {
    print_info "Starting service..."
    systemctl start iptv-aggregator
    print_success "Service started"
}

check_service_status() {
    print_header "Service Status"
    systemctl status iptv-aggregator
}

show_next_steps() {
    print_header "Next Steps"
    echo ""
    echo "Service management commands:"
    echo "  Start:    sudo systemctl start iptv-aggregator"
    echo "  Stop:     sudo systemctl stop iptv-aggregator"
    echo "  Restart:  sudo systemctl restart iptv-aggregator"
    echo "  Status:   sudo systemctl status iptv-aggregator"
    echo ""
    echo "View logs:"
    echo "  Real-time:  sudo journalctl -u iptv-aggregator -f"
    echo "  Last 50:    sudo journalctl -u iptv-aggregator -n 50"
    echo ""
    echo "Configuration:"
    echo "  Config file: /opt/iptv-aggregator/config.json"
    echo "  Edit:        sudo nano /opt/iptv-aggregator/config.json"
    echo "  Restart after editing: sudo systemctl restart iptv-aggregator"
    echo ""
    echo "Uninstall:"
    echo "  sudo /opt/iptv-aggregator/iptv-aggregator -s uninstall"
    echo ""
}

# Main execution
main() {
    print_header "IPTV Aggregator Service Installation"
    
    # Check prerequisites
    check_root
    check_linux
    
    # Parse arguments
    if [[ "$BINARY_PATH" == "build" ]]; then
        # If first argument is "build", build the binary first
        build_binary
        BINARY_PATH="$PROJECT_ROOT/build/linux"
        CONFIG_PATH="${2:-}"
    fi
    
    check_binary
    
    # Install service
    install_service
    
    # Start service
    read -p "Start service now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        start_service
        sleep 2
        check_service_status
    fi
    
    # Show next steps
    show_next_steps
}

# Run main function
main "$@"
