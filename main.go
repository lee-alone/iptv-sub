package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"iptv-aggregator/config"
	"iptv-aggregator/handlers"
	"iptv-aggregator/services"
	"iptv-aggregator/utils"
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

	flag.Parse()

	if *help {
		flag.PrintDefaults()
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

	// 如果启用了流测试，在启动时重置所有状态，确保不会显示旧的离线/在线数据
	if cfg.EnableStreamTest {
		logger.Info("Stream testing enabled, resetting all channel states for a clean start...")
		aggregator.ResetTestResults()
	}

	// 初始化 HTTP 服务器
	logger.Info("Setting up HTTP server...")
	router := handlers.SetupRouter(
		subscriptionMgr,
		parser,
		aggregator,
		tester,
		scheduler,
		exporter,
		cfg,
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

	// 启动服务器
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		logger.Info("Server listening on http://%s", addr)
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
