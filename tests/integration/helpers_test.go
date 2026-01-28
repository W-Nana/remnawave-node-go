package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestCredentials holds all generated test credentials
type TestCredentials struct {
	CACert     []byte
	CAKey      *rsa.PrivateKey
	NodeCert   []byte
	NodeKey    []byte
	ClientCert tls.Certificate
	JWTKey     *rsa.PrivateKey
	JWTPubPEM  string
	SecretKey  string // base64 encoded
}

// GenerateTestCredentials creates all needed test credentials
func GenerateTestCredentials() (*TestCredentials, error) {
	tc := &TestCredentials{}

	// Generate CA key and cert
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	tc.CAKey = caKey

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	tc.CACert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	// Generate node key and cert (signed by CA)
	nodeKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	nodeTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	nodeCertDER, err := x509.CreateCertificate(rand.Reader, nodeTemplate, caTemplate, &nodeKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	tc.NodeCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: nodeCertDER})
	tc.NodeKey = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(nodeKey)})

	// Generate client cert (signed by CA) for mTLS
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "Test Client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	clientCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	clientKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)})

	tc.ClientCert, err = tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		return nil, err
	}

	// Generate JWT key pair
	jwtKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	tc.JWTKey = jwtKey

	jwtPubDER, err := x509.MarshalPKIXPublicKey(&jwtKey.PublicKey)
	if err != nil {
		return nil, err
	}
	tc.JWTPubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: jwtPubDER}))

	// Create SecretKey JSON and base64 encode it
	secretPayload := map[string]string{
		"caCertPem":    string(tc.CACert),
		"jwtPublicKey": tc.JWTPubPEM,
		"nodeCertPem":  string(tc.NodeCert),
		"nodeKeyPem":   string(tc.NodeKey),
	}
	secretJSON, err := json.Marshal(secretPayload)
	if err != nil {
		return nil, err
	}
	tc.SecretKey = base64.StdEncoding.EncodeToString(secretJSON)

	return tc, nil
}

// GenerateJWT creates a valid JWT token
func (tc *TestCredentials) GenerateJWT() (string, error) {
	claims := jwt.MapClaims{
		"sub": "test-node",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tc.JWTKey)
}

// GenerateExpiredJWT creates an expired JWT token
func (tc *TestCredentials) GenerateExpiredJWT() (string, error) {
	claims := jwt.MapClaims{
		"sub": "test-node",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tc.JWTKey)
}

// CreateHTTPClient creates an HTTP client with mTLS
func (tc *TestCredentials) CreateHTTPClient() *http.Client {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(tc.CACert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tc.ClientCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 30 * time.Second,
	}
}

// StartRequestConfig holds the xray configuration for a start request
type StartRequestConfig struct {
	XrayConfig map[string]interface{} `json:"xrayConfig"`
	Internals  StartRequestInternals  `json:"internals"`
}

// StartRequestInternals holds internal configuration for start request
type StartRequestInternals struct {
	ForceRestart bool                    `json:"forceRestart"`
	Hashes       StartRequestHashPayload `json:"hashes"`
}

// StartRequestHashPayload holds the hash payload for start request
type StartRequestHashPayload struct {
	EmptyConfig string             `json:"emptyConfig"`
	Inbounds    []InboundHashEntry `json:"inbounds"`
}

// InboundHashEntry represents a single inbound hash entry
type InboundHashEntry struct {
	Tag        string `json:"tag"`
	Hash       string `json:"hash"`
	UsersCount int    `json:"usersCount"`
}

// CreateMinimalXrayConfig creates a minimal xray configuration for testing
func CreateMinimalXrayConfig() *StartRequestConfig {
	return &StartRequestConfig{
		XrayConfig: map[string]interface{}{
			"log": map[string]interface{}{
				"loglevel": "warning",
			},
			"inbounds": []interface{}{
				map[string]interface{}{
					"tag":      "vless-in",
					"port":     10000,
					"protocol": "vless",
					"settings": map[string]interface{}{
						"clients":    []interface{}{},
						"decryption": "none",
					},
					"streamSettings": map[string]interface{}{
						"network": "tcp",
					},
				},
			},
			"outbounds": []interface{}{
				map[string]interface{}{
					"tag":      "direct",
					"protocol": "freedom",
				},
			},
			"stats": map[string]interface{}{},
		},
		Internals: StartRequestInternals{
			ForceRestart: false,
			Hashes: StartRequestHashPayload{
				EmptyConfig: "a1b2c3d4e5f67890",
				Inbounds: []InboundHashEntry{
					{
						Tag:        "vless-in",
						Hash:       "0000000000000000",
						UsersCount: 0,
					},
				},
			},
		},
	}
}

// AddUserRequest represents a request to add a user
type AddUserRequest struct {
	Data     []AddUserInboundData `json:"data"`
	HashData AddUserHashData      `json:"hashData"`
}

// AddUserInboundData holds inbound data for adding a user
type AddUserInboundData struct {
	Tag      string `json:"tag"`
	Username string `json:"username"`
	Type     string `json:"type"`
	UUID     string `json:"uuid,omitempty"`
	Flow     string `json:"flow,omitempty"`
}

// AddUserHashData holds hash data for add user request
type AddUserHashData struct {
	VlessUUID     string `json:"vlessUuid,omitempty"`
	PrevVlessUUID string `json:"prevVlessUuid,omitempty"`
}
