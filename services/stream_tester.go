package services

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
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
	st.logger.Info("MaxWorkers updated to %d, maxPerHost=%d", maxWorkers, maxPerHost)
}

// TestStream 测试单个流
func (st *StreamTester) TestStream(rawURL string) (bool, int64, error) {
	start := time.Now()

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	// 1. RTMP 协议检测
	if strings.HasPrefix(strings.ToLower(rawURL), "rtmp") {
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
		conn.Close()
		return true, time.Since(start).Milliseconds(), nil
	}

	// 2. 特殊域名处理 (douyu, huya, bilibili)
	lowerURL := strings.ToLower(rawURL)
	if strings.Contains(lowerURL, "douyu") || strings.Contains(lowerURL, "huya") || strings.Contains(lowerURL, "bilibili") {
		req, err := http.NewRequest("GET", rawURL, nil)
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

	// 3. M3U8 专门处理
	if strings.HasSuffix(lowerURL, ".m3u8") || strings.Contains(lowerURL, ".m3u8?") {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return false, 0, err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := st.client.Do(req)
		if err != nil {
			return false, 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return false, 0, fmt.Errorf("M3U8下载失败: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, 0, err
		}

		m3u8Text := string(body)
		tsURLs := extractTSURLs(rawURL, m3u8Text)

		if len(tsURLs) == 0 {
			return false, 0, fmt.Errorf("未找到TS分片")
		}

		// 只要有一个分片可用即认为可用
		headOK := false
		for _, tsURL := range tsURLs {
			tsReq, _ := http.NewRequest("HEAD", tsURL, nil)
			tsReq.Header.Set("User-Agent", userAgent)
			tsResp, err := st.client.Do(tsReq)
			if err == nil {
				defer tsResp.Body.Close()
				if tsResp.StatusCode == 200 {
					headOK = true
					break
				}
			}
		}

		if !headOK {
			return false, 0, fmt.Errorf("TS分片不可访问")
		}

		// 深度检测
		if st.deepCheck {
			if progressed, _ := st.hlsProgressing(rawURL, userAgent); !progressed {
				return false, 0, fmt.Errorf("疑似循环/停滞的HLS播放列表")
			}
		}

		return true, time.Since(start).Milliseconds(), nil
	}

	// 4. 其他普通 URL 使用 HEAD
	req, err := http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := st.client.Do(req)
	if err != nil {
		// HEAD 失败尝试 GET (有些服务器不响应 HEAD)
		req, _ = http.NewRequest("GET", rawURL, nil)
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

// extractTSURLs 提取 M3U8 中的 TS URL
func extractTSURLs(baseURL string, m3u8Text string) []string {
	var tsURLs []string
	tsReg := regexp.MustCompile(`^[^#?]+\.ts([?#].*)?$`)

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	lines := strings.Split(m3u8Text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		isTS := strings.HasSuffix(strings.ToLower(line), ".ts") ||
			tsReg.MatchString(line) ||
			strings.Contains(strings.ToLower(line), ".ts")

		// 兼容无扩展名的分片
		if !isTS && !strings.Contains(line, "://") {
			isTS = true
		}

		if isTS {
			u, err := url.Parse(line)
			if err == nil {
				tsURLs = append(tsURLs, base.ResolveReference(u).String())
			}
		}

		if len(tsURLs) >= 3 {
			break
		}
	}
	return tsURLs
}

// hlsProgressing 深度检测 HLS 是否在前进
func (st *StreamTester) hlsProgressing(playlistURL string, userAgent string) (bool, error) {
	type signature struct {
		mediaSeq string
		segments []string
		progTime string
	}

	parseSig := func(text string) signature {
		sig := signature{}
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				sig.mediaSeq = strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
			} else if strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:") {
				sig.progTime = strings.TrimPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:")
			} else if line != "" && !strings.HasPrefix(line, "#") {
				sig.segments = append(sig.segments, line)
			}
		}
		if len(sig.segments) > st.segmentWindow {
			sig.segments = sig.segments[len(sig.segments)-st.segmentWindow:]
		}
		return sig
	}

	var signatures []signature
	for i := 0; i < st.loopChecks; i++ {
		req, _ := http.NewRequest("GET", playlistURL, nil)
		req.Header.Set("User-Agent", userAgent)
		resp, err := st.client.Do(req)
		if err == nil {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				// 如果读取失败，认为流是前进的
				return true, nil
			}
			signatures = append(signatures, parseSig(string(body)))
		} else {
			// 如果拉取失败，不做阻断，暂且认为它是前进的
			return true, nil
		}

		if i < st.loopChecks-1 {
			time.Sleep(st.loopInterval)
		}
	}

	if len(signatures) < 2 {
		return true, nil
	}

	first := signatures[0]
	for _, sig := range signatures[1:] {
		// 媒体序列递增 (按字符串比较可能不准，但通常是数字)
		if first.mediaSeq != "" && sig.mediaSeq != "" && sig.mediaSeq > first.mediaSeq {
			return true, nil
		}
		// 分片窗口变化
		if len(sig.segments) != len(first.segments) {
			return true, nil
		}
		for i := range sig.segments {
			if sig.segments[i] != first.segments[i] {
				return true, nil
			}
		}
		// 时间戳变化
		if sig.progTime != "" && first.progTime != "" && sig.progTime != first.progTime {
			return true, nil
		}
	}

	return false, nil
}

// TestSingleChannel 测试单个频道
func (st *StreamTester) TestSingleChannel(ch *models.Channel, testAllSources bool) {
	var mu sync.Mutex
	st.testChannel(ch, testAllSources, &mu)
}

// BatchTest 批量测试流
func (st *StreamTester) BatchTest(channels []*models.Channel, testAllSources bool) ([]*models.Channel, error) {
	if len(channels) == 0 {
		return channels, nil
	}

	// 可选：去重测试 - 避免重复测试相同的 URL
	if !testAllSources {
		channels = st.deduplicateChannels(channels)
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

	for _, channel := range channels {
		wg.Add(1)
		go func(ch *models.Channel) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			st.testChannel(ch, testAllSources, &mu)

			mu.Lock()
			completedCount++
			if ch.IsOnline() {
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

	wg.Wait()
	st.logger.Info("Batch test completed! Total: %d, online: %d, offline: %d", len(channels), onlineCount, offlineCount)

	// 输出限流统计信息
	st.logRateLimiterStats()

	return channels, nil
}

// deduplicateChannels 去重频道 - 避免重复测试相同的 URL
func (st *StreamTester) deduplicateChannels(channels []*models.Channel) []*models.Channel {
	deduped := make(map[string]bool)
	uniqueChannels := make([]*models.Channel, 0, len(channels))

	for _, ch := range channels {
		if len(ch.URLs) > 0 && !deduped[ch.URLs[0]] {
			deduped[ch.URLs[0]] = true
			uniqueChannels = append(uniqueChannels, ch)
		}
	}

	if len(uniqueChannels) < len(channels) {
		st.logger.Info("Deduplicated channels: %d -> %d", len(channels), len(uniqueChannels))
	}

	return uniqueChannels
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

// testChannel 测试单个频道
func (st *StreamTester) testChannel(channel *models.Channel, testAllSources bool, mu *sync.Mutex) {
	var workingURL string
	var responseTime int64
	var isOnline bool
	var lastError error

	if testAllSources {
		// 测试所有 URL
		for _, url := range channel.URLs {
			st.hostCooldown.Wait(url)
			release := st.domainLimiter.Acquire(url)

			online, rt, err := st.TestStream(url)
			st.hostCooldown.Record(url)

			if err == nil && online {
				st.adaptiveLimiter.RecordSuccess()
				workingURL = url
				responseTime = rt
				isOnline = true
				release() // ✅ 立即释放，不使用 defer
				break
			} else if err != nil {
				// 检测是否是限流错误
				if isRateLimitError(err) {
					st.adaptiveLimiter.RecordError()
					st.logger.Warn("Rate limit detected for URL: %s, error: %v", url, err)
				}
				lastError = err
			}

			release() // ✅ 失败后立即释放，继续下一个 URL
		}
	} else {
		// 只测试第一个 URL
		if len(channel.URLs) > 0 {
			url := channel.URLs[0]
			st.hostCooldown.Wait(url)
			release := st.domainLimiter.Acquire(url)

			online, rt, err := st.TestStream(url)
			st.hostCooldown.Record(url)

			if err == nil && online {
				st.adaptiveLimiter.RecordSuccess()
				workingURL = url
				responseTime = rt
				isOnline = true
			} else {
				if isRateLimitError(err) {
					st.adaptiveLimiter.RecordError()
					st.logger.Warn("Rate limit detected for URL: %s, error: %v", url, err)
				}
				lastError = err
			}

			release() // ✅ 显式释放
		}
	}

	// 更新测试结果
	mu.Lock()
	defer mu.Unlock()

	status := "offline"
	if isOnline {
		status = "online"
	}

	channel.UpdateTestResult(status, workingURL, responseTime)
	if lastError != nil && !isOnline {
		channel.TestResults.Details = lastError.Error()
	}
}

// isRateLimitError 检测是否是限流错误（只检测明确的限流错误）
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 只检测明确的限流错误，避免误判网络错误
	return contains(errStr, "429") ||
		contains(errStr, "403") ||
		contains(errStr, "Too Many Requests") ||
		contains(errStr, "Forbidden")
}

// contains 检查字符串是否包含子串
func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
