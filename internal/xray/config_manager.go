package xray

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/remnawave/node-go/internal/logger"
)

// InboundHash represents the hash information for a single inbound.
type InboundHash struct {
	Tag        string `json:"tag"`
	Hash       string `json:"hash"`
	UsersCount int    `json:"usersCount"`
}

// Hashes represents the hash payload from the start command.
type Hashes struct {
	EmptyConfig string        `json:"emptyConfig"`
	Inbounds    []InboundHash `json:"inbounds"`
}

// Internals represents the internal configuration from start command.
type Internals struct {
	ForceRestart bool   `json:"forceRestart"`
	Hashes       Hashes `json:"hashes"`
}

// InboundSettings represents the settings section of an inbound.
type InboundSettings struct {
	Clients []struct {
		ID string `json:"id"`
	} `json:"clients"`
}

// Inbound represents an inbound configuration from xray config.
type Inbound struct {
	Tag      string          `json:"tag"`
	Settings InboundSettings `json:"settings"`
}

// XrayConfig represents the xray configuration structure.
// Only fields needed for user extraction are defined.
type XrayConfig struct {
	Inbounds []Inbound `json:"inbounds"`
}

// ConfigManager manages xray configuration state and hash-based restart logic.
// It tracks user hashes per inbound to determine if core restart is needed.
type ConfigManager struct {
	mu                 sync.RWMutex
	xrayConfig         map[string]interface{}
	emptyConfigHash    string
	inboundsHashMap    map[string]*HashedSet
	xtlsConfigInbounds map[string]struct{}
	log                *logger.Logger
}

// NewConfigManager creates a new ConfigManager instance.
func NewConfigManager(log *logger.Logger) *ConfigManager {
	return &ConfigManager{
		xrayConfig:         nil,
		emptyConfigHash:    "",
		inboundsHashMap:    make(map[string]*HashedSet),
		xtlsConfigInbounds: make(map[string]struct{}),
		log:                log,
	}
}

// GetXrayConfig returns the current xray configuration.
func (m *ConfigManager) GetXrayConfig() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.xrayConfig == nil {
		return map[string]interface{}{}
	}
	return m.xrayConfig
}

// SetXrayConfig sets the xray configuration.
func (m *ConfigManager) SetXrayConfig(config map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.xrayConfig = config
}

// IsNeedRestartCore determines if xray-core needs to be restarted based on hash comparison.
// Returns true if restart is needed, false otherwise.
//
// Restart conditions:
// 1. emptyConfigHash is empty (first start)
// 2. incoming emptyConfig differs from stored (base config changed)
// 3. number of inbounds changed
// 4. any inbound tag no longer exists
// 5. any inbound user hash changed
func (m *ConfigManager) IsNeedRestartCore(incomingHashes Hashes) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Condition 1: First start
	if m.emptyConfigHash == "" {
		return true
	}

	// Condition 2: Base config changed
	if incomingHashes.EmptyConfig != m.emptyConfigHash {
		if m.log != nil {
			m.log.Warn("Detected changes in Xray Core base configuration")
		}
		return true
	}

	// Condition 3: Number of inbounds changed
	if len(incomingHashes.Inbounds) != len(m.inboundsHashMap) {
		if m.log != nil {
			m.log.Warn("Number of Xray Core inbounds has changed")
		}
		return true
	}

	// Condition 4 & 5: Check each stored inbound
	for inboundTag, usersSet := range m.inboundsHashMap {
		// Find matching incoming inbound
		var incomingInbound *InboundHash
		for i := range incomingHashes.Inbounds {
			if incomingHashes.Inbounds[i].Tag == inboundTag {
				incomingInbound = &incomingHashes.Inbounds[i]
				break
			}
		}

		// Condition 4: Inbound no longer exists
		if incomingInbound == nil {
			if m.log != nil {
				m.log.WithField("inbound", inboundTag).
					Warn("Inbound no longer exists in Xray Core configuration")
			}
			return true
		}

		// Condition 5: User hash changed
		if usersSet.Hash64String() != incomingInbound.Hash {
			if m.log != nil {
				m.log.WithField("inbound", inboundTag).
					WithField("current", usersSet.Hash64String()).
					WithField("incoming", incomingInbound.Hash).
					Warn("User configuration changed for inbound")
			}
			return true
		}
	}

	if m.log != nil {
		m.log.Info("Xray Core configuration is up-to-date - no restart required")
	}

	return false
}

// ExtractUsersFromConfig extracts users from the xray config and updates hash maps.
// This should be called after a successful xray-core start.
func (m *ConfigManager) ExtractUsersFromConfig(hashes Hashes, newConfig map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cleanup existing state
	m.cleanup()

	m.emptyConfigHash = hashes.EmptyConfig
	m.xrayConfig = newConfig

	if m.log != nil {
		hashJSON, _ := json.Marshal(hashes)
		m.log.Info(fmt.Sprintf("Starting user extraction from inbounds... Hash payload: %s", string(hashJSON)))
	}

	// Extract inbounds from config
	inboundsRaw, ok := newConfig["inbounds"]
	if !ok {
		return nil
	}

	inboundsSlice, ok := inboundsRaw.([]interface{})
	if !ok {
		return nil
	}

	// Build set of valid tags from hashes
	validTags := make(map[string]struct{})
	for _, h := range hashes.Inbounds {
		validTags[h.Tag] = struct{}{}
	}

	// Process each inbound
	for _, inboundRaw := range inboundsSlice {
		inbound, ok := inboundRaw.(map[string]interface{})
		if !ok {
			continue
		}

		tag, ok := inbound["tag"].(string)
		if !ok || tag == "" {
			continue
		}

		// Skip if not in valid tags
		if _, valid := validTags[tag]; !valid {
			continue
		}

		usersSet := NewHashedSet()

		// Extract clients
		if settings, ok := inbound["settings"].(map[string]interface{}); ok {
			if clients, ok := settings["clients"].([]interface{}); ok {
				for _, clientRaw := range clients {
					if client, ok := clientRaw.(map[string]interface{}); ok {
						if id, ok := client["id"].(string); ok && id != "" {
							usersSet.Add(id)
						}
					}
				}
			}
		}

		m.inboundsHashMap[tag] = usersSet
		m.xtlsConfigInbounds[tag] = struct{}{}

		if m.log != nil {
			m.log.Info(fmt.Sprintf("%s has %d users", tag, usersSet.Size()))
		}
	}

	return nil
}

// AddUserToInbound adds a user to the specified inbound's hash set.
func (m *ConfigManager) AddUserToInbound(inboundTag, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	usersSet, exists := m.inboundsHashMap[inboundTag]
	if !exists {
		if m.log != nil {
			m.log.WithField("inbound", inboundTag).
				Warn("Inbound not found in inboundsHashMap, creating new one")
		}
		usersSet = NewHashedSet()
		usersSet.Add(userID)
		m.inboundsHashMap[inboundTag] = usersSet
		return
	}

	usersSet.Add(userID)
}

// RemoveUserFromInbound removes a user from the specified inbound's hash set.
func (m *ConfigManager) RemoveUserFromInbound(inboundTag, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	usersSet, exists := m.inboundsHashMap[inboundTag]
	if !exists {
		return
	}

	usersSet.Delete(userID)

	// Remove inbound if no users left
	if usersSet.Size() == 0 {
		delete(m.xtlsConfigInbounds, inboundTag)
		delete(m.inboundsHashMap, inboundTag)

		if m.log != nil {
			m.log.WithField("inbound", inboundTag).
				Warn("Inbound has no users, clearing from inboundsHashMap")
		}
	}
}

// GetXtlsConfigInbounds returns the set of inbound tags.
func (m *ConfigManager) GetXtlsConfigInbounds() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tags := make([]string, 0, len(m.xtlsConfigInbounds))
	for tag := range m.xtlsConfigInbounds {
		tags = append(tags, tag)
	}
	return tags
}

// GetInboundHash returns the current hash for an inbound, or empty string if not found.
func (m *ConfigManager) GetInboundHash(inboundTag string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if usersSet, exists := m.inboundsHashMap[inboundTag]; exists {
		return usersSet.Hash64String()
	}
	return ""
}

// Cleanup clears all internal state.
func (m *ConfigManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanup()
}

// cleanup clears all internal state (no lock, internal use).
func (m *ConfigManager) cleanup() {
	if m.log != nil {
		m.log.Info("Cleaning up config manager")
	}

	m.inboundsHashMap = make(map[string]*HashedSet)
	m.xtlsConfigInbounds = make(map[string]struct{})
	m.xrayConfig = nil
	m.emptyConfigHash = ""
}
