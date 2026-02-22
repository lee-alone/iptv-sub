package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"io/fs"

	"iptv-aggregator/config"
	"iptv-aggregator/services"
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
	aggregator *services.ChannelAggregator,
	tester *services.StreamTester,
	scheduler *services.Scheduler,
	exporter *services.ChannelExporter,
	cfg *config.Config,
) *gin.Engine {
	router := gin.Default()

	// 添加 CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		stats := aggregator.GetAllChannels()
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

	// API 路由
	api := router.Group("/api")
	{
		// 订阅源 API
		subscriptions := api.Group("/subscriptions")
		{
			subscriptions.GET("", func(c *gin.Context) {
				subs := subscriptionMgr.GetAllSubscriptions()
				c.JSON(http.StatusOK, gin.H{
					"data":  subs,
					"count": len(subs),
				})
			})

			subscriptions.POST("", func(c *gin.Context) {
				var req struct {
					URL     string `json:"url" binding:"required"`
					Name    string `json:"name" binding:"required"`
					Enabled bool   `json:"enabled"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				if err := subscriptionMgr.AddSubscription(req.URL, req.Name, req.Enabled); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusCreated, gin.H{"message": "subscription added"})
			})

			subscriptions.DELETE("/:url", func(c *gin.Context) {
				url := c.Param("url")
				if err := subscriptionMgr.RemoveSubscription(url); err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "subscription removed"})
			})

			subscriptions.PUT("/:url", func(c *gin.Context) {
				var req struct {
					Name    string `json:"name" binding:"required"`
					Enabled bool   `json:"enabled"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				oldURL := c.Param("url")
				if err := subscriptionMgr.UpdateSubscription(oldURL, oldURL, req.Name, req.Enabled); err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "subscription updated"})
			})
		}

		// 频道 API
		channels := api.Group("/channels")
		{
			channels.GET("", func(c *gin.Context) {
				onlyOnline := c.Query("online") == "true"
				var chList interface{}

				if onlyOnline {
					chList = aggregator.GetOnlineChannels()
				} else {
					chList = aggregator.GetAllChannels()
				}

				c.JSON(http.StatusOK, gin.H{
					"data": chList,
				})
			})

			channels.GET("/:id", func(c *gin.Context) {
				id := c.Param("id")
				ch := aggregator.GetChannelByID(id)
				if ch == nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"data": ch})
			})

			channels.DELETE("/:id", func(c *gin.Context) {
				id := c.Param("id")
				if err := aggregator.RemoveChannel(id); err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "channel deleted"})
			})

			channels.POST("/:id/test", func(c *gin.Context) {
				id := c.Param("id")
				ch := aggregator.GetChannelByID(id)
				if ch == nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
					return
				}

				tester.TestSingleChannel(ch, true)
				aggregator.Save() // 保存测试结果到磁盘

				c.JSON(http.StatusOK, gin.H{
					"message": "test completed",
					"data":    ch,
				})
			})
		}

		// 聚合 API
		api.POST("/aggregate", func(c *gin.Context) {
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
			sub := subscriptionMgr.GetSubscription(req.SubscriptionURL)
			if sub == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "subscription not found"})
				return
			}

			// 获取 M3U 文件
			content, err := parser.FetchM3U(sub.URL)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to fetch M3U: %v", err)})
				subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "failed", 0)
				return
			}

			// 解析 M3U
			channels, err := parser.ParseM3U(content, sub.URL)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to parse M3U: %v", err)})
				subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "failed", 0)
				return
			}

			// 聚合频道
			matchBy := req.MatchBy
			if matchBy == "" {
				matchBy = cfg.MatchBy
			}
			threshold := req.Threshold
			if threshold == 0 {
				threshold = cfg.SimilarityThreshold
			}

			added, updated, skipped, err := aggregator.AggregateChannels(channels, matchBy, threshold)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to aggregate: %v", err)})
				return
			}

			// 更新订阅源状态和频道计数
			subCount := aggregator.GetChannelCountBySource(req.SubscriptionURL)
			subscriptionMgr.UpdateSubscriptionStatus(req.SubscriptionURL, "active", subCount)

			c.JSON(http.StatusOK, gin.H{
				"added":   added,
				"updated": updated,
				"skipped": skipped,
			})
		})

		// 测试 API
		api.POST("/test", func(c *gin.Context) {
			var req struct {
				TestAllSources bool `json:"test_all_sources"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// 1. 立即重置所有频道为未测试状态
			logger := utils.NewLogger()
			logger.Info("Global test triggered, resetting all channel results")
			aggregator.ResetTestResults()

			logger.Info("Reset complete, starting batch test workflow")
			channels := aggregator.GetAllChannels()
			_, err := tester.BatchTest(channels, req.TestAllSources)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("test failed: %v", err)})
				return
			}

			// 更新所有订阅源的频道计数
			subs := subscriptionMgr.GetAllSubscriptions()
			for _, sub := range subs {
				count := aggregator.GetChannelCountBySource(sub.URL)
				subscriptionMgr.UpdateSubscriptionStatus(sub.URL, sub.Status, count)
			}

			c.JSON(http.StatusOK, gin.H{"message": "test completed"})
		})

		// 更新所有订阅源 API
		api.POST("/subscriptions/update", func(c *gin.Context) {
			logger := utils.NewLogger()
			subs := subscriptionMgr.GetAllSubscriptions()
			logger.Info("Starting update for %d subscriptions", len(subs))
			var results []gin.H

			for _, sub := range subs {
				logger.Info("Updating subscription: %s (%s)", sub.Name, sub.URL)
				if !sub.Enabled {
					logger.Info("Subscription %s is disabled, skipping", sub.Name)
					continue
				}

				// 获取 M3U 文件
				content, err := parser.FetchM3U(sub.URL)
				if err != nil {
					subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "failed", 0)
					continue
				}

				// 解析 M3U
				channels, err := parser.ParseM3U(content, sub.URL)
				if err != nil {
					subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "failed", 0)
					continue
				}

				// 聚合频道
				added, updated, skipped, _ := aggregator.AggregateChannels(channels, cfg.MatchBy, cfg.SimilarityThreshold)

				// 更新频道计数
				count := aggregator.GetChannelCountBySource(sub.URL)
				subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "active", count)

				results = append(results, gin.H{
					"name":    sub.Name,
					"added":   added,
					"updated": updated,
					"skipped": skipped,
				})
				logger.Info("Subscription %s updated: +%d, ~%d", sub.Name, added, updated)
			}

			logger.Info("Update completed, processed %d subscriptions", len(results))
			c.JSON(http.StatusOK, gin.H{
				"message": "update completed",
				"results": results,
			})
		})

		// 导出 API
		api.POST("/export", func(c *gin.Context) {
			var req struct {
				Format      string `json:"format" binding:"required"` // m3u, json
				OnlyWorking bool   `json:"only_working"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			channels := aggregator.GetAllChannels()
			var filepath string
			var err error

			switch req.Format {
			case "m3u":
				filepath, err = exporter.ExportM3U(channels, req.OnlyWorking)
			case "json":
				filepath, err = exporter.ExportJSON(channels, req.OnlyWorking)
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
				return
			}

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("export failed: %v", err)})
				return
			}

			c.JSON(http.StatusOK, gin.H{"filepath": filepath})
		})

		// 配置 API
		api.GET("/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"data": gin.H{
					"update_interval":      int(cfg.UpdateInterval.Hours()),
					"match_by":             cfg.MatchBy,
					"similarity_threshold": int(cfg.SimilarityThreshold * 100),
					"test_all_sources":     cfg.TestAllSources,
					"stream_test_timeout":  int(cfg.StreamTestTimeout.Seconds()),
					"max_test_workers":     cfg.MaxTestWorkers,
					"deep_check":           cfg.DeepCheck,
					"loop_checks":          cfg.LoopChecks,
					"loop_interval":        int(cfg.LoopInterval.Seconds()),
				},
			})
		})

		api.PUT("/config", func(c *gin.Context) {
			var req struct {
				UpdateInterval      int    `json:"update_interval"`
				MatchBy             string `json:"match_by"`
				SimilarityThreshold int    `json:"similarity_threshold"`
				TestAllSources      bool   `json:"test_all_sources"`
				StreamTestTimeout   int    `json:"stream_test_timeout"`
				MaxTestWorkers      int    `json:"max_test_workers"`
				DeepCheck           bool   `json:"deep_check"`
				LoopChecks          int    `json:"loop_checks"`
				LoopInterval        int    `json:"loop_interval"`
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

			// 验证 MaxTestWorkers 上限
			if req.MaxTestWorkers > 0 && req.MaxTestWorkers > 100 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "max_test_workers must be <= 100"})
				return
			}

			// 更新配置对象
			if req.UpdateInterval > 0 {
				cfg.UpdateInterval = time.Duration(req.UpdateInterval) * time.Hour
			}
			if req.MatchBy != "" {
				cfg.MatchBy = req.MatchBy
			}
			if req.SimilarityThreshold >= 0 && req.SimilarityThreshold <= 100 {
				cfg.SimilarityThreshold = float64(req.SimilarityThreshold) / 100.0
			}
			cfg.TestAllSources = req.TestAllSources
			if req.StreamTestTimeout > 0 {
				cfg.StreamTestTimeout = time.Duration(req.StreamTestTimeout) * time.Second
				tester.SetStreamTestTimeout(cfg.StreamTestTimeout)
			}
			if req.MaxTestWorkers > 0 {
				cfg.MaxTestWorkers = req.MaxTestWorkers
				tester.SetMaxWorkers(cfg.MaxTestWorkers)
			}
			cfg.DeepCheck = req.DeepCheck
			if req.LoopChecks > 0 {
				cfg.LoopChecks = req.LoopChecks
			}
			if req.LoopInterval > 0 {
				cfg.LoopInterval = time.Duration(req.LoopInterval) * time.Second
			}

			// 同步更新 StreamTester 的深度检查选项
			tester.SetDeepCheckOptions(cfg.DeepCheck, cfg.LoopChecks, cfg.LoopInterval, cfg.SegmentWindow)

			// 保存配置到文件
			if err := config.SaveConfig("config.json", cfg); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to save config: %v", err)})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "config updated successfully"})
		})

		// 统计 API
		api.GET("/stats", func(c *gin.Context) {
			allChannels := aggregator.GetAllChannels()
			onlineChannels := aggregator.GetOnlineChannels()
			offlineChannels := aggregator.GetOfflineChannels()
			untestedChannels := aggregator.GetUntestedChannels()
			subs := subscriptionMgr.GetAllSubscriptions()

			// 可以在这里也触发一次刷新，确保 channel_count 是最新的
			for _, sub := range subs {
				count := aggregator.GetChannelCountBySource(sub.URL)
				if sub.ChannelCount != count {
					subscriptionMgr.UpdateSubscriptionStatus(sub.URL, sub.Status, count)
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"total_channels":    len(allChannels),
				"online_channels":   len(onlineChannels),
				"offline_channels":  len(offlineChannels),
				"untested_channels": len(untestedChannels),
				"subscriptions":     len(subs),
			})
		})
	}

	// 加载 HTML 模板 (内嵌)
	templatesSub, _ := fs.Sub(templatesFS, "templates")
	templ := template.Must(template.New("").ParseFS(templatesSub, "*.html"))
	router.SetHTMLTemplate(templ)

	// 静态资源 (内嵌)
	staticSub, _ := fs.Sub(staticFS, "static")
	router.StaticFS("/static", http.FS(staticSub))

	// 导出目录 (物理路径)
	router.Static("/exports", filepath.Join(cfg.DataDir, "exports"))

	// 永久订阅链接
	router.GET("/playlist.m3u", func(c *gin.Context) {
		channels := aggregator.GetOnlineChannels()

		// 设置响应头
		c.Header("Content-Type", "application/x-mpegurl")
		c.Header("Content-Disposition", "attachment; filename=playlist.m3u")

		// 直接输出内容
		c.String(http.StatusOK, "#EXTM3U\n")
		for _, ch := range channels {
			extinf := "#EXTINF:-1"
			if ch.TvgID != "" {
				extinf += fmt.Sprintf(` tvg-id="%s"`, ch.TvgID)
			}
			if ch.TvgName != "" {
				extinf += fmt.Sprintf(` tvg-name="%s"`, ch.TvgName)
			}
			if ch.TvgLogo != "" {
				extinf += fmt.Sprintf(` tvg-logo="%s"`, ch.TvgLogo)
			}
			if ch.GroupTitle != "" {
				extinf += fmt.Sprintf(` group-title="%s"`, ch.GroupTitle)
			}
			extinf += fmt.Sprintf(",%s\n", ch.Name)
			c.String(http.StatusOK, extinf)

			// 选择最佳 URL
			url := ""
			if ch.TestResults != nil && ch.TestResults.WorkingURL != "" {
				url = ch.TestResults.WorkingURL
			} else if len(ch.URLs) > 0 {
				url = ch.URLs[0]
			}
			if url != "" {
				c.String(http.StatusOK, url+"\n")
			}
		}
	})

	// Web 路由
	web := router.Group("/")
	{
		web.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{})
		})
		web.GET("/subscriptions.html", func(c *gin.Context) {
			c.HTML(http.StatusOK, "subscriptions.html", gin.H{})
		})
		web.GET("/channels.html", func(c *gin.Context) {
			c.HTML(http.StatusOK, "channels.html", gin.H{})
		})
		web.GET("/settings.html", func(c *gin.Context) {
			c.HTML(http.StatusOK, "settings.html", gin.H{})
		})
	}

	return router
}
