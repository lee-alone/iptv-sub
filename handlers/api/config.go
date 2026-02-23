package api

import (
	"fmt"
	"net/http"
	"time"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// ConfigHandlers 配置相关处理器
type ConfigHandlers struct {
	tester *services.StreamTester
	cfg    *config.Config
	logger *utils.Logger
}

// NewConfigHandlers 创建配置处理器
func NewConfigHandlers(
	tester *services.StreamTester,
	cfg *config.Config,
	logger *utils.Logger,
) *ConfigHandlers {
	return &ConfigHandlers{
		tester: tester,
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterRoutes 注册配置相关路由
func (h *ConfigHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/config", h.get)
	rg.PUT("/config", h.update)
}

// get 获取配置
func (h *ConfigHandlers) get(c *gin.Context) {
	// 获取本机 IP 和完整地址
	var localIP string
	if h.cfg.Host == "0.0.0.0" || h.cfg.Host == "" {
		localIP = utils.GetBestLocalIP()
	} else {
		localIP = h.cfg.Host
	}

	serverAddress := utils.GetPrimaryAddress(localIP, h.cfg.Port)
	playlistURL := utils.GetPlaylistURL(localIP, h.cfg.Port, "/playlist.m3u")

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"server_address":              serverAddress,
			"playlist_url":                playlistURL,
			"local_ip":                    localIP,
			"port":                        h.cfg.Port,
			"update_interval":             int(h.cfg.UpdateInterval.Hours()),
			"test_interval":               int(h.cfg.TestInterval.Hours()),
			"match_by":                    h.cfg.MatchBy,
			"similarity_threshold":        int(h.cfg.SimilarityThreshold * 100),
			"test_all_sources":            h.cfg.TestAllSources,
			"enable_stream_test":          h.cfg.EnableStreamTest,
			"stream_test_timeout":         int(h.cfg.StreamTestTimeout.Seconds()),
			"max_test_workers":            h.cfg.MaxTestWorkers,
			"deep_check":                  h.cfg.DeepCheck,
			"loop_checks":                 h.cfg.LoopChecks,
			"loop_interval":               int(h.cfg.LoopInterval.Seconds()),
			"auto_test_on_startup":        h.cfg.AutoTestOnStartup,
			"auto_test_interval_hours":    h.cfg.AutoTestIntervalHours,
			"test_on_subscription_update": h.cfg.TestOnSubscriptionUpdate,
			"log_level":                   h.cfg.LogLevel,
		},
	})
}

// update 更新配置
func (h *ConfigHandlers) update(c *gin.Context) {
	var req struct {
		UpdateInterval           int     `json:"update_interval"`
		TestInterval             int     `json:"test_interval"`
		MatchBy                  string  `json:"match_by"`
		SimilarityThreshold      int     `json:"similarity_threshold"`
		TestAllSources           bool    `json:"test_all_sources"`
		EnableStreamTest         bool    `json:"enable_stream_test"`
		StreamTestTimeout        int     `json:"stream_test_timeout"`
		MaxTestWorkers           int     `json:"max_test_workers"`
		DeepCheck                bool    `json:"deep_check"`
		LoopChecks               int     `json:"loop_checks"`
		LoopInterval             float64 `json:"loop_interval"`
		AutoTestOnStartup        bool    `json:"auto_test_on_startup"`
		AutoTestIntervalHours    int     `json:"auto_test_interval_hours"`
		TestOnSubscriptionUpdate bool    `json:"test_on_subscription_update"`
		LogLevel                 string  `json:"log_level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 MatchBy 参数
	if req.MatchBy != "" {
		if req.MatchBy != "name" && req.MatchBy != "tvg_id" && req.MatchBy != "both" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match_by value, must be one of: name, tvg_id, both"})
			return
		}
	}

	// 验证 UpdateInterval
	if req.UpdateInterval > 0 && req.UpdateInterval < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "update_interval must be >= 1"})
		return
	}

	// 验证 TestInterval
	if req.TestInterval > 0 && req.TestInterval < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "test_interval must be >= 1"})
		return
	}

	// 验证 MaxTestWorkers 上限
	if req.MaxTestWorkers > 0 && req.MaxTestWorkers > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_test_workers must be <= 100"})
		return
	}

	// 验证 AutoTestIntervalHours
	if req.AutoTestIntervalHours > 0 && req.AutoTestIntervalHours < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto_test_interval_hours must be >= 1"})
		return
	}

	// 更新配置对象
	if req.UpdateInterval > 0 {
		h.cfg.UpdateInterval = time.Duration(req.UpdateInterval) * time.Hour
	}
	if req.TestInterval > 0 {
		h.cfg.TestInterval = time.Duration(req.TestInterval) * time.Hour
	}
	if req.MatchBy != "" {
		h.cfg.MatchBy = req.MatchBy
	}
	if req.SimilarityThreshold >= 0 && req.SimilarityThreshold <= 100 {
		h.cfg.SimilarityThreshold = float64(req.SimilarityThreshold) / 100.0
	}
	h.cfg.TestAllSources = req.TestAllSources
	h.cfg.EnableStreamTest = req.EnableStreamTest
	if req.StreamTestTimeout > 0 {
		h.cfg.StreamTestTimeout = time.Duration(req.StreamTestTimeout) * time.Second
		h.tester.SetStreamTestTimeout(h.cfg.StreamTestTimeout)
	}
	if req.MaxTestWorkers > 0 {
		h.cfg.MaxTestWorkers = req.MaxTestWorkers
		h.tester.SetMaxWorkers(h.cfg.MaxTestWorkers)
	}
	h.cfg.DeepCheck = req.DeepCheck
	if req.LoopChecks > 0 {
		h.cfg.LoopChecks = req.LoopChecks
	}
	if req.LoopInterval > 0 {
		h.cfg.LoopInterval = time.Duration(req.LoopInterval * float64(time.Second))
	}
	h.cfg.AutoTestOnStartup = req.AutoTestOnStartup
	if req.AutoTestIntervalHours > 0 {
		h.cfg.AutoTestIntervalHours = req.AutoTestIntervalHours
	}
	h.cfg.TestOnSubscriptionUpdate = req.TestOnSubscriptionUpdate
	if req.LogLevel != "" {
		h.cfg.LogLevel = req.LogLevel
	}

	// 同步更新 StreamTester 的深度检查选项
	h.tester.SetDeepCheckOptions(h.cfg.DeepCheck, h.cfg.LoopChecks, h.cfg.LoopInterval, h.cfg.SegmentWindow)

	// 保存配置到文件
	if err := config.SaveConfig("config.json", h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save config: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated successfully"})
}
