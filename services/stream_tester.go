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
}

// NewStreamTester 创建新的流测试器
func NewStreamTester(timeout time.Duration, maxWorkers int) *StreamTester {
	return &StreamTester{
		timeout:    timeout,
		maxWorkers: maxWorkers,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				ResponseHeaderTimeout: timeout,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				DisableKeepAlives:     false,
				DialContext: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
		// 默认配置，可以通过后接的配置方法修改
		deepCheck:     false,
		loopChecks:    3,
		loopInterval:  4 * time.Second,
		segmentWindow: 5,
		logger:        utils.NewLogger(),
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
	st.client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: timeout,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			DisableKeepAlives:     false,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}
}

// SetMaxWorkers 设置最大并发工作数
func (st *StreamTester) SetMaxWorkers(maxWorkers int) {
	st.maxWorkers = maxWorkers
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

	st.logger.Info("开始批量测试 %d 个频道", len(channels))

	// 创建工作池
	semaphore := make(chan struct{}, st.maxWorkers)
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
				st.logger.Info("测试进度: %d/%d (在线: %d, 离线: %d)", completedCount, len(channels), onlineCount, offlineCount)
			}
			mu.Unlock()
		}(channel)
	}

	wg.Wait()
	st.logger.Info("所有频道测试完成! 总计: %d, 在线: %d, 离线: %d", len(channels), onlineCount, offlineCount)
	return channels, nil
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
			online, rt, err := st.TestStream(url)
			if err == nil && online {
				workingURL = url
				responseTime = rt
				isOnline = true
				break
			} else if err != nil {
				lastError = err
			}
		}
	} else {
		// 只测试第一个 URL
		if len(channel.URLs) > 0 {
			online, rt, err := st.TestStream(channel.URLs[0])
			if err == nil && online {
				workingURL = channel.URLs[0]
				responseTime = rt
				isOnline = true
			} else if err != nil {
				lastError = err
			}
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
