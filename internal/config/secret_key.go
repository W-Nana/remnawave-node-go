package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrSecretKeyEmpty         = errors.New("SECRET_KEY is not set")
	ErrSecretKeyInvalidBase64 = errors.New("SECRET_KEY contains invalid base64")
	ErrSecretKeyInvalidJSON   = errors.New("SECRET_KEY contains invalid JSON")
	ErrSecretKeyMissingField  = errors.New("SECRET_KEY payload missing required field")
)

type NodePayload struct {
	CACertPEM    string `json:"caCertPem"`
	JWTPublicKey string `json:"jwtPublicKey"`
	NodeCertPEM  string `json:"nodeCertPem"`
	NodeKeyPEM   string `json:"nodeKeyPem"`
}

func ParseSecretKey(base64Str string) (*NodePayload, error) {
	if base64Str == "" {
		return nil, ErrSecretKeyEmpty
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSecretKeyInvalidBase64, err)
	}

	var payload NodePayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSecretKeyInvalidJSON, err)
	}

	if err := validateNodePayload(&payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func validateNodePayload(p *NodePayload) error {
	if p.CACertPEM == "" {
		return fmt.Errorf("%w: caCertPem", ErrSecretKeyMissingField)
	}
	if p.JWTPublicKey == "" {
		return fmt.Errorf("%w: jwtPublicKey", ErrSecretKeyMissingField)
	}
	if p.NodeCertPEM == "" {
		return fmt.Errorf("%w: nodeCertPem", ErrSecretKeyMissingField)
	}
	if p.NodeKeyPEM == "" {
		return fmt.Errorf("%w: nodeKeyPem", ErrSecretKeyMissingField)
	}
	return nil
}
