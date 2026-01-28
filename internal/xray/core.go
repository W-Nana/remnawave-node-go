package xray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/remnawave/node-go/internal/logger"
)

func init() {
	if os.Getenv("XRAY_LOCATION_ASSET") == "" {
		for _, path := range []string{
			"/usr/local/share/xray",
			"/usr/share/xray",
			"/opt/xray",
			".",
		} {
			if _, err := os.Stat(path + "/geoip.dat"); err == nil {
				os.Setenv("XRAY_LOCATION_ASSET", path)
				break
			}
		}
	}
}

type Core struct {
	mu       sync.RWMutex
	instance *core.Instance
	logger   *logger.Logger
	running  bool
}

func NewCore(log *logger.Logger) *Core {
	return &Core{
		logger: log,
	}
}

func (c *Core) Start(configJSON []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		if err := c.stopLocked(); err != nil {
			return fmt.Errorf("failed to stop existing instance: %w", err)
		}
	}

	config, err := core.LoadConfig("json", bytes.NewReader(configJSON))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	instance, err := core.New(config)
	if err != nil {
		return fmt.Errorf("failed to create xray instance: %w", err)
	}

	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("failed to start xray: %w", err)
	}

	c.instance = instance
	c.running = true
	c.logger.Info("xray-core started successfully")

	return nil
}

func (c *Core) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopLocked()
}

func (c *Core) stopLocked() error {
	if c.instance == nil {
		return nil
	}

	if err := c.instance.Close(); err != nil {
		return fmt.Errorf("failed to close xray instance: %w", err)
	}

	c.instance = nil
	c.running = false
	c.logger.Info("xray-core stopped")

	return nil
}

func (c *Core) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *Core) GetVersion() string {
	return core.Version()
}

func (c *Core) Instance() *core.Instance {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.instance
}

func (c *Core) Restart(configJSON []byte) error {
	return c.Start(configJSON)
}

func ValidateConfig(configJSON []byte) error {
	var cfg map[string]interface{}
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	_, err := core.LoadConfig("json", bytes.NewReader(configJSON))
	if err != nil {
		return fmt.Errorf("invalid xray config: %w", err)
	}

	return nil
}
