package xray

import (
	"testing"
)

func TestConfigManager_IsNeedRestartCore_FirstStart(t *testing.T) {
	// Condition 1: emptyConfigHash is empty (first start) → true
	m := NewConfigManager(nil)

	hashes := Hashes{
		EmptyConfig: "abc123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}

	if !m.IsNeedRestartCore(hashes) {
		t.Error("First start should require restart")
	}
}

func TestConfigManager_IsNeedRestartCore_BaseConfigChanged(t *testing.T) {
	// Condition 2: incoming emptyConfig differs → true
	m := NewConfigManager(nil)

	// Setup initial state
	initialHashes := Hashes{
		EmptyConfig: "original-hash",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(initialHashes, config)

	// Now test with different emptyConfig
	newHashes := Hashes{
		EmptyConfig: "different-hash",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}

	if !m.IsNeedRestartCore(newHashes) {
		t.Error("Changed base config should require restart")
	}
}

func TestConfigManager_IsNeedRestartCore_InboundCountChanged(t *testing.T) {
	// Condition 3: number of inbounds changed → true
	m := NewConfigManager(nil)

	initialHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds: []InboundHash{
			{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0},
			{Tag: "trojan-in", Hash: "0000000000000000", UsersCount: 0},
		},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
			map[string]interface{}{
				"tag":      "trojan-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(initialHashes, config)

	// Now test with fewer inbounds
	newHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}

	if !m.IsNeedRestartCore(newHashes) {
		t.Error("Changed inbound count should require restart")
	}
}

func TestConfigManager_IsNeedRestartCore_InboundNoLongerExists(t *testing.T) {
	// Condition 4: any inbound tag no longer exists → true
	m := NewConfigManager(nil)

	initialHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds: []InboundHash{
			{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0},
		},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(initialHashes, config)

	// Same count but different tag
	newHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "trojan-in", Hash: "0000000000000000", UsersCount: 0}},
	}

	if !m.IsNeedRestartCore(newHashes) {
		t.Error("Missing existing inbound should require restart")
	}
}

func TestConfigManager_IsNeedRestartCore_UserHashChanged(t *testing.T) {
	// Condition 5: any inbound user hash changed → true
	m := NewConfigManager(nil)

	initialHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(initialHashes, config)

	// Same inbound but different hash
	newHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "differenthash123", UsersCount: 1}},
	}

	if !m.IsNeedRestartCore(newHashes) {
		t.Error("Changed user hash should require restart")
	}
}

func TestConfigManager_IsNeedRestartCore_NoRestartNeeded(t *testing.T) {
	// All conditions pass → false (no restart)
	m := NewConfigManager(nil)

	initialHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(initialHashes, config)

	// Same hashes
	sameHashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}

	if m.IsNeedRestartCore(sameHashes) {
		t.Error("Identical config should not require restart")
	}
}

func TestConfigManager_ExtractUsersFromConfig(t *testing.T) {
	m := NewConfigManager(nil)

	hashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds: []InboundHash{
			{Tag: "vless-in", Hash: "somehash", UsersCount: 2},
		},
	}

	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag": "vless-in",
				"settings": map[string]interface{}{
					"clients": []interface{}{
						map[string]interface{}{"id": "uuid-1"},
						map[string]interface{}{"id": "uuid-2"},
					},
				},
			},
			map[string]interface{}{
				"tag": "ignored-inbound", // Not in hashes, should be skipped
				"settings": map[string]interface{}{
					"clients": []interface{}{
						map[string]interface{}{"id": "uuid-3"},
					},
				},
			},
		},
	}

	err := m.ExtractUsersFromConfig(hashes, config)
	if err != nil {
		t.Fatalf("ExtractUsersFromConfig failed: %v", err)
	}

	// Check vless-in was processed
	vlessHash := m.GetInboundHash("vless-in")
	if vlessHash == "" {
		t.Error("vless-in should have a hash")
	}
	if vlessHash == "0000000000000000" {
		t.Error("vless-in hash should not be empty (has 2 users)")
	}

	// Check ignored-inbound was not processed
	ignoredHash := m.GetInboundHash("ignored-inbound")
	if ignoredHash != "" {
		t.Error("ignored-inbound should not be in hash map")
	}

	// Check inbounds list
	tags := m.GetXtlsConfigInbounds()
	if len(tags) != 1 {
		t.Errorf("Expected 1 inbound, got %d", len(tags))
	}
}

func TestConfigManager_AddRemoveUser(t *testing.T) {
	m := NewConfigManager(nil)

	// Setup initial state
	hashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(hashes, config)

	emptyHash := m.GetInboundHash("vless-in")
	if emptyHash != "0000000000000000" {
		t.Errorf("Empty inbound should have zero hash, got %s", emptyHash)
	}

	// Add user
	m.AddUserToInbound("vless-in", "test-uuid")
	afterAddHash := m.GetInboundHash("vless-in")
	if afterAddHash == "0000000000000000" {
		t.Error("After adding user, hash should not be zero")
	}

	// Remove user
	m.RemoveUserFromInbound("vless-in", "test-uuid")
	afterRemoveHash := m.GetInboundHash("vless-in")

	// After removing only user, inbound should be removed
	if afterRemoveHash != "" {
		t.Error("After removing last user, inbound should be removed from map")
	}
}

func TestConfigManager_AddUserToNewInbound(t *testing.T) {
	m := NewConfigManager(nil)

	// Add user to non-existent inbound
	m.AddUserToInbound("new-inbound", "user-id")

	hash := m.GetInboundHash("new-inbound")
	if hash == "" {
		t.Error("New inbound should be created with user")
	}
	if hash == "0000000000000000" {
		t.Error("Inbound with user should not have zero hash")
	}
}

func TestConfigManager_Cleanup(t *testing.T) {
	m := NewConfigManager(nil)

	// Setup state
	hashes := Hashes{
		EmptyConfig: "hash123",
		Inbounds:    []InboundHash{{Tag: "vless-in", Hash: "0000000000000000", UsersCount: 0}},
	}
	config := map[string]interface{}{
		"inbounds": []interface{}{
			map[string]interface{}{
				"tag":      "vless-in",
				"settings": map[string]interface{}{"clients": []interface{}{}},
			},
		},
	}
	_ = m.ExtractUsersFromConfig(hashes, config)

	// Verify state exists
	if len(m.GetXtlsConfigInbounds()) == 0 {
		t.Error("Should have inbounds before cleanup")
	}

	// Cleanup
	m.Cleanup()

	// Verify state cleared
	if len(m.GetXtlsConfigInbounds()) != 0 {
		t.Error("Inbounds should be empty after cleanup")
	}
	if m.GetInboundHash("vless-in") != "" {
		t.Error("Hash map should be empty after cleanup")
	}

	// After cleanup, should need restart
	if !m.IsNeedRestartCore(hashes) {
		t.Error("After cleanup, should need restart")
	}
}

func TestConfigManager_GetXrayConfig(t *testing.T) {
	m := NewConfigManager(nil)

	// Empty config returns empty map
	cfg := m.GetXrayConfig()
	if cfg == nil {
		t.Error("GetXrayConfig should never return nil")
	}
	if len(cfg) != 0 {
		t.Error("Initial config should be empty")
	}

	// Set config
	m.SetXrayConfig(map[string]interface{}{"key": "value"})
	cfg = m.GetXrayConfig()
	if cfg["key"] != "value" {
		t.Error("Config should be retrievable")
	}
}
