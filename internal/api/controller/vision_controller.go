package controller

import (
	"crypto/md5"
	"encoding/hex"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/remnawave/node-go/internal/logger"
	"github.com/remnawave/node-go/internal/xray"
)

// BlockIPRequest represents the request body for block/unblock IP endpoints.
type BlockIPRequest struct {
	IP string `json:"ip" binding:"required"`
}

// BlockIPResponse represents the response for block/unblock IP endpoints.
type BlockIPResponse struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

// VisionController handles IP blocking/unblocking operations.
// Note: Currently uses in-memory tracking. Full xray-core integration
// would require the grpc command service to add/remove routing rules.
type VisionController struct {
	core       *xray.Core
	logger     *logger.Logger
	blockedIPs map[string]string // ruleTag (MD5 hash) -> IP
	mu         sync.RWMutex
}

// NewVisionController creates a new VisionController instance.
func NewVisionController(core *xray.Core, log *logger.Logger) *VisionController {
	return &VisionController{
		core:       core,
		logger:     log,
		blockedIPs: make(map[string]string),
	}
}

// RegisterRoutes registers the vision controller routes.
func (c *VisionController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/block-ip", c.handleBlockIP)
	group.POST("/unblock-ip", c.handleUnblockIP)
}

// getIPHash generates an MD5 hash of the IP address for use as a rule tag.
func (c *VisionController) getIPHash(ip string) string {
	hash := md5.Sum([]byte(ip))
	return hex.EncodeToString(hash[:])
}

// handleBlockIP handles the POST /vision/block-ip endpoint.
func (c *VisionController) handleBlockIP(ctx *gin.Context) {
	var req BlockIPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.WithError(err).Error("Failed to parse block-ip request")
		errMsg := "invalid request body: " + err.Error()
		ctx.JSON(http.StatusBadRequest, wrapResponse(BlockIPResponse{
			Success: false,
			Error:   &errMsg,
		}))
		return
	}

	if net.ParseIP(req.IP) == nil {
		errMsg := "invalid IP address format"
		ctx.JSON(http.StatusBadRequest, wrapResponse(BlockIPResponse{
			Success: false,
			Error:   &errMsg,
		}))
		return
	}

	ruleTag := c.getIPHash(req.IP)

	c.mu.Lock()
	c.blockedIPs[ruleTag] = req.IP
	c.mu.Unlock()

	// Note: Full xray-core integration would add a routing rule here:
	// - Rule tag: ruleTag (MD5 hex of IP)
	// - Source IP: req.IP
	// - Outbound: "BLOCK"
	// - Would use xray-core router feature API or grpc command service

	c.logger.WithField("ip", req.IP).WithField("ruleTag", ruleTag).Info("IP blocked")

	ctx.JSON(http.StatusOK, wrapResponse(BlockIPResponse{
		Success: true,
		Error:   nil,
	}))
}

// handleUnblockIP handles the POST /vision/unblock-ip endpoint.
func (c *VisionController) handleUnblockIP(ctx *gin.Context) {
	var req BlockIPRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.WithError(err).Error("Failed to parse unblock-ip request")
		errMsg := "invalid request body: " + err.Error()
		ctx.JSON(http.StatusBadRequest, wrapResponse(BlockIPResponse{
			Success: false,
			Error:   &errMsg,
		}))
		return
	}

	if net.ParseIP(req.IP) == nil {
		errMsg := "invalid IP address format"
		ctx.JSON(http.StatusBadRequest, wrapResponse(BlockIPResponse{
			Success: false,
			Error:   &errMsg,
		}))
		return
	}

	ruleTag := c.getIPHash(req.IP)

	c.mu.Lock()
	delete(c.blockedIPs, ruleTag)
	c.mu.Unlock()

	// Note: Full xray-core integration would remove the routing rule here:
	// - Remove rule by tag: ruleTag

	c.logger.WithField("ip", req.IP).WithField("ruleTag", ruleTag).Info("IP unblocked")

	ctx.JSON(http.StatusOK, wrapResponse(BlockIPResponse{
		Success: true,
		Error:   nil,
	}))
}

// GetBlockedIPs returns a list of all currently blocked IPs.
func (c *VisionController) GetBlockedIPs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ips := make([]string, 0, len(c.blockedIPs))
	for _, ip := range c.blockedIPs {
		ips = append(ips, ip)
	}
	return ips
}

// IsBlocked checks if an IP is currently blocked.
func (c *VisionController) IsBlocked(ip string) bool {
	ruleTag := c.getIPHash(ip)
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, blocked := c.blockedIPs[ruleTag]
	return blocked
}
