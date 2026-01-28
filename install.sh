#!/usr/bin/env bash
set -euo pipefail

REPO="W-Nana/remnawave-node-go"
BINARY_NAME="remnawave-node-go"
INSTALL_DIR="/usr/local/bin"
SERVICE_DIR="/etc/systemd/system"
CONFIG_DIR="/etc/remnawave-node"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root. Use: sudo bash install.sh"
    fi
}

detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux) OS="linux" ;;
        darwin) OS="darwin" ;;
        freebsd) OS="freebsd" ;;
        *) log_error "Unsupported OS: $OS" ;;
    esac
    echo "$OS"
}

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv7) ARCH="arm" ;;
        *) log_error "Unsupported architecture: $ARCH" ;;
    esac
    echo "$ARCH"
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$version" ]]; then
        log_error "Failed to get latest version"
    fi
    echo "$version"
}

download_binary() {
    local version=$1
    local os=$2
    local arch=$3
    
    local filename="remnawave-node-go-${version}-${os}-${arch}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    local tmp_dir=$(mktemp -d)
    
    log_info "Downloading ${filename}..."
    
    if ! curl -fsSL "$url" -o "${tmp_dir}/${filename}"; then
        rm -rf "$tmp_dir"
        log_error "Failed to download from: $url"
    fi
    
    log_info "Extracting..."
    tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"
    
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_info "Stopping existing service..."
        systemctl stop ${BINARY_NAME} 2>/dev/null || true
    fi
    
    mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    rm -rf "$tmp_dir"
    log_success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

create_config_dir() {
    if [[ ! -d "$CONFIG_DIR" ]]; then
        mkdir -p "$CONFIG_DIR"
        log_info "Created config directory: $CONFIG_DIR"
    fi
    
    if [[ ! -f "${CONFIG_DIR}/.env" ]]; then
        cat > "${CONFIG_DIR}/.env" << 'EOF'
# Remnawave Node Configuration
# Copy this file and fill in your values

# Required: Panel connection
APP_PORT=3000
PANEL_HOST=https://your-panel.example.com
PANEL_TOKEN=your-node-token-here

# Optional: SSL configuration
# SSL_CERT_PATH=/etc/remnawave-node/cert.pem
# SSL_KEY_PATH=/etc/remnawave-node/key.pem
EOF
        chmod 600 "${CONFIG_DIR}/.env"
        log_info "Created sample config: ${CONFIG_DIR}/.env"
        log_warn "Edit ${CONFIG_DIR}/.env with your panel details before starting"
    fi
}

install_systemd_service() {
    cat > "${SERVICE_DIR}/${BINARY_NAME}.service" << EOF
[Unit]
Description=Remnawave Node (Go)
Documentation=https://github.com/${REPO}
After=network.target nss-lookup.target

[Service]
Type=simple
User=root
EnvironmentFile=${CONFIG_DIR}/.env
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=on-failure
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    log_success "Systemd service installed"
}

show_usage() {
    echo ""
    log_info "Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Edit config:     nano ${CONFIG_DIR}/.env"
    echo "  2. Start service:   systemctl start ${BINARY_NAME}"
    echo "  3. Enable on boot:  systemctl enable ${BINARY_NAME}"
    echo "  4. Check status:    systemctl status ${BINARY_NAME}"
    echo "  5. View logs:       journalctl -u ${BINARY_NAME} -f"
    echo ""
}

uninstall() {
    log_info "Uninstalling ${BINARY_NAME}..."
    
    systemctl stop ${BINARY_NAME} 2>/dev/null || true
    systemctl disable ${BINARY_NAME} 2>/dev/null || true
    
    rm -f "${INSTALL_DIR}/${BINARY_NAME}"
    rm -f "${SERVICE_DIR}/${BINARY_NAME}.service"
    systemctl daemon-reload
    
    log_success "Uninstalled successfully"
    log_info "Config preserved at: ${CONFIG_DIR}"
    log_info "To remove config: rm -rf ${CONFIG_DIR}"
}

main() {
    local cmd="${1:-install}"
    local version="${2:-}"
    
    case "$cmd" in
        install)
            check_root
            
            local os=$(detect_os)
            local arch=$(detect_arch)
            
            log_info "Detected: ${os}/${arch}"
            
            if [[ -z "$version" ]]; then
                version=$(get_latest_version)
            fi
            
            log_info "Installing version: ${version}"
            
            download_binary "$version" "$os" "$arch"
            create_config_dir
            
            if [[ "$os" == "linux" ]] && command -v systemctl &>/dev/null; then
                install_systemd_service
            fi
            
            show_usage
            ;;
        uninstall)
            check_root
            uninstall
            ;;
        update)
            check_root
            main install "$version"
            systemctl restart ${BINARY_NAME} 2>/dev/null || true
            ;;
        *)
            echo "Usage: $0 {install|uninstall|update} [version]"
            echo ""
            echo "Commands:"
            echo "  install [version]  Install (latest if no version specified)"
            echo "  uninstall          Remove binary and service"
            echo "  update [version]   Update to latest or specified version"
            echo ""
            echo "Examples:"
            echo "  sudo bash install.sh install"
            echo "  sudo bash install.sh install v1.0.0"
            echo "  sudo bash install.sh update"
            echo "  sudo bash install.sh uninstall"
            ;;
    esac
}

main "$@"
