package xray

import (
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/shadowsocks"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
)

// CipherType represents shadowsocks cipher types.
// Values match xray-core's shadowsocks.CipherType.
type CipherType int32

const (
	CipherTypeUnknown           CipherType = 0
	CipherTypeAES128GCM         CipherType = 5
	CipherTypeAES256GCM         CipherType = 6
	CipherTypeCHACHA20POLY1305  CipherType = 7
	CipherTypeXCHACHA20POLY1305 CipherType = 8
	CipherTypeNone              CipherType = 9
)

// BuildVlessUser creates a protocol.User for VLESS protocol.
// Parameters:
//   - email: User identifier (used as email field in xray-core)
//   - uuid: VLESS client ID (UUID format)
//   - flow: VLESS flow setting (e.g., "xtls-rprx-vision" or "")
//   - level: User permission level (typically 0)
func BuildVlessUser(email, uuid, flow string, level uint32) *protocol.User {
	vlessAccount := &vless.Account{
		Id:   uuid,
		Flow: flow,
	}

	return &protocol.User{
		Level:   level,
		Email:   email,
		Account: serial.ToTypedMessage(vlessAccount),
	}
}

// BuildTrojanUser creates a protocol.User for Trojan protocol.
// Parameters:
//   - email: User identifier (used as email field in xray-core)
//   - password: Trojan password
//   - level: User permission level (typically 0)
func BuildTrojanUser(email, password string, level uint32) *protocol.User {
	trojanAccount := &trojan.Account{
		Password: password,
	}

	return &protocol.User{
		Level:   level,
		Email:   email,
		Account: serial.ToTypedMessage(trojanAccount),
	}
}

// BuildShadowsocksUser creates a protocol.User for Shadowsocks protocol.
// Parameters:
//   - email: User identifier (used as email field in xray-core)
//   - password: Shadowsocks password
//   - cipherType: Encryption cipher type
//   - ivCheck: Whether to enable IV check (for replay attack protection)
//   - level: User permission level (typically 0)
func BuildShadowsocksUser(email, password string, cipherType CipherType, ivCheck bool, level uint32) *protocol.User {
	ssAccount := &shadowsocks.Account{
		Password:   password,
		CipherType: shadowsocks.CipherType(cipherType),
		IvCheck:    ivCheck,
	}

	return &protocol.User{
		Level:   level,
		Email:   email,
		Account: serial.ToTypedMessage(ssAccount),
	}
}

// UserData represents user-specific data for all protocols.
// This matches the original project's userData structure.
type UserData struct {
	UserID         string // Username/email for identification
	HashUUID       string // UUID used for hash tracking
	VlessUUID      string // UUID for VLESS protocol
	TrojanPassword string // Password for Trojan
	SSPassword     string // Password for Shadowsocks
}

// InboundUserData represents protocol-specific data for a single inbound.
type InboundUserData struct {
	Type string // "vless", "trojan", "shadowsocks"
	Tag  string // Inbound tag

	// VLESS-specific
	Flow string // e.g., "xtls-rprx-vision" or ""

	// Shadowsocks-specific
	CipherType CipherType
	IVCheck    bool
}

// BuildUserForInbound creates a protocol.User based on inbound type and user data.
func BuildUserForInbound(inbound InboundUserData, user UserData) *protocol.User {
	const level uint32 = 0

	switch inbound.Type {
	case "vless":
		return BuildVlessUser(user.UserID, user.VlessUUID, inbound.Flow, level)
	case "trojan":
		return BuildTrojanUser(user.UserID, user.TrojanPassword, level)
	case "shadowsocks":
		return BuildShadowsocksUser(user.UserID, user.SSPassword, inbound.CipherType, inbound.IVCheck, level)
	default:
		return nil
	}
}

// ParseCipherType converts a cipher type string to CipherType.
func ParseCipherType(s string) CipherType {
	switch s {
	case "aes-128-gcm", "AES_128_GCM":
		return CipherTypeAES128GCM
	case "aes-256-gcm", "AES_256_GCM":
		return CipherTypeAES256GCM
	case "chacha20-poly1305", "chacha20-ietf-poly1305", "CHACHA20_POLY1305":
		return CipherTypeCHACHA20POLY1305
	case "xchacha20-poly1305", "xchacha20-ietf-poly1305", "XCHACHA20_POLY1305":
		return CipherTypeXCHACHA20POLY1305
	case "none", "NONE":
		return CipherTypeNone
	default:
		return CipherTypeUnknown
	}
}
