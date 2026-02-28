package api

import (
	"encoding/json"
	"net/http"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// SubscriptionHandlers 订阅源相关处理器
type SubscriptionHandlers struct {
	subscriptionMgr *services.SubscriptionManager
	parser          *services.M3UParser
	agg             *aggregator.ChannelAggregator
	tester          *services.StreamTester
	cfg             *config.Config
	logger          *utils.Logger
}

// NewSubscriptionHandlers 创建订阅源处理器
func NewSubscriptionHandlers(
	subscriptionMgr *services.SubscriptionManager,
	parser *services.M3UParser,
	agg *aggregator.ChannelAggregator,
	tester *services.StreamTester,
	cfg *config.Config,
	logger *utils.Logger,
) *SubscriptionHandlers {
	return &SubscriptionHandlers{
		subscriptionMgr: subscriptionMgr,
		parser:          parser,
		agg:             agg,
		tester:          tester,
		cfg:             cfg,
		logger:          logger,
	}
}

// RegisterRoutes 注册订阅源相关路由
func (h *SubscriptionHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	subs := rg.Group("/subscriptions")
	{
		subs.GET("", h.list)
		subs.POST("", h.add)
		subs.DELETE("", h.remove)
		subs.PUT("", h.update)
		subs.GET("/export", h.export)
		subs.POST("/import", h.importSubs)
	}
	// 更新所有订阅源
	rg.POST("/subscriptions/update", h.updateAll)
}

// list 获取所有订阅源
func (h *SubscriptionHandlers) list(c *gin.Context) {
	subs := h.subscriptionMgr.GetAllSubscriptions()
	c.JSON(http.StatusOK, gin.H{
		"data":  subs,
		"count": len(subs),
	})
}

// add 添加订阅源
func (h *SubscriptionHandlers) add(c *gin.Context) {
	var req struct {
		URL     string `json:"url" binding:"required"`
		Name    string `json:"name" binding:"required"`
		Enabled bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.subscriptionMgr.AddSubscription(req.URL, req.Name, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "subscription added"})
}

// remove 删除订阅源
func (h *SubscriptionHandlers) remove(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}
	if err := h.subscriptionMgr.RemoveSubscription(url); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription removed"})
}

// update 更新订阅源
func (h *SubscriptionHandlers) update(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Enabled bool   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	oldURL := c.Query("url")
	if oldURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}
	if err := h.subscriptionMgr.UpdateSubscription(oldURL, oldURL, req.Name, req.Enabled); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription updated"})
}

// updateAll 更新所有订阅源
func (h *SubscriptionHandlers) updateAll(c *gin.Context) {
	logger := utils.NewLogger()
	subs := h.subscriptionMgr.GetAllSubscriptions()
	logger.Info("Starting update for %d subscriptions", len(subs))
	var results []gin.H

	for _, sub := range subs {
		logger.Info("Updating subscription: %s (%s)", sub.Name, sub.URL)
		if !sub.Enabled {
			logger.Info("Subscription %s is disabled, skipping", sub.Name)
			continue
		}

		// 获取 M3U 文件
		content, err := h.parser.FetchM3U(sub.URL)
		if err != nil {
			h.subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "failed", 0)
			continue
		}

		// 解析 M3U
		channels, err := h.parser.ParseM3U(content, sub.URL)
		if err != nil {
			h.subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "failed", 0)
			continue
		}

		// 聚合频道
		added, updated, skipped, _ := h.agg.AggregateChannels(channels, h.cfg.MatchBy, h.cfg.SimilarityThreshold)

		// 更新频道计数
		count := h.agg.GetChannelCountBySource(sub.URL)
		h.subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "active", count)

		results = append(results, gin.H{
			"name":    sub.Name,
			"added":   added,
			"updated": updated,
			"skipped": skipped,
		})
		logger.Info("Subscription %s updated: +%d, ~%d", sub.Name, added, updated)
	}

	// 更新完成后，如果配置了自动测试，则执行测试
	if h.cfg.TestOnSubscriptionUpdate && h.cfg.EnableStreamTest {
		logger.Info("Auto-testing channels after subscription update...")
		allChannels := h.agg.GetAllChannels()
		if len(allChannels) > 0 {
			h.agg.ResetTestResults()
			tested, err := h.tester.BatchTest(allChannels, h.cfg.TestAllSources)
			if err != nil {
				logger.Error("Auto-test after subscription update failed: %v", err)
			} else {
				logger.Info("Auto-test after subscription update completed: %d channels tested", len(tested))
				h.agg.Save()
			}
		}
	}

	logger.Info("Update completed, processed %d subscriptions", len(results))
	c.JSON(http.StatusOK, gin.H{
		"message": "update completed",
		"results": results,
	})
}

// export 导出所有订阅源为JSON
func (h *SubscriptionHandlers) export(c *gin.Context) {
	subs := h.subscriptionMgr.GetAllSubscriptions()

	// 设置响应头为JSON格式下载
	c.Header("Content-Disposition", "attachment; filename=subscriptions.json")
	c.Header("Content-Type", "application/json")

	data := gin.H{
		"subscriptions": subs,
		"version":       "1.0",
	}

	c.JSON(http.StatusOK, data)
}

// importSubs 导入订阅源
func (h *SubscriptionHandlers) importSubs(c *gin.Context) {
	var req struct {
		Subscriptions interface{} `json:"subscriptions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换为订阅源列表
	var subs []map[string]interface{}
	subsBytes, err := json.Marshal(req.Subscriptions)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriptions format"})
		return
	}

	if err := json.Unmarshal(subsBytes, &subs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriptions format"})
		return
	}

	var added, failed int
	for _, sub := range subs {
		url, ok := sub["url"].(string)
		if !ok || url == "" {
			failed++
			continue
		}

		name, _ := sub["name"].(string)
		if name == "" {
			name = url
		}

		enabled := true
		if e, ok := sub["enabled"].(bool); ok {
			enabled = e
		}

		if err := h.subscriptionMgr.AddSubscription(url, name, enabled); err != nil {
			h.logger.Warn("Failed to import subscription %s: %v", url, err)
			failed++
			continue
		}
		added++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "import completed",
		"added":   added,
		"failed":  failed,
	})
}
