package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"

	"iptv-aggregator/config"
	"iptv-aggregator/services/aggregator"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// WebHandlers Web页面相关处理器
type WebHandlers struct {
	agg    *aggregator.ChannelAggregator
	cfg    *config.Config
	logger *utils.Logger
}

// NewWebHandlers 创建Web处理器
func NewWebHandlers(
	agg *aggregator.ChannelAggregator,
	cfg *config.Config,
	logger *utils.Logger,
) *WebHandlers {
	return &WebHandlers{
		agg:    agg,
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterRoutes 注册Web页面路由
func (h *WebHandlers) RegisterRoutes(router *gin.Engine, templatesFS, staticFS fs.FS) {
	// 加载 HTML 模板 (内嵌)
	templatesSub, _ := fs.Sub(templatesFS, "templates")
	templ := template.Must(template.New("").ParseFS(templatesSub, "*.html"))
	router.SetHTMLTemplate(templ)

	// 静态资源 (内嵌)
	staticSub, _ := fs.Sub(staticFS, "static")
	router.StaticFS("/static", http.FS(staticSub))

	// 导出目录 (物理路径)
	router.Static("/exports", filepath.Join(h.cfg.DataDir, "exports"))

	// 永久订阅链接
	router.GET("/playlist.m3u", h.playlist)

	// Web 页面路由
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
}

// playlist 生成M3U播放列表
func (h *WebHandlers) playlist(c *gin.Context) {
	channels := h.agg.GetOnlineChannels()

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
}
