package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"iptv-aggregator/config"
	"iptv-aggregator/handlers"
	"iptv-aggregator/services"
	"iptv-aggregator/utils"

	"github.com/gin-gonic/gin"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// 命令行标志
	configPath := flag.String("config", "config.json", "Configuration file path")
	port := flag.Int("port", 8080, "Server port")
	version := flag.Bool("version", false, "Show version")
	help := flag.Bool("help", false, "Show help")
	serviceCmd := flag.String("s", "", "Service management: install|uninstall|start|stop|restart|status")

	flag.Parse()

	if *help {
		flag.PrintDefaults()
		fmt.Println("\nService management (Linux only):")
		fmt.Println("  -s install      Install as system service")
		fmt.Println("  -s uninstall    Uninstall system service")
		fmt.Println("  -s start        Start service")
		fmt.Println("  -s stop         Stop service")
		fmt.Println("  -s restart      Restart service")
		fmt.Println("  -s status       Show service status")
		os.Exit(0)
	}

	if *version {
		fmt.Printf("IPTV M3U Aggregator\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// 初始化日志
	logger := utils.NewLogger()

	// 处理服务管理命令
	if *serviceCmd != "" {
		execPath, err := os.Executable()
		if err != nil {
			logger.Error("Failed to get executable path: %v", err)
			os.Exit(1)
		}
		if err := utils.HandleServiceCommand(*serviceCmd, execPath, *configPath, logger); err != nil {
			logger.Error("Service command failed: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	logger.Info("Starting IPTV M3U Aggregator...")
	logger.Info("Version: %s, Build Time: %s, Git Commit: %s", Version, BuildTime, GitCommit)

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// 覆盖端口
	if *port != 8080 {
		cfg.Port = *port
	}

	// 根据配置设置全局日志级别
	level := utils.ParseLevel(cfg.LogLevel)
	utils.SetGlobalLevel(level)
	logger.SetLevel(level)

	logger.Info("Configuration loaded successfully")
	logger.Info("Server will listen on %s:%d", cfg.Host, cfg.Port)

	// 初始化服务
	logger.Info("Initializing services...")
	subscriptionMgr := services.NewSubscriptionManager(cfg.DataDir)
	parser := services.NewM3UParser(cfg.RequestTimeout)
	aggregator := services.NewChannelAggregator(cfg.DataDir)
	tester := services.NewStreamTester(cfg.StreamTestTimeout, cfg.MaxTestWorkers)
	tester.SetDeepCheckOptions(cfg.DeepCheck, cfg.LoopChecks, cfg.LoopInterval, cfg.SegmentWindow)
	scheduler := services.NewScheduler()
	exporter := services.NewChannelExporter(cfg.DataDir)

	logger.Info("Services initialized successfully")

	// 如果启用了流测试，在启动时根据配置决定是否进行测试
	if cfg.EnableStreamTest {
		channels := aggregator.GetAllChannels()
		hasResults := aggregator.HasTestResults()
		lastTestTime := aggregator.GetLastTestTime()

		// 判断是否需要启动时自动测试
		shouldAutoTest := cfg.AutoTestOnStartup &&
			(len(channels) == 0 || !hasResults ||
				time.Since(lastTestTime) > time.Duration(cfg.AutoTestIntervalHours)*time.Hour)

		if shouldAutoTest {
			logger.Info("Auto-testing channels on startup (no results or results expired)...")
			aggregator.ResetTestResults()
			tested, err := tester.BatchTest(channels, cfg.TestAllSources)
			if err != nil {
				logger.Error("Startup auto-test failed: %v", err)
			} else {
				logger.Info("Startup auto-test completed: %d channels tested", len(tested))
				aggregator.Save()
			}
		} else {
			if hasResults {
				logger.Info("Skipping startup test, using existing results (last test: %v ago)",
					time.Since(lastTestTime).Round(time.Minute))
			} else {
				logger.Info("Skipping startup test (auto_test_on_startup disabled)")
			}
		}
	}

	// 初始化 HTTP 服务器
	logger.Info("Setting up HTTP server...")

	// 根据日志级别设置 Gin 模式
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode) // release 模式不会输出路由信息
	}

	// 禁用 Gin 的默认输出，除非是 debug 模式
	if cfg.LogLevel != "debug" {
		gin.DefaultWriter = io.Discard
	}

	router := handlers.SetupRouter(
		subscriptionMgr,
		parser,
		aggregator,
		tester,
		scheduler,
		exporter,
		cfg,
		logger,
	)

	// 注册定时任务
	logger.Info("Registering scheduled jobs...")

	// 自动更新订阅源任务
	scheduler.AddJob("update_subscriptions", cfg.UpdateInterval.String(), func() error {
		logger.Info("Starting scheduled subscription update...")
		subs := subscriptionMgr.GetAllSubscriptions()
		for _, sub := range subs {
			if !sub.Enabled {
				continue
			}
			logger.Info("Updating subscription: %s", sub.Name)
			content, err := parser.FetchM3U(sub.URL)
			if err != nil {
				logger.Error("Failed to fetch M3U for %s: %v", sub.Name, err)
				continue
			}
			channels, err := parser.ParseM3U(content, sub.URL)
			if err != nil {
				logger.Error("Failed to parse M3U for %s: %v", sub.Name, err)
				continue
			}
			added, updated, _, _ := aggregator.AggregateChannels(channels, cfg.MatchBy, cfg.SimilarityThreshold)
			logger.Info("Subscription %s updated: %d added, %d updated", sub.Name, added, updated)

			// 更新频道计数
			count := aggregator.GetChannelCountBySource(sub.URL)
			subscriptionMgr.UpdateSubscriptionStatus(sub.URL, "active", count)
		}
		return nil
	})

	// 自动测试流任务
	if cfg.EnableStreamTest {
		scheduler.AddJob("test_streams", cfg.TestInterval.String(), func() error {
			logger.Info("Starting scheduled stream test...")
			channels := aggregator.GetAllChannels()
			if len(channels) == 0 {
				logger.Info("No channels to test")
				return nil
			}

			// 定时测试前也进行复位
			aggregator.ResetTestResults()

			tested, err := tester.BatchTest(channels, cfg.TestAllSources)
			if err != nil {
				logger.Error("Scheduled stream test failed: %v", err)
				return err
			}
			logger.Info("Scheduled stream test completed: %d channels tested", len(tested))
			// 保存测试结果
			return aggregator.Save()
		})
	}

	// 启动调度器
	logger.Info("Starting scheduler...")
	if err := scheduler.Start(); err != nil {
		logger.Error("Failed to start scheduler: %v", err)
		os.Exit(1)
	}

	// 获取本机 IP 和完整地址（在启动服务器前）
	var localIP string
	if cfg.Host == "0.0.0.0" || cfg.Host == "" {
		localIP = utils.GetBestLocalIP()
	} else {
		localIP = cfg.Host
	}

	primaryAddr := utils.GetPrimaryAddress(localIP, cfg.Port)
	playlistURL := utils.GetPlaylistURL(localIP, cfg.Port, "/playlist.m3u")

	logger.Info("Local IP: %s", localIP)
	logger.Info("Server listening on %s", primaryAddr)
	logger.Info("Subscription URL: %s", playlistURL)

	// 启动服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		if err := router.Run(addr); err != nil {
			logger.Error("Server error: %v", err)
		}
	}()

	// 优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	logger.Info("Shutting down...")
	scheduler.Stop()
	logger.Info("Goodbye!")
}
