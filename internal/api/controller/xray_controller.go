package controller

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/remnawave/node-go/internal/logger"
	"github.com/remnawave/node-go/internal/xray"
)

type successResponse struct {
	Response interface{} `json:"response"`
}

func wrapResponse(data interface{}) successResponse {
	return successResponse{Response: data}
}

const (
	NodeVersion = "1.0.0"
	APIPort     = 61012
)

type StartRequest struct {
	XrayConfig map[string]interface{} `json:"xrayConfig" binding:"required"`
	Internals  xray.Internals         `json:"internals" binding:"required"`
}

type NodeInfo struct {
	Version string `json:"version"`
}

type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	NumCPU       int    `json:"numCpu"`
	GoVersion    string `json:"goVersion"`
	NumGoroutine int    `json:"numGoroutine"`
}

type StartResponse struct {
	IsStarted  bool        `json:"isStarted"`
	Version    *string     `json:"version"`
	Error      *string     `json:"error"`
	SystemInfo *SystemInfo `json:"systemInfo"`
	NodeInfo   NodeInfo    `json:"nodeInfo"`
}

type StopResponse struct {
	IsStopped bool `json:"isStopped"`
}

type StatusResponse struct {
	IsRunning bool    `json:"isRunning"`
	Version   *string `json:"version"`
}

type HealthcheckResponse struct {
	IsHealthy     bool    `json:"isHealthy"`
	IsXrayRunning bool    `json:"isXrayRunning"`
	XrayVersion   *string `json:"xrayVersion"`
	NodeVersion   string  `json:"nodeVersion"`
}

type XrayController struct {
	core          *xray.Core
	configManager *xray.ConfigManager
	logger        *logger.Logger
	startMu       sync.Mutex
	isProcessing  atomic.Bool
}

func NewXrayController(core *xray.Core, configManager *xray.ConfigManager, log *logger.Logger) *XrayController {
	return &XrayController{
		core:          core,
		configManager: configManager,
		logger:        log,
	}
}

func (c *XrayController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/start", c.handleStart)
	group.GET("/stop", c.handleStop)
	group.GET("/status", c.handleStatus)
	group.GET("/healthcheck", c.handleHealthcheck)
}

func (c *XrayController) handleStart(ctx *gin.Context) {
	if !c.isProcessing.CompareAndSwap(false, true) {
		c.logger.Warn("Start request already in progress, rejecting duplicate")
		errMsg := "another start request is already in progress"
		ctx.JSON(http.StatusConflict, wrapResponse(StartResponse{
			IsStarted: false,
			Error:     &errMsg,
			NodeInfo:  NodeInfo{Version: NodeVersion},
		}))
		return
	}
	defer c.isProcessing.Store(false)

	c.startMu.Lock()
	defer c.startMu.Unlock()

	var req StartRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.logger.WithError(err).Error("Failed to parse start request")
		errMsg := "invalid request body: " + err.Error()
		ctx.JSON(http.StatusBadRequest, wrapResponse(StartResponse{
			IsStarted: false,
			Error:     &errMsg,
			NodeInfo:  NodeInfo{Version: NodeVersion},
		}))
		return
	}

	hashes := req.Internals.Hashes
	forceRestart := req.Internals.ForceRestart

	if c.core.IsRunning() && !forceRestart {
		needRestart := c.configManager.IsNeedRestartCore(hashes)
		if !needRestart {
			version := c.core.GetVersion()
			sysInfo := getSystemInfo()
			ctx.JSON(http.StatusOK, wrapResponse(StartResponse{
				IsStarted:  true,
				Version:    &version,
				SystemInfo: &sysInfo,
				NodeInfo:   NodeInfo{Version: NodeVersion},
			}))
			return
		}
		c.logger.Info("Restart required - proceeding with xray core restart")
	}

	config := generateAPIConfig(req.XrayConfig)

	if err := c.configManager.ExtractUsersFromConfig(hashes, config); err != nil {
		c.logger.WithError(err).Error("Failed to extract users from config")
		errMsg := "failed to extract users: " + err.Error()
		ctx.JSON(http.StatusInternalServerError, wrapResponse(StartResponse{
			IsStarted: false,
			Error:     &errMsg,
			NodeInfo:  NodeInfo{Version: NodeVersion},
		}))
		return
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal xray config")
		errMsg := "failed to serialize config: " + err.Error()
		ctx.JSON(http.StatusInternalServerError, wrapResponse(StartResponse{
			IsStarted: false,
			Error:     &errMsg,
			NodeInfo:  NodeInfo{Version: NodeVersion},
		}))
		return
	}

	if err := c.core.Start(configJSON); err != nil {
		c.logger.WithError(err).Error("Failed to start xray core")
		errMsg := "failed to start xray: " + err.Error()
		ctx.JSON(http.StatusInternalServerError, wrapResponse(StartResponse{
			IsStarted: false,
			Error:     &errMsg,
			NodeInfo:  NodeInfo{Version: NodeVersion},
		}))
		return
	}

	version := c.core.GetVersion()
	sysInfo := getSystemInfo()

	c.logger.WithField("version", version).Info("Xray core started successfully")

	ctx.JSON(http.StatusOK, wrapResponse(StartResponse{
		IsStarted:  true,
		Version:    &version,
		SystemInfo: &sysInfo,
		NodeInfo:   NodeInfo{Version: NodeVersion},
	}))
}

func (c *XrayController) handleStop(ctx *gin.Context) {
	c.startMu.Lock()
	defer c.startMu.Unlock()

	if err := c.core.Stop(); err != nil {
		c.logger.WithError(err).Error("Failed to stop xray core")
		ctx.JSON(http.StatusInternalServerError, wrapResponse(StopResponse{
			IsStopped: false,
		}))
		return
	}

	c.configManager.Cleanup()

	c.logger.Info("Xray core stopped and config manager cleaned up")

	ctx.JSON(http.StatusOK, wrapResponse(StopResponse{
		IsStopped: true,
	}))
}

func (c *XrayController) handleStatus(ctx *gin.Context) {
	isRunning := c.core.IsRunning()
	var version *string
	if isRunning {
		v := c.core.GetVersion()
		version = &v
	}

	ctx.JSON(http.StatusOK, wrapResponse(StatusResponse{
		IsRunning: isRunning,
		Version:   version,
	}))
}

func (c *XrayController) handleHealthcheck(ctx *gin.Context) {
	isRunning := c.core.IsRunning()
	var xrayVersion *string
	if isRunning {
		v := c.core.GetVersion()
		xrayVersion = &v
	}

	ctx.JSON(http.StatusOK, wrapResponse(HealthcheckResponse{
		IsHealthy:     true,
		IsXrayRunning: isRunning,
		XrayVersion:   xrayVersion,
		NodeVersion:   NodeVersion,
	}))
}

func getSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

func generateAPIConfig(config map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range config {
		result[k] = v
	}

	apiInbound := map[string]interface{}{
		"tag":      "api",
		"port":     APIPort,
		"listen":   "127.0.0.1",
		"protocol": "dokodemo-door",
		"settings": map[string]interface{}{
			"address": "127.0.0.1",
		},
	}

	inbounds, ok := result["inbounds"].([]interface{})
	if !ok {
		inbounds = []interface{}{}
	}

	hasAPIInbound := false
	for _, inbound := range inbounds {
		if ib, ok := inbound.(map[string]interface{}); ok {
			if tag, ok := ib["tag"].(string); ok && tag == "api" {
				hasAPIInbound = true
				break
			}
		}
	}

	if !hasAPIInbound {
		inbounds = append(inbounds, apiInbound)
		result["inbounds"] = inbounds
	}

	routing, ok := result["routing"].(map[string]interface{})
	if !ok {
		routing = map[string]interface{}{}
	}

	rules, ok := routing["rules"].([]interface{})
	if !ok {
		rules = []interface{}{}
	}

	hasAPIRule := false
	for _, rule := range rules {
		if r, ok := rule.(map[string]interface{}); ok {
			if outboundTag, ok := r["outboundTag"].(string); ok && outboundTag == "api" {
				hasAPIRule = true
				break
			}
		}
	}

	if !hasAPIRule {
		apiRule := map[string]interface{}{
			"type":        "field",
			"outboundTag": "api",
			"inboundTag":  []interface{}{"api"},
		}
		rules = append([]interface{}{apiRule}, rules...)
		routing["rules"] = rules
		result["routing"] = routing
	}

	if _, ok := result["api"]; !ok {
		result["api"] = map[string]interface{}{
			"services": []interface{}{"HandlerService", "LoggerService", "StatsService"},
			"tag":      "api",
		}
	}

	if _, ok := result["stats"]; !ok {
		result["stats"] = map[string]interface{}{}
	}

	return result
}
