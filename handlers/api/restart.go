package api

import (
	"net/http"
	"os"
	"time"

	"iptv-aggregator/config"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

// RestartHandlers 重启相关处理器
type RestartHandlers struct {
	cfg    *config.Config
	logger *utils.Logger
}

// NewRestartHandlers 创建重启处理器
func NewRestartHandlers(
	cfg *config.Config,
	logger *utils.Logger,
) *RestartHandlers {
	return &RestartHandlers{
		cfg:    cfg,
		logger: logger,
	}
}

// RegisterRoutes 注册重启相关路由
func (h *RestartHandlers) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/restart", h.restart)
}

// restart 重启服务器
func (h *RestartHandlers) restart(c *gin.Context) {
	logger := utils.NewLogger()
	logger.Info("Restart requested via API...")

	c.JSON(http.StatusOK, gin.H{"message": "restarting server..."})

	// 给前端一点时间接收响应
	go func() {
		time.Sleep(1 * time.Second)
		logger.Info("Server is restarting!")

		// 获取当前程序的路径和参数
		args := os.Args
		cwd, _ := os.Getwd()

		// 启动新进程
		attr := os.ProcAttr{
			Dir:   cwd,
			Env:   os.Environ(),
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		}

		process, err := os.StartProcess(args[0], args, &attr)
		if err != nil {
			logger.Error("Failed to restart: %v", err)
			return
		}

		logger.Info("New process started with PID: %d", process.Pid)
		os.Exit(0)
	}()
}
