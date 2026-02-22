package handlers

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"io/fs"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/iptv-aggregator/config"
	"github.com/yourusername/iptv-aggregator/services"
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
			fmt.Println("API: Global test triggered, resetting all channel results...")
			aggregator.ResetTestResults()

			// 2. 预先更新统计信息的基准（可选：如果需要立即持久化到订阅源模型，可以在此处添加）

			fmt.Println("API: Reset complete, starting batch test workflow...")
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
			subs := subscriptionMgr.GetAllSubscriptions()
			fmt.Printf("API: Starting update for %d subscriptions\n", len(subs))
			var results []gin.H

			for _, sub := range subs {
				fmt.Printf("API: Updating subscription: %s (%s)\n", sub.Name, sub.URL)
				if !sub.Enabled {
					fmt.Printf("API: Subscription %s is disabled, skipping\n", sub.Name)
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
				fmt.Printf("API: Subscription %s updated: +%d, ~%d\n", sub.Name, added, updated)
			}

			fmt.Printf("API: Update completed, processed %d subscriptions\n", len(results))
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

			if req.Format == "m3u" {
				filepath, err = exporter.ExportM3U(channels, req.OnlyWorking)
			} else if req.Format == "json" {
				filepath, err = exporter.ExportJSON(channels, req.OnlyWorking)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid format"})
				return
			}

			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("export failed: %v", err)})
				return
			}

			c.JSON(http.StatusOK, gin.H{"filepath": filepath})
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
