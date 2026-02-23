package api

import (
	"fmt"
	"net/http"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// ExportHandlers 导出相关处理器
type ExportHandlers struct {
	agg      *aggregator.ChannelAggregator
	exporter *services.ChannelExporter
	cfg      *config.Config
	logger   *utils.Logger
}

// NewExportHandlers 创建导出处理器
func NewExportHandlers(
	agg *aggregator.ChannelAggregator,
	exporter *services.ChannelExporter,
	cfg *config.Config,
	logger *utils.Logger,
) *ExportHandlers {
	return &ExportHandlers{
		agg:      agg,
		exporter: exporter,
		cfg:      cfg,
		logger:   logger,
	}
}

// RegisterRoutes 注册导出相关路由
func (h *ExportHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/export", h.export)
}

// export 导出频道
func (h *ExportHandlers) export(c *gin.Context) {
	var req struct {
		Format      string `json:"format" binding:"required"` // m3u, json
		OnlyWorking bool   `json:"only_working"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channels := h.agg.GetAllChannels()
	var filepath string
	var err error

	switch req.Format {
	case "m3u":
		filepath, err = h.exporter.ExportM3U(channels, req.OnlyWorking)
	case "json":
		filepath, err = h.exporter.ExportJSON(channels, req.OnlyWorking)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("export failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"filepath": filepath})
}
