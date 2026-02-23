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

// TestHandlers 测试相关处理器
type TestHandlers struct {
	agg             *aggregator.ChannelAggregator
	tester          *services.StreamTester
	subscriptionMgr *services.SubscriptionManager
	cfg             *config.Config
	logger          *utils.Logger
}

// NewTestHandlers 创建测试处理器
func NewTestHandlers(
	agg *aggregator.ChannelAggregator,
	tester *services.StreamTester,
	subscriptionMgr *services.SubscriptionManager,
	cfg *config.Config,
	logger *utils.Logger,
) *TestHandlers {
	return &TestHandlers{
		agg:             agg,
		tester:          tester,
		subscriptionMgr: subscriptionMgr,
		cfg:             cfg,
		logger:          logger,
	}
}

// RegisterRoutes 注册测试相关路由
func (h *TestHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/test", h.test)
}

// test 批量测试频道
func (h *TestHandlers) test(c *gin.Context) {
	var req struct {
		TestAllSources bool `json:"test_all_sources"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 立即重置所有频道为未测试状态
	h.logger.Info("Global test triggered, resetting all channel results")
	h.agg.ResetTestResults()

	h.logger.Info("Reset complete, starting batch test workflow")
	channels := h.agg.GetAllChannels()
	_, err := h.tester.BatchTest(channels, req.TestAllSources)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("test failed: %v", err)})
		return
	}

	// 保存测试结果到磁盘
	h.agg.Save()

	// 更新所有订阅源的频道计数
	subs := h.subscriptionMgr.GetAllSubscriptions()
	for _, sub := range subs {
		count := h.agg.GetChannelCountBySource(sub.URL)
		h.subscriptionMgr.UpdateSubscriptionStatus(sub.URL, sub.Status, count)
	}

	c.JSON(http.StatusOK, gin.H{"message": "test completed"})
}
