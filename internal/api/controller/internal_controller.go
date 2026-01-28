package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/remnawave/node-go/internal/logger"
	"github.com/remnawave/node-go/internal/xray"
)

// InternalController handles internal API endpoints.
type InternalController struct {
	configManager *xray.ConfigManager
	logger        *logger.Logger
}

// NewInternalController creates a new InternalController instance.
func NewInternalController(configManager *xray.ConfigManager, log *logger.Logger) *InternalController {
	return &InternalController{
		configManager: configManager,
		logger:        log,
	}
}

// RegisterRoutes registers the internal controller routes.
func (c *InternalController) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/get-config", c.handleGetConfig)
}

// handleGetConfig returns the raw xray configuration JSON (not wrapped).
func (c *InternalController) handleGetConfig(ctx *gin.Context) {
	config := c.configManager.GetXrayConfig()
	ctx.JSON(http.StatusOK, config)
}
