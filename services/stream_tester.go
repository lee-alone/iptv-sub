package services

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"iptv-aggregator/models"
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
	st.client = st.createHTTPClient(timeout, maxWorkers)
	st.hlsChecker = NewHLSChecker(st.client, st.deepCheck, st.loopChecks, st.loopInterval, st.segmentWindow, st.logger)
	st.logger.Info("StreamTester initialized: maxWorkers=%d, maxPerHost=%d", maxWorkers, maxPerHost)
	return st
}

// createHTTPClient 创建 HTTP 客户端，根据 maxWorkers 自适应连接池大小
func (st *StreamTester) createHTTPClient(timeout time.Duration, maxWorkers int) *http.Client {
	// MaxIdleConnsPerHost 应该至少等于 maxWorkers，以支持并发测试
	// 如果 maxWorkers 很大，也要考虑系统资源，设置一个合理的上限
	maxConnsPerHost := maxWorkers
	if maxConnsPerHost < 10 {
		maxConnsPerHost = 10 // 最小值
	}
	if maxConnsPerHost > 100 {
		maxConnsPerHost = 100 // 最大值
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: timeout,
			IdleConnTimeout:       30 * time.Second,    // 空闲连接超时，防止僵尸连接
			TLSHandshakeTimeout:   10 * time.Second,    // TLS 握手超时
			MaxIdleConns:          maxConnsPerHost * 2, // 总连接数是单主机的 2 倍
			MaxIdleConnsPerHost:   maxConnsPerHost,
			DisableKeepAlives:     false,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
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
	st.client = st.createHTTPClient(timeout, st.maxWorkers)
}

// SetMaxWorkers 设置最大并发工作数
func (st *StreamTester) SetMaxWorkers(maxWorkers int) {
	st.maxWorkers = maxWorkers
	// 重新创建 HTTP 客户端，以适应新的并发数
	st.client = st.createHTTPClient(st.timeout, maxWorkers)
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
		return st.testRTMP(rawURL)
	}

	// 2. 特殊域名处理 (douyu, huya, bilibili)
	lowerURL := strings.ToLower(rawURL)
	if strings.Contains(lowerURL, "douyu") || strings.Contains(lowerURL, "huya") || strings.Contains(lowerURL, "bilibili") {
		return st.testSpecialDomain(ctx, rawURL, userAgent, start)
	}

	// 3. M3U8 专门处理
	if strings.HasSuffix(lowerURL, ".m3u8") || strings.Contains(lowerURL, ".m3u8?") {
		return st.hlsChecker.CheckM3U8(ctx, rawURL, userAgent)
	}

	// 4. 其他普通 URL 使用 HEAD
	return st.testGenericURL(ctx, rawURL, userAgent, start)
}

// testRTMP 测试 RTMP 协议
func (st *StreamTester) testRTMP(rawURL string) (bool, int64, error) {
	start := time.Now()

	u, err := url.Parse(rawURL)
	if err != nil {
		return false, 0, fmt.Errorf("invalid rtmp url: %w", err)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "1935"
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), st.timeout)
	if err != nil {
		return false, 0, fmt.Errorf("RTMP端口不可达: %w", err)
	}
	defer conn.Close()
	return true, time.Since(start).Milliseconds(), nil
}

// testSpecialDomain 测试特殊域名
func (st *StreamTester) testSpecialDomain(ctx context.Context, rawURL, userAgent string, start time.Time) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := st.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// 尝试读取一小块数据确认是否有内容
		buf := make([]byte, 1024)
		_, _ = resp.Body.Read(buf)
		return true, time.Since(start).Milliseconds(), nil
	}
	return false, 0, fmt.Errorf("HTTP错误: %d", resp.StatusCode)
}

// testGenericURL 测试普通 URL
func (st *StreamTester) testGenericURL(ctx context.Context, rawURL, userAgent string, start time.Time) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := st.client.Do(req)
	if err != nil {
		// HEAD 失败尝试 GET (有些服务器不响应 HEAD)
		req, _ = http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		req.Header.Set("User-Agent", userAgent)
		resp, err = st.client.Do(req)
		if err != nil {
			return false, 0, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, time.Since(start).Milliseconds(), nil
	}

	return false, 0, fmt.Errorf("HTTP错误: %d", resp.StatusCode)
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
	if !testAllSources {
		channels = channelTester.DeduplicateChannels(channels)
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
