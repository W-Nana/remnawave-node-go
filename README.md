# Remnawave Node Go

[繁體中文](README_zh.md)

High-performance Go rewrite of [Remnawave Node](https://github.com/remnawave/node) with embedded xray-core. This node connects to the Remnawave panel and provides proxy services with VLESS, Trojan, and Shadowsocks support.

## Features

- **Embedded xray-core** - No external xray binary required
- **Multi-protocol** - VLESS, Trojan, Shadowsocks
- **Real-time management** - Add/remove users without restart
- **Traffic statistics** - Per-user, per-inbound, per-outbound stats
- **Auto geo updates** - Weekly automatic geoip/geosite updates
- **Single directory** - Everything in `/etc/remnawave-node`

## Quick Install

```bash
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) -s <SECRET_KEY> -p <NODE_PORT>
```

### Parameters

| Parameter | Description |
|-----------|-------------|
| `-s, --secret` | SECRET_KEY from panel (required) |
| `-p, --port` | Node port (default: 3000) |
| `-v, --version` | Specific version |

### Examples

```bash
# Install with secret key and port
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) -s YOUR_SECRET_KEY -p 3000

# Update to latest
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) update

# Update geo files only
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) update-geo

# Uninstall
bash <(curl -sL https://raw.githubusercontent.com/W-Nana/remnawave-node-go/main/install.sh) uninstall
```

## Installation Directory

Everything is installed to `/etc/remnawave-node/`:

```
/etc/remnawave-node/
├── remnawave-node-go    # Binary
├── .env                 # Configuration
├── geoip.dat           # GeoIP data (auto-updated weekly)
└── geosite.dat         # GeoSite data (auto-updated weekly)
```

## Service Management

```bash
# Start
systemctl start remnawave-node-go

# Stop
systemctl stop remnawave-node-go

# Status
systemctl status remnawave-node-go

# View logs
journalctl -u remnawave-node-go -f

# Enable on boot
systemctl enable remnawave-node-go
```

## Configuration

Edit `/etc/remnawave-node/.env`:

```bash
SECRET_KEY=your-secret-key-here
NODE_PORT=3000
XRAY_LOCATION_ASSET=/etc/remnawave-node
```

## Build from Source

```bash
# Clone
git clone https://github.com/W-Nana/remnawave-node-go.git
cd remnawave-node-go

# Build
make build

# Run
./remnawave-node-go
```

## API Endpoints

### Main Server (mTLS + JWT)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/node/xray/start` | Start xray with config |
| `GET` | `/node/xray/stop` | Stop xray |
| `GET` | `/node/xray/status` | Get status |
| `GET` | `/node/xray/healthcheck` | Health check |
| `POST` | `/node/handler/add-user` | Add user |
| `POST` | `/node/handler/add-users` | Bulk add users |
| `POST` | `/node/handler/remove-user` | Remove user |
| `POST` | `/node/handler/remove-users` | Bulk remove users |
| `POST` | `/node/stats/get-users-stats` | Get user stats |
| `GET` | `/node/stats/get-system-stats` | Get system stats |

### Internal Server (localhost only)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/internal/get-config` | Get xray config |
| `POST` | `/vision/block-ip` | Block IP |
| `POST` | `/vision/unblock-ip` | Unblock IP |

## Credits

This project is a Go rewrite of the original [Remnawave Node](https://github.com/remnawave/node) (TypeScript/NestJS).

## License

See LICENSE file.
