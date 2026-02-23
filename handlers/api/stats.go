package api

import (
	"net/http"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// StatsHandlers 统计相关处理器
type StatsHandlers struct {
	agg             *aggregator.ChannelAggregator
	subscriptionMgr *services.SubscriptionManager
	cfg             *config.Config
	logger          *utils.Logger
}

// NewStatsHandlers 创建统计处理器
func NewStatsHandlers(
	agg *aggregator.ChannelAggregator,
	subscriptionMgr *services.SubscriptionManager,
	cfg *config.Config,
	logger *utils.Logger,
) *StatsHandlers {
	return &StatsHandlers{
		agg:             agg,
		subscriptionMgr: subscriptionMgr,
		cfg:             cfg,
		logger:          logger,
	}
}

// RegisterRoutes 注册统计相关路由
func (h *StatsHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/stats", h.get)
}

// get 获取统计数据
func (h *StatsHandlers) get(c *gin.Context) {
	allChannels := h.agg.GetAllChannels()
	onlineChannels := h.agg.GetOnlineChannels()
	offlineChannels := h.agg.GetOfflineChannels()
	untestedChannels := h.agg.GetUntestedChannels()
	subs := h.subscriptionMgr.GetAllSubscriptions()

	// 可以在这里也触发一次刷新，确保 channel_count 是最新的
	for _, sub := range subs {
		count := h.agg.GetChannelCountBySource(sub.URL)
		if sub.ChannelCount != count {
			h.subscriptionMgr.UpdateSubscriptionStatus(sub.URL, sub.Status, count)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_channels":    len(allChannels),
		"online_channels":   len(onlineChannels),
		"offline_channels":  len(offlineChannels),
		"untested_channels": len(untestedChannels),
		"subscriptions":     len(subs),
	})
}
