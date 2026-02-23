package handlers

import (
	"time"

	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware 创建自定义日志中间件
// 根据系统 LogLevel 控制输出
func LoggerMiddleware(logger *utils.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 获取响应状态和耗时
		status := c.Writer.Status()
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method

		currentLevel := logger.GetLevel()

		// 1. ERROR 级别：仅输出 5xx 错误
		if currentLevel >= utils.ERROR {
			if status >= 500 {
				logger.Error("[GIN] %d | %v | %s | %s %s", status, latency, clientIP, method, path)
			}
			return
		}

		// 2. WARN 级别：输出 4xx 和 5xx 错误
		if currentLevel == utils.WARN {
			if status >= 400 {
				logger.Warn("[GIN] %d | %v | %s | %s %s", status, latency, clientIP, method, path)
			}
			return
		}

		// 3. INFO 级别：常规业务输出，但过滤高频噪音
		if currentLevel == utils.INFO {
			// 过滤静态文件、健康检查和高频 API (轮询) 请求的正常 200 响应
			isNoise := (path == "/health" ||
				path == "/favicon.ico" ||
				path == "/api/stats" ||
				path == "/api/config" ||
				path == "/api/subscriptions" ||
				len(path) > 7 && path[:8] == "/static/")

			if isNoise && status < 400 {
				return
			}
			logger.Info("[GIN] %d | %v | %s | %s %s", status, latency, clientIP, method, path)
			return
		}

		// 4. DEBUG 级别：输出所有细节
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}
		logger.Debug("[GIN] %d | %13v | %15s | %-7s %s", status, latency, clientIP, method, path)
	}
}

// CORSMiddleware 创建 CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
