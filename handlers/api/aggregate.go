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

// AggregateHandlers 聚合相关处理器
type AggregateHandlers struct {
	subscriptionMgr *services.SubscriptionManager
	parser          *services.M3UParser
	agg             *aggregator.ChannelAggregator
	cfg             *config.Config
	logger          *utils.Logger
}

// NewAggregateHandlers 创建聚合处理器
func NewAggregateHandlers(
	subscriptionMgr *services.SubscriptionManager,
	parser *services.M3UParser,
	agg *aggregator.ChannelAggregator,
	cfg *config.Config,
	logger *utils.Logger,
) *AggregateHandlers {
	return &AggregateHandlers{
		subscriptionMgr: subscriptionMgr,
		parser:          parser,
		agg:             agg,
		cfg:             cfg,
		logger:          logger,
	}
}

// RegisterRoutes 注册聚合相关路由
func (h *AggregateHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/aggregate", h.aggregate)
}

// aggregate 聚合频道
func (h *AggregateHandlers) aggregate(c *gin.Context) {
	var req struct {
		SubscriptionURL string  `json:"subscription_url" binding:"required"`
		MatchBy         string  `json:"match_by"`
		Threshold       float64 `json:"threshold"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取订阅源
	sub := h.subscriptionMgr.GetSubscription(req.SubscriptionURL)
	if sub == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
		return
	}

	// 获取 M3U 文件
	content, err := h.parser.FetchM3U(sub.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to fetch M3U: %v", err)})
		h.subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "failed", 0)
		return
	}

	// 解析 M3U
	channels, err := h.parser.ParseM3U(content, sub.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to parse M3U: %v", err)})
		h.subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "failed", 0)
		return
	}

	// 聚合频道
	matchBy := req.MatchBy
	if matchBy == "" {
		matchBy = h.cfg.MatchBy
	}
	threshold := req.Threshold
	if threshold == 0 {
		threshold = h.cfg.SimilarityThreshold
	}

	added, updated, skipped, err := h.agg.AggregateChannels(channels, matchBy, threshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to aggregate: %v", err)})
		return
	}

	// 更新订阅源状态和频道计数
	subCount := h.agg.GetChannelCountBySource(req.SubscriptionURL)
	h.subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "active", subCount)

	c.JSON(http.StatusOK, gin.H{
		"added":   added,
		"updated": updated,
		"skipped": skipped,
	})
}
