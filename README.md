# Remnawave Node Go

High-performance Go rewrite of Remnawave Node with embedded xray-core. This node connects to the Remnawave backend and provides a unified proxy server with VLESS, Trojan, and Shadowsocks support.

## Features

- **Embedded xray-core**: No external xray binary required
- **mTLS + JWT Authentication**: Secure communication with the Remnawave backend
- **Multi-protocol Support**: VLESS, Trojan, Shadowsocks
- **Real-time User Management**: Add/remove users without restart
- **Traffic Statistics**: Per-user, per-inbound, per-outbound stats
- **IP Blocking**: Vision-based IP blocking capabilities
- **Zstd Compression**: Efficient request/response compression
- **Dual Server Architecture**: External HTTPS server + internal HTTP API

## Installation

### Prerequisites

- Go 1.21 or later
- Git

### Build from Source

```bash
# Clone the repository
git clone https://github.com/remnawave/node-go.git
cd node-go

# Build the binary
make build

# Or install to /usr/local/bin
make install
```

### Systemd Setup

Create `/etc/systemd/system/remnawave-node.service`:

```ini
[Unit]
Description=Remnawave Node Go
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/remnawave-node-go
Restart=always
RestartSec=5
Environment=SECRET_KEY=<your-base64-encoded-secret>
Environment=NODE_PORT=2222
Environment=INTERNAL_REST_PORT=61001
Environment=LOG_LEVEL=info

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable remnawave-node
sudo systemctl start remnawave-node
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SECRET_KEY` | (required) | Base64-encoded JSON containing certificates and keys |
| `NODE_PORT` | `2222` | External HTTPS port (mTLS + JWT) |
| `INTERNAL_REST_PORT` | `61001` | Internal HTTP API port (localhost only) |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

### SECRET_KEY Structure

The `SECRET_KEY` is a base64-encoded JSON object:

```json
{
  "caCertPem": "-----BEGIN CERTIFICATE-----\n...",
  "jwtPublicKey": "-----BEGIN PUBLIC KEY-----\n...",
  "nodeCertPem": "-----BEGIN CERTIFICATE-----\n...",
  "nodeKeyPem": "-----BEGIN PRIVATE KEY-----\n..."
}
```

## Usage

### Quick Start

```bash
# Generate test secrets (development only)
make generate-secrets
source test-secrets/env.sh

# Run the node
make run
```

### Testing the API

```bash
# Health check (requires mTLS + JWT)
curl --cacert ca.crt --cert client.crt --key client.key \
     -H "Authorization: Bearer $TEST_JWT" \
     https://localhost:2222/node/xray/healthcheck

# Internal config endpoint (localhost only)
curl http://127.0.0.1:61001/internal/get-config
```

## API Endpoints

### Main Server (Port 2222 - mTLS + JWT)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/node/xray/start` | Start xray core with configuration |
| `GET` | `/node/xray/stop` | Stop xray core |
| `GET` | `/node/xray/status` | Get xray running status |
| `GET` | `/node/xray/healthcheck` | Health check with version info |
| `POST` | `/node/handler/add-user` | Add single user to inbounds |
| `POST` | `/node/handler/add-users` | Bulk add users |
| `POST` | `/node/handler/remove-user` | Remove single user |
| `POST` | `/node/handler/remove-users` | Bulk remove users |
| `POST` | `/node/handler/get-inbound-users` | List users in inbound |
| `POST` | `/node/handler/get-inbound-users-count` | Count users in inbound |
| `GET` | `/node/stats/get-system-stats` | Get system statistics |
| `POST` | `/node/stats/get-users-stats` | Get all users traffic stats |
| `POST` | `/node/stats/get-user-online-status` | Check if user is online |
| `POST` | `/node/stats/get-inbound-stats` | Get inbound traffic stats |
| `POST` | `/node/stats/get-outbound-stats` | Get outbound traffic stats |
| `POST` | `/node/stats/get-all-inbounds-stats` | Get all inbounds stats |
| `POST` | `/node/stats/get-all-outbounds-stats` | Get all outbounds stats |
| `POST` | `/node/stats/get-combined-stats` | Get combined inbound/outbound stats |

### Internal Server (Port 61001 - localhost only)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/internal/get-config` | Get current xray configuration |
| `POST` | `/vision/block-ip` | Block an IP address |
| `POST` | `/vision/unblock-ip` | Unblock an IP address |

## Development

### Build Commands

```bash
# Build
make build

# Run tests
make test

# Run tests with coverage
make test-cover

# Run linter
make lint

# Clean build artifacts
make clean
```

### Generate Test Secrets

For local development and testing:

```bash
# Generate CA, node, client certificates, and JWT keys
./scripts/generate-test-secrets.sh

# Load environment variables
source test-secrets/env.sh
```

### Project Structure

```
.
├── cmd/
│   └── node-go/
│       └── main.go           # Entry point
├── internal/
│   ├── api/
│   │   ├── controller/       # HTTP handlers
│   │   ├── middleware/       # JWT middleware
│   │   └── server.go         # Server setup
│   ├── config/               # Configuration parsing
│   ├── errors/               # Error definitions
│   ├── logger/               # Structured logging
│   └── xray/                 # Xray core wrapper
├── scripts/
│   └── generate-test-secrets.sh
├── Makefile
└── README.md
```

## License

See LICENSE file for details.
