package api

import (
	"net/http"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// ChannelHandlers 频道相关处理器
type ChannelHandlers struct {
	agg    *aggregator.ChannelAggregator
	tester *services.StreamTester
	cfg    *config.Config
	logger *utils.Logger
}

// NewChannelHandlers 创建频道处理器
func NewChannelHandlers(
	agg *aggregator.ChannelAggregator,
	tester *services.StreamTester,
	cfg *config.Config,
	logger *utils.Logger,
) *ChannelHandlers {
	return &ChannelHandlers{
		agg:    agg,
		tester: tester,
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterRoutes 注册频道相关路由
func (h *ChannelHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	channels := rg.Group("/channels")
	{
		channels.GET("", h.list)
		channels.GET("/:id", h.get)
		channels.DELETE("/:id", h.remove)
		channels.POST("/:id/test", h.test)
	}
}

// list 获取频道列表
func (h *ChannelHandlers) list(c *gin.Context) {
	onlyOnline := c.Query("online") == "true"
	var chList interface{}

	if onlyOnline {
		chList = h.agg.GetOnlineChannels()
	} else {
		chList = h.agg.GetAllChannels()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": chList,
	})
}

// get 获取单个频道
func (h *ChannelHandlers) get(c *gin.Context) {
	id := c.Param("id")
	ch := h.agg.GetChannelByID(id)
	if ch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": ch})
}

// remove 删除频道
func (h *ChannelHandlers) remove(c *gin.Context) {
	id := c.Param("id")
	if err := h.agg.RemoveChannel(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "channel deleted"})
}

// test 测试单个频道
func (h *ChannelHandlers) test(c *gin.Context) {
	id := c.Param("id")
	ch := h.agg.GetChannelByID(id)
	if ch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	h.tester.TestSingleChannel(ch, true)
	h.agg.Save() // 保存测试结果到磁盘

	c.JSON(http.StatusOK, gin.H{
		"message": "test completed",
		"data":    ch,
	})
}
