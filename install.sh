#!/usr/bin/env bash
set -euo pipefail

REPO="W-Nana/remnawave-node-go"
BINARY_NAME="remnawave-node-go"
INSTALL_DIR="/etc/remnawave-node"
SERVICE_DIR="/etc/systemd/system"

GEOIP_URL="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
GEOSITE_URL="https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

show_help() {
    cat << EOF
Remnawave Node Go - Installation Script

Usage:
  bash install.sh [command] [options]

Commands:
  install     Install the node (default)
  uninstall   Remove the node
  update      Update to latest version
  update-geo  Update geoip/geosite data files

Options:
  -s, --secret <SECRET>   Panel secret/token (required for install)
  -p, --port <PORT>       Node port (default: 3000)
  -h, --host <HOST>       Panel host URL (e.g., https://panel.example.com)
  -v, --version <VER>     Install specific version (e.g., v1.0.0)
  --help                  Show this help message

Examples:
  # Install with panel secret and port
  bash <(curl -sL https://raw.githubusercontent.com/${REPO}/main/install.sh) -s YOUR_SECRET -p 3000

  # Install with full options
  bash install.sh install -s SECRET -p 3000 -h https://panel.example.com

  # Update node
  bash install.sh update

  # Update geo data files only
  bash install.sh update-geo

  # Uninstall
  bash install.sh uninstall

EOF
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root. Use: sudo bash install.sh"
    fi
}

detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux) echo "linux" ;;
        darwin) echo "darwin" ;;
        freebsd) echo "freebsd" ;;
        *) log_error "Unsupported OS: $os" ;;
    esac
}

detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        armv7l|armv7) echo "arm" ;;
        *) log_error "Unsupported architecture: $arch" ;;
    esac
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$version" ]]; then
        log_error "Failed to get latest version. Check your network connection."
    fi
    echo "$version"
}

download_geo_files() {
    log_info "Downloading geo data files..."
    
    mkdir -p "$INSTALL_DIR"
    
    local tmp_dir
    tmp_dir=$(mktemp -d)
    
    if curl -fsSL "$GEOIP_URL" -o "${tmp_dir}/geoip.dat"; then
        mv "${tmp_dir}/geoip.dat" "${INSTALL_DIR}/geoip.dat"
        log_success "Downloaded geoip.dat"
    else
        log_warn "Failed to download geoip.dat"
    fi
    
    if curl -fsSL "$GEOSITE_URL" -o "${tmp_dir}/geosite.dat"; then
        mv "${tmp_dir}/geosite.dat" "${INSTALL_DIR}/geosite.dat"
        log_success "Downloaded geosite.dat"
    else
        log_warn "Failed to download geosite.dat"
    fi
    
    rm -rf "$tmp_dir"
}

setup_geo_update_timer() {
    if ! command -v systemctl &>/dev/null; then
        return
    fi
    
    cat > "${SERVICE_DIR}/remnawave-geo-update.service" << EOF
[Unit]
Description=Update Remnawave geo data files
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=/bin/bash -c 'curl -fsSL ${GEOIP_URL} -o ${INSTALL_DIR}/geoip.dat && curl -fsSL ${GEOSITE_URL} -o ${INSTALL_DIR}/geosite.dat && systemctl restart ${BINARY_NAME} 2>/dev/null || true'
EOF

    cat > "${SERVICE_DIR}/remnawave-geo-update.timer" << EOF
[Unit]
Description=Weekly update of Remnawave geo data files

[Timer]
OnCalendar=weekly
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
EOF

    systemctl daemon-reload
    systemctl enable --now remnawave-geo-update.timer 2>/dev/null || true
    log_success "Geo auto-update timer installed (weekly)"
}

download_binary() {
    local version=$1
    local os=$2
    local arch=$3
    
    local filename="remnawave-node-go-${version}-${os}-${arch}.tar.gz"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"
    local tmp_dir
    tmp_dir=$(mktemp -d)
    
    log_info "Downloading ${filename}..."
    
    if ! curl -fsSL "$url" -o "${tmp_dir}/${filename}"; then
        rm -rf "$tmp_dir"
        log_error "Failed to download from: $url"
    fi
    
    log_info "Extracting..."
    tar -xzf "${tmp_dir}/${filename}" -C "$tmp_dir"
    
    if systemctl is-active --quiet ${BINARY_NAME} 2>/dev/null; then
        log_info "Stopping existing service..."
        systemctl stop ${BINARY_NAME}
    fi
    
    mkdir -p "$INSTALL_DIR"
    mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    
    rm -rf "$tmp_dir"
    log_success "Binary installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

create_config() {
    local secret="${1:-}"
    local port="${2:-3000}"
    local host="${3:-}"
    
    mkdir -p "$INSTALL_DIR"
    
    if [[ -f "${INSTALL_DIR}/.env" ]] && [[ -z "$secret" ]]; then
        log_info "Existing config found, preserving..."
        return
    fi
    
    if [[ -z "$secret" ]]; then
        log_warn "No secret provided. You must edit ${INSTALL_DIR}/.env manually."
        cat > "${INSTALL_DIR}/.env" << EOF
# Remnawave Node Configuration
APP_PORT=${port}
PANEL_HOST=${host:-https://your-panel.example.com}
PANEL_TOKEN=your-node-token-here

# Geo data location
XRAY_LOCATION_ASSET=${INSTALL_DIR}
EOF
    else
        cat > "${INSTALL_DIR}/.env" << EOF
# Remnawave Node Configuration
APP_PORT=${port}
PANEL_HOST=${host}
PANEL_TOKEN=${secret}

# Geo data location
XRAY_LOCATION_ASSET=${INSTALL_DIR}
EOF
        log_success "Config created with provided credentials"
    fi
    
    chmod 600 "${INSTALL_DIR}/.env"
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
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/${BINARY_NAME}
Restart=on-failure
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable ${BINARY_NAME}
    log_success "Systemd service installed and enabled"
}

show_status() {
    local secret="${1:-}"
    
    echo ""
    echo "========================================"
    log_success "Installation complete!"
    echo "========================================"
    echo ""
    echo "Install directory: ${INSTALL_DIR}/"
    echo "  - Binary:        ${INSTALL_DIR}/${BINARY_NAME}"
    echo "  - Config:        ${INSTALL_DIR}/.env"
    echo "  - Geo data:      ${INSTALL_DIR}/geoip.dat, geosite.dat"
    echo ""
    
    if [[ -n "$secret" ]]; then
        echo "Service commands:"
        echo "  Start:   systemctl start ${BINARY_NAME}"
        echo "  Status:  systemctl status ${BINARY_NAME}"
        echo "  Logs:    journalctl -u ${BINARY_NAME} -f"
        echo ""
        log_info "Starting service..."
        systemctl start ${BINARY_NAME}
        sleep 2
        if systemctl is-active --quiet ${BINARY_NAME}; then
            log_success "Service is running!"
        else
            log_warn "Service may have failed to start. Check: journalctl -u ${BINARY_NAME} -e"
        fi
    else
        echo "Next steps:"
        echo "  1. Edit config:  nano ${INSTALL_DIR}/.env"
        echo "  2. Start:        systemctl start ${BINARY_NAME}"
        echo "  3. Check logs:   journalctl -u ${BINARY_NAME} -f"
    fi
    echo ""
}

do_install() {
    local secret="$1"
    local port="$2"
    local host="$3"
    local version="$4"
    
    check_root
    
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    
    log_info "Detected: ${os}/${arch}"
    
    if [[ -z "$version" ]]; then
        version=$(get_latest_version)
    fi
    
    log_info "Installing version: ${version}"
    
    download_binary "$version" "$os" "$arch"
    download_geo_files
    create_config "$secret" "$port" "$host"
    
    if [[ "$os" == "linux" ]] && command -v systemctl &>/dev/null; then
        install_systemd_service
        setup_geo_update_timer
    fi
    
    show_status "$secret"
}

do_uninstall() {
    check_root
    
    log_info "Uninstalling ${BINARY_NAME}..."
    
    systemctl stop ${BINARY_NAME} 2>/dev/null || true
    systemctl disable ${BINARY_NAME} 2>/dev/null || true
    systemctl stop remnawave-geo-update.timer 2>/dev/null || true
    systemctl disable remnawave-geo-update.timer 2>/dev/null || true
    
    rm -f "${SERVICE_DIR}/${BINARY_NAME}.service"
    rm -f "${SERVICE_DIR}/remnawave-geo-update.service"
    rm -f "${SERVICE_DIR}/remnawave-geo-update.timer"
    systemctl daemon-reload
    
    log_success "Uninstalled successfully"
    log_info "Data preserved at: ${INSTALL_DIR}"
    echo ""
    echo "To remove all data: rm -rf ${INSTALL_DIR}"
}

do_update() {
    local version="$1"
    
    check_root
    
    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    
    if [[ -z "$version" ]]; then
        version=$(get_latest_version)
    fi
    
    log_info "Updating to version: ${version}"
    
    download_binary "$version" "$os" "$arch"
    download_geo_files
    
    if systemctl is-enabled --quiet ${BINARY_NAME} 2>/dev/null; then
        systemctl restart ${BINARY_NAME}
        log_success "Service restarted"
    fi
}

do_update_geo() {
    check_root
    download_geo_files
    
    if systemctl is-active --quiet ${BINARY_NAME} 2>/dev/null; then
        systemctl restart ${BINARY_NAME}
        log_success "Service restarted with new geo data"
    fi
}

main() {
    local cmd="install"
    local secret=""
    local port="3000"
    local host=""
    local version=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            install|uninstall|update|update-geo)
                cmd="$1"
                shift
                ;;
            -s|--secret)
                secret="$2"
                shift 2
                ;;
            -p|--port)
                port="$2"
                shift 2
                ;;
            -h|--host)
                host="$2"
                shift 2
                ;;
            -v|--version)
                version="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1. Use --help for usage."
                ;;
        esac
    done
    
    case "$cmd" in
        install)
            do_install "$secret" "$port" "$host" "$version"
            ;;
        uninstall)
            do_uninstall
            ;;
        update)
            do_update "$version"
            ;;
        update-geo)
            do_update_geo
            ;;
    esac
}

main "$@"
