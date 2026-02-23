package services

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"iptv-aggregator/models"
	"iptv-aggregator/services/stream_tester"
	"iptv-aggregator/utils"
)

// StreamTester 流测试器
type StreamTester struct {
	timeout       time.Duration
	maxWorkers    int
	client        *http.Client
	deepCheck     bool
	loopChecks    int
	loopInterval  time.Duration
	segmentWindow int
	logger        *utils.Logger
	// 限流相关
	domainLimiter   *DomainLimiter
	adaptiveLimiter *AdaptiveLimiter
	hostCooldown    *HostCooldown
	// HLS 检查器
	hlsChecker *HLSChecker
}

// NewStreamTester 创建新的流测试器
func NewStreamTester(timeout time.Duration, maxWorkers int) *StreamTester {
	// 计算单主机最大并发数（总并发的 1/4，最少 2，最多 10）
	maxPerHost := max(2, min(10, maxWorkers/4))

	st := &StreamTester{
		timeout:         timeout,
		maxWorkers:      maxWorkers,
		deepCheck:       false,
		loopChecks:      3,
		loopInterval:    4 * time.Second,
		segmentWindow:   5,
		logger:          utils.NewLogger(),
		domainLimiter:   NewDomainLimiter(maxPerHost),
		adaptiveLimiter: NewAdaptiveLimiter(maxWorkers),
		hostCooldown:    NewHostCooldown(100 * time.Millisecond), // 100ms 冷却时间
	}
	st.client = stream_tester.CreateHTTPClient(timeout, maxWorkers)
	st.hlsChecker = NewHLSChecker(st.client, st.deepCheck, st.loopChecks, st.loopInterval, st.segmentWindow, st.logger)
	st.logger.Info("StreamTester initialized: maxWorkers=%d, maxPerHost=%d", maxWorkers, maxPerHost)
	return st
}

// SetDeepCheckOptions 设置深度检查选项
func (st *StreamTester) SetDeepCheckOptions(enabled bool, checks int, interval time.Duration, window int) {
	st.deepCheck = enabled
	st.loopChecks = checks
	st.loopInterval = interval
	st.segmentWindow = window
	// 更新 HLSChecker 的配置
	st.hlsChecker = NewHLSChecker(st.client, st.deepCheck, st.loopChecks, st.loopInterval, st.segmentWindow, st.logger)
}

// SetStreamTestTimeout 设置流测试超时时间
func (st *StreamTester) SetStreamTestTimeout(timeout time.Duration) {
	st.timeout = timeout
	st.client = stream_tester.CreateHTTPClient(timeout, st.maxWorkers)
}

// SetMaxWorkers 设置最大并发工作数
func (st *StreamTester) SetMaxWorkers(maxWorkers int) {
	st.maxWorkers = maxWorkers
	// 重新创建 HTTP 客户端，以适应新的并发数
	st.client = stream_tester.CreateHTTPClient(st.timeout, maxWorkers)
	// 更新自适应限流器
	st.adaptiveLimiter = NewAdaptiveLimiter(maxWorkers)
	// 更新域名限流器
	maxPerHost := max(2, min(10, maxWorkers/4))
	st.domainLimiter = NewDomainLimiter(maxPerHost)
	// 更新 HLSChecker 的客户端
	st.hlsChecker = NewHLSChecker(st.client, st.deepCheck, st.loopChecks, st.loopInterval, st.segmentWindow, st.logger)
	st.logger.Info("MaxWorkers updated to %d, maxPerHost=%d", maxWorkers, maxPerHost)
}

// TestStream 测试单个流 - 添加超时保护
func (st *StreamTester) TestStream(rawURL string) (bool, int64, error) {
	start := time.Now()

	// 总体超时保护
	ctx, cancel := context.WithTimeout(context.Background(), st.timeout)
	defer cancel()

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	// 1. RTMP 协议检测
	if strings.HasPrefix(strings.ToLower(rawURL), "rtmp") {
		return stream_tester.TestRTMP(rawURL, st.timeout)
	}

	// 2. 特殊域名处理 (douyu, huya, bilibili)
	lowerURL := strings.ToLower(rawURL)
	if strings.Contains(lowerURL, "douyu") || strings.Contains(lowerURL, "huya") || strings.Contains(lowerURL, "bilibili") {
		return stream_tester.TestSpecialDomain(ctx, st.client, rawURL, userAgent, start)
	}

	// 3. M3U8 专门处理
	if strings.HasSuffix(lowerURL, ".m3u8") || strings.Contains(lowerURL, ".m3u8?") {
		return st.hlsChecker.CheckM3U8(ctx, rawURL, userAgent)
	}

	// 4. 其他普通 URL 使用 HEAD
	return stream_tester.TestGenericURL(ctx, st.client, rawURL, userAgent, start)
}

// TestSingleChannel 测试单个频道
func (st *StreamTester) TestSingleChannel(ch *models.Channel, testAllSources bool) {
	channelTester := NewChannelTester(st)
	channelTester.TestChannel(ch, testAllSources)
}

// BatchTest 批量测试流
func (st *StreamTester) BatchTest(channels []*models.Channel, testAllSources bool) ([]*models.Channel, error) {
	if len(channels) == 0 {
		return channels, nil
	}

	// 可选：去重测试 - 避免重复测试相同的 URL
	channelTester := NewChannelTester(st)
	var skippedChannels []*models.Channel
	if !testAllSources {
		channels, skippedChannels = channelTester.DeduplicateChannels(channels)
	}

	st.logger.Info("Starting batch test for %d channels", len(channels))

	// 创建工作池，使用自适应并发数
	effectiveWorkers := st.adaptiveLimiter.GetEffectiveWorkers()
	semaphore := make(chan struct{}, effectiveWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	completedCount := 0
	onlineCount := 0
	offlineCount := 0

	// 启动诊断 goroutine - 监控测试进度
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 计算合理的超时时间
	timeout := time.Duration(len(channels)) * time.Second / time.Duration(effectiveWorkers)
	timeout = max(timeout, 5*time.Minute) // 最少 5 分钟

	for _, channel := range channels {
		wg.Add(1)
		go func(ch *models.Channel) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := channelTester.TestChannel(ch, testAllSources)

			mu.Lock()
			completedCount++
			if result.IsOnline {
				onlineCount++
			} else {
				offlineCount++
			}

			if completedCount%10 == 0 || completedCount == len(channels) {
				st.logger.Info("Test progress: %d/%d (online: %d, offline: %d, effective workers: %d)",
					completedCount, len(channels), onlineCount, offlineCount, st.adaptiveLimiter.GetEffectiveWorkers())
			}
			mu.Unlock()
		}(channel)
	}

	// 等待完成或超时
	select {
	case <-done:
		st.logger.Info("Batch test completed successfully! Total: %d, online: %d, offline: %d", len(channels), onlineCount, offlineCount)
	case <-time.After(timeout):
		st.logger.Warn("Batch test timeout detected! Some goroutines may be blocked. Timeout: %v", timeout)
		st.logger.Warn("Domain limiter stats: %v", st.domainLimiter.GetStats())
		st.logger.Warn("Effective workers: %d", st.adaptiveLimiter.GetEffectiveWorkers())
		// ✅ 继续等待完成，但记录了超时警告
		// 这样可以获得完整的测试结果，同时提醒用户可能有性能问题
		<-done
		st.logger.Info("Batch test finally completed after timeout. Total: %d, online: %d, offline: %d", len(channels), onlineCount, offlineCount)
	}

	// 输出限流统计信息
	st.logRateLimiterStats()

	// 同步测试结果到被跳过的频道（共享相同URL的频道）
	if len(skippedChannels) > 0 {
		stream_tester.SyncTestResultsToSkippedChannels(channels, skippedChannels, st.logger)
	}

	return channels, nil
}

// logRateLimiterStats 输出限流统计信息
func (st *StreamTester) logRateLimiterStats() {
	domainStats := st.domainLimiter.GetStats()
	if len(domainStats) > 0 {
		st.logger.Info("Domain limiter stats: %v", domainStats)
	}

	cooldownStats := st.hostCooldown.GetStats()
	if len(cooldownStats) > 0 {
		st.logger.Debug("Host cooldown stats: %d hosts tracked", len(cooldownStats))
	}
}
