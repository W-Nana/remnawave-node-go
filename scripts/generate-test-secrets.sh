#!/bin/bash
#
# Generate test secrets for remnawave-node-go development
#
# This script generates:
# - CA certificate and key
# - Node certificate signed by CA
# - Client certificate for mTLS
# - JWT RS256 keypair
# - SECRET_KEY (base64 encoded JSON)
# - TEST_JWT for API testing
# - env.sh with all environment variables
#
# Dependencies: openssl, jq, python3 with cryptography library

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${PROJECT_ROOT}/test-secrets"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check dependencies
check_dependencies() {
    local missing=()
    
    if ! command -v openssl &> /dev/null; then
        missing+=("openssl")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing+=("jq")
    fi
    
    if ! command -v python3 &> /dev/null; then
        missing+=("python3")
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        echo "Please install them and try again."
        exit 1
    fi
    
    # Check for cryptography library
    if ! python3 -c "import cryptography" 2>/dev/null; then
        log_warn "Python cryptography library not found."
        echo "Install with: pip3 install cryptography"
        exit 1
    fi
}

# Create output directory
setup_output_dir() {
    if [ -d "$OUTPUT_DIR" ]; then
        log_warn "Output directory exists, cleaning up..."
        rm -rf "$OUTPUT_DIR"
    fi
    mkdir -p "$OUTPUT_DIR"
    log_info "Created output directory: $OUTPUT_DIR"
}

# Generate CA certificate
generate_ca() {
    log_info "Generating CA certificate..."
    
    openssl genrsa -out "${OUTPUT_DIR}/ca.key" 4096 2>/dev/null
    
    openssl req -new -x509 \
        -days 3650 \
        -key "${OUTPUT_DIR}/ca.key" \
        -out "${OUTPUT_DIR}/ca.crt" \
        -subj "/C=US/ST=State/L=City/O=Remnawave/OU=CA/CN=Remnawave CA" \
        2>/dev/null
    
    log_info "CA certificate generated: ca.crt, ca.key"
}

# Generate node certificate signed by CA
generate_node_cert() {
    log_info "Generating node certificate..."
    
    openssl genrsa -out "${OUTPUT_DIR}/node.key" 2048 2>/dev/null
    
    openssl req -new \
        -key "${OUTPUT_DIR}/node.key" \
        -out "${OUTPUT_DIR}/node.csr" \
        -subj "/C=US/ST=State/L=City/O=Remnawave/OU=Node/CN=localhost" \
        2>/dev/null
    
    # Create extension file for SAN
    cat > "${OUTPUT_DIR}/node_ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
    
    openssl x509 -req \
        -in "${OUTPUT_DIR}/node.csr" \
        -CA "${OUTPUT_DIR}/ca.crt" \
        -CAkey "${OUTPUT_DIR}/ca.key" \
        -CAcreateserial \
        -out "${OUTPUT_DIR}/node.crt" \
        -days 365 \
        -extfile "${OUTPUT_DIR}/node_ext.cnf" \
        2>/dev/null
    
    rm -f "${OUTPUT_DIR}/node.csr" "${OUTPUT_DIR}/node_ext.cnf" "${OUTPUT_DIR}/ca.srl"
    
    log_info "Node certificate generated: node.crt, node.key"
}

# Generate client certificate for mTLS
generate_client_cert() {
    log_info "Generating client certificate..."
    
    openssl genrsa -out "${OUTPUT_DIR}/client.key" 2048 2>/dev/null
    
    openssl req -new \
        -key "${OUTPUT_DIR}/client.key" \
        -out "${OUTPUT_DIR}/client.csr" \
        -subj "/C=US/ST=State/L=City/O=Remnawave/OU=Client/CN=remnawave-backend" \
        2>/dev/null
    
    cat > "${OUTPUT_DIR}/client_ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature
extendedKeyUsage = clientAuth
EOF
    
    openssl x509 -req \
        -in "${OUTPUT_DIR}/client.csr" \
        -CA "${OUTPUT_DIR}/ca.crt" \
        -CAkey "${OUTPUT_DIR}/ca.key" \
        -CAcreateserial \
        -out "${OUTPUT_DIR}/client.crt" \
        -days 365 \
        -extfile "${OUTPUT_DIR}/client_ext.cnf" \
        2>/dev/null
    
    rm -f "${OUTPUT_DIR}/client.csr" "${OUTPUT_DIR}/client_ext.cnf" "${OUTPUT_DIR}/ca.srl"
    
    log_info "Client certificate generated: client.crt, client.key"
}

# Generate JWT RS256 keypair
generate_jwt_keys() {
    log_info "Generating JWT RS256 keypair..."
    
    openssl genrsa -out "${OUTPUT_DIR}/jwt.key" 2048 2>/dev/null
    openssl rsa -in "${OUTPUT_DIR}/jwt.key" -pubout -out "${OUTPUT_DIR}/jwt.pub" 2>/dev/null
    
    log_info "JWT keys generated: jwt.key, jwt.pub"
}

# Generate SECRET_KEY (base64 encoded JSON)
generate_secret_key() {
    log_info "Generating SECRET_KEY..."
    
    local ca_cert=$(cat "${OUTPUT_DIR}/ca.crt")
    local jwt_pub=$(cat "${OUTPUT_DIR}/jwt.pub")
    local node_cert=$(cat "${OUTPUT_DIR}/node.crt")
    local node_key=$(cat "${OUTPUT_DIR}/node.key")
    
    # Create JSON payload using jq for proper escaping
    local payload=$(jq -n \
        --arg caCertPem "$ca_cert" \
        --arg jwtPublicKey "$jwt_pub" \
        --arg nodeCertPem "$node_cert" \
        --arg nodeKeyPem "$node_key" \
        '{caCertPem: $caCertPem, jwtPublicKey: $jwtPublicKey, nodeCertPem: $nodeCertPem, nodeKeyPem: $nodeKeyPem}')
    
    # Base64 encode
    local secret_key=$(echo -n "$payload" | base64 -w 0)
    
    echo "$secret_key" > "${OUTPUT_DIR}/secret_key.txt"
    
    log_info "SECRET_KEY generated: secret_key.txt"
}

# Generate TEST_JWT using Python
generate_test_jwt() {
    log_info "Generating TEST_JWT..."
    
    python3 << 'PYTHON_SCRIPT'
import json
import time
import base64
import sys
import os

try:
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import padding
    from cryptography.hazmat.backends import default_backend
except ImportError:
    print("Error: cryptography library not found", file=sys.stderr)
    sys.exit(1)

def base64url_encode(data):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode('utf-8')

def create_jwt(private_key_path):
    # Read private key
    with open(private_key_path, 'rb') as f:
        private_key = serialization.load_pem_private_key(
            f.read(),
            password=None,
            backend=default_backend()
        )
    
    # Create header
    header = {
        "alg": "RS256",
        "typ": "JWT"
    }
    
    # Create payload with long expiration for testing
    now = int(time.time())
    payload = {
        "sub": "remnawave-backend",
        "iat": now,
        "exp": now + (365 * 24 * 60 * 60),  # 1 year
        "iss": "remnawave",
        "aud": "remnawave-node"
    }
    
    # Encode header and payload
    header_b64 = base64url_encode(json.dumps(header, separators=(',', ':')))
    payload_b64 = base64url_encode(json.dumps(payload, separators=(',', ':')))
    
    # Create signature
    message = f"{header_b64}.{payload_b64}".encode('utf-8')
    
    signature = private_key.sign(
        message,
        padding.PKCS1v15(),
        hashes.SHA256()
    )
    
    signature_b64 = base64url_encode(signature)
    
    return f"{header_b64}.{payload_b64}.{signature_b64}"

if __name__ == "__main__":
    script_dir = os.path.dirname(os.path.abspath(__file__)) if '__file__' in dir() else os.getcwd()
    # Look for jwt.key in test-secrets directory
    project_root = os.environ.get('PROJECT_ROOT', os.getcwd())
    jwt_key_path = os.path.join(project_root, 'test-secrets', 'jwt.key')
    
    if not os.path.exists(jwt_key_path):
        # Try relative path
        jwt_key_path = 'test-secrets/jwt.key'
    
    jwt = create_jwt(jwt_key_path)
    print(jwt)
PYTHON_SCRIPT

    local test_jwt
    test_jwt=$(PROJECT_ROOT="$PROJECT_ROOT" python3 << 'PYTHON_SCRIPT'
import json
import time
import base64
import sys
import os

try:
    from cryptography.hazmat.primitives import hashes, serialization
    from cryptography.hazmat.primitives.asymmetric import padding
    from cryptography.hazmat.backends import default_backend
except ImportError:
    print("Error: cryptography library not found", file=sys.stderr)
    sys.exit(1)

def base64url_encode(data):
    if isinstance(data, str):
        data = data.encode('utf-8')
    return base64.urlsafe_b64encode(data).rstrip(b'=').decode('utf-8')

def create_jwt(private_key_path):
    with open(private_key_path, 'rb') as f:
        private_key = serialization.load_pem_private_key(
            f.read(),
            password=None,
            backend=default_backend()
        )
    
    header = {"alg": "RS256", "typ": "JWT"}
    now = int(time.time())
    payload = {
        "sub": "remnawave-backend",
        "iat": now,
        "exp": now + (365 * 24 * 60 * 60),
        "iss": "remnawave",
        "aud": "remnawave-node"
    }
    
    header_b64 = base64url_encode(json.dumps(header, separators=(',', ':')))
    payload_b64 = base64url_encode(json.dumps(payload, separators=(',', ':')))
    message = f"{header_b64}.{payload_b64}".encode('utf-8')
    
    signature = private_key.sign(message, padding.PKCS1v15(), hashes.SHA256())
    signature_b64 = base64url_encode(signature)
    
    return f"{header_b64}.{payload_b64}.{signature_b64}"

project_root = os.environ.get('PROJECT_ROOT', os.getcwd())
jwt_key_path = os.path.join(project_root, 'test-secrets', 'jwt.key')
print(create_jwt(jwt_key_path))
PYTHON_SCRIPT
)
    
    echo "$test_jwt" > "${OUTPUT_DIR}/test_jwt.txt"
    
    log_info "TEST_JWT generated: test_jwt.txt"
}

# Generate env.sh with all environment variables
generate_env_file() {
    log_info "Generating env.sh..."
    
    local secret_key=$(cat "${OUTPUT_DIR}/secret_key.txt")
    local test_jwt=$(cat "${OUTPUT_DIR}/test_jwt.txt")
    
    cat > "${OUTPUT_DIR}/env.sh" << EOF
#!/bin/bash
# Environment variables for remnawave-node-go testing
# Source this file: source test-secrets/env.sh

export SECRET_KEY='${secret_key}'
export NODE_PORT=2222
export INTERNAL_REST_PORT=61001
export LOG_LEVEL=debug
export TEST_JWT='${test_jwt}'

# Certificate paths (for curl testing)
export CA_CERT='${OUTPUT_DIR}/ca.crt'
export CLIENT_CERT='${OUTPUT_DIR}/client.crt'
export CLIENT_KEY='${OUTPUT_DIR}/client.key'

echo "Environment variables loaded:"
echo "  NODE_PORT=\$NODE_PORT"
echo "  INTERNAL_REST_PORT=\$INTERNAL_REST_PORT"
echo "  LOG_LEVEL=\$LOG_LEVEL"
echo "  SECRET_KEY=(set)"
echo "  TEST_JWT=(set)"
EOF
    
    chmod +x "${OUTPUT_DIR}/env.sh"
    
    log_info "env.sh generated: source test-secrets/env.sh to use"
}

# Main
main() {
    echo "========================================"
    echo "  Remnawave Node Go - Secret Generator"
    echo "========================================"
    echo ""
    
    check_dependencies
    setup_output_dir
    generate_ca
    generate_node_cert
    generate_client_cert
    generate_jwt_keys
    generate_secret_key
    generate_test_jwt
    generate_env_file
    
    echo ""
    echo "========================================"
    log_info "All secrets generated successfully!"
    echo "========================================"
    echo ""
    echo "Output directory: ${OUTPUT_DIR}"
    echo ""
    echo "Generated files:"
    ls -la "${OUTPUT_DIR}"
    echo ""
    echo "Usage:"
    echo "  source ${OUTPUT_DIR}/env.sh"
    echo "  make run"
    echo ""
    echo "Test with curl:"
    echo "  curl --cacert \$CA_CERT --cert \$CLIENT_CERT --key \$CLIENT_KEY \\"
    echo "       -H \"Authorization: Bearer \$TEST_JWT\" \\"
    echo "       https://localhost:2222/node/xray/healthcheck"
}

main "$@"
