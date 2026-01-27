package config

import (
	"encoding/json"
	"errors"
	"os"
)

const (
	DefaultNodePort         = 2222
	DefaultInternalRestPort = 61001
	DefaultLogLevel         = "info"
)

var (
	ErrConfigSecretKeyRequired = errors.New("SECRET_KEY environment variable is required")
)

type Config struct {
	SecretKey        string `json:"secretKey"`
	NodePort         int    `json:"nodePort"`
	InternalRestPort int    `json:"internalRestPort"`
	LogLevel         string `json:"logLevel"`

	Payload *NodePayload `json:"-"`
}

func Load() (*Config, error) {
	cfg := &Config{
		NodePort:         DefaultNodePort,
		InternalRestPort: DefaultInternalRestPort,
		LogLevel:         DefaultLogLevel,
	}

	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, err
		}
	}

	loadFromEnv(cfg)

	if cfg.SecretKey == "" {
		return nil, ErrConfigSecretKeyRequired
	}

	payload, err := ParseSecretKey(cfg.SecretKey)
	if err != nil {
		return nil, err
	}
	cfg.Payload = payload

	return cfg, nil
}

func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func loadFromEnv(cfg *Config) {
	if v := os.Getenv("SECRET_KEY"); v != "" {
		cfg.SecretKey = v
	}
	if v := os.Getenv("NODE_PORT"); v != "" {
		if port := parseIntOr(v, 0); port > 0 {
			cfg.NodePort = port
		}
	}
	if v := os.Getenv("INTERNAL_REST_PORT"); v != "" {
		if port := parseIntOr(v, 0); port > 0 {
			cfg.InternalRestPort = port
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
}

func parseIntOr(s string, fallback int) int {
	var n int
	if err := json.Unmarshal([]byte(s), &n); err != nil {
		return fallback
	}
	return n
}
