package handlers

import (
	"embed"
	"net/http"
	"time"

	"iptv-aggregator/config"
	"iptv-aggregator/handlers/api"
	"iptv-aggregator/handlers/web"
	"iptv-aggregator/services"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// SetupRouter 设置 HTTP 路由
func SetupRouter(
	subscriptionMgr *services.SubscriptionManager,
	parser *services.M3UParser,
	agg *aggregator.ChannelAggregator,
	tester *services.StreamTester,
	scheduler *services.Scheduler,
	exporter *services.ChannelExporter,
	cfg *config.Config,
	logger *utils.Logger,
) *gin.Engine {
	router := gin.New()

	// 注册 Recovery 中间件
	router.Use(gin.Recovery())

	// 注册自定义日志中间件
	router.Use(LoggerMiddleware(logger))

	// 注册 CORS 中间件
	router.Use(CORSMiddleware())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		stats := agg.GetAllChannels()
		onlineCount := 0
		for _, ch := range stats {
			if ch.IsOnline() {
				onlineCount++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":          "ok",
			"version":         "1.0.0",
			"total_channels":  len(stats),
			"online_channels": onlineCount,
			"subscriptions":   len(subscriptionMgr.GetAllSubscriptions()),
			"timestamp":       time.Now().Unix(),
		})
	})

	// 注册 API 路由
	registerAPIRoutes(router, subscriptionMgr, parser, agg, tester, exporter, cfg, logger)

	// 注册 Web 路由
	registerWebRoutes(router, agg, cfg, logger)

	return router
}

// registerAPIRoutes 注册 API 路由
func registerAPIRoutes(
	router *gin.Engine,
	subscriptionMgr *services.SubscriptionManager,
	parser *services.M3UParser,
	agg *aggregator.ChannelAggregator,
	tester *services.StreamTester,
	exporter *services.ChannelExporter,
	cfg *config.Config,
	logger *utils.Logger,
) {
	apiGroup := router.Group("/api")
	{
		// 订阅源 API
		subHandlers := api.NewSubscriptionHandlers(subscriptionMgr, parser, agg, tester, cfg, logger)
		subHandlers.RegisterRoutes(apiGroup)

		// 频道 API
		channelHandlers := api.NewChannelHandlers(agg, tester, cfg, logger)
		channelHandlers.RegisterRoutes(apiGroup)

		// 聚合 API
		aggregateHandlers := api.NewAggregateHandlers(subscriptionMgr, parser, agg, cfg, logger)
		aggregateHandlers.RegisterRoutes(apiGroup)

		// 测试 API
		testHandlers := api.NewTestHandlers(agg, tester, subscriptionMgr, cfg, logger)
		testHandlers.RegisterRoutes(apiGroup)

		// 导出 API
		exportHandlers := api.NewExportHandlers(agg, exporter, cfg, logger)
		exportHandlers.RegisterRoutes(apiGroup)

		// 配置 API
		configHandlers := api.NewConfigHandlers(tester, cfg, logger)
		configHandlers.RegisterRoutes(apiGroup)

		// 统计 API
		statsHandlers := api.NewStatsHandlers(agg, subscriptionMgr, cfg, logger)
		statsHandlers.RegisterRoutes(apiGroup)

		// 重启 API
		restartHandlers := api.NewRestartHandlers(cfg, logger)
		restartHandlers.RegisterRoutes(apiGroup)
	}
}

// registerWebRoutes 注册 Web 页面路由
func registerWebRoutes(
	router *gin.Engine,
	agg *aggregator.ChannelAggregator,
	cfg *config.Config,
	logger *utils.Logger,
) {
	webHandlers := web.NewWebHandlers(agg, cfg, logger)
	webHandlers.RegisterRoutes(router, templatesFS, staticFS)
}
