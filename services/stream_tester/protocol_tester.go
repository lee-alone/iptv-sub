package stream_tester

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// TestRTMP 测试 RTMP 协议
func TestRTMP(rawURL string, timeout time.Duration) (bool, int64, error) {
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

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false, 0, fmt.Errorf("RTMP端口不可达: %w", err)
	}
	defer conn.Close()
	return true, time.Since(start).Milliseconds(), nil
}

// TestSpecialDomain 测试特殊域名
func TestSpecialDomain(ctx context.Context, client *http.Client, rawURL, userAgent string, start time.Time) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
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

// TestGenericURL 测试普通 URL
func TestGenericURL(ctx context.Context, client *http.Client, rawURL, userAgent string, start time.Time) (bool, int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		// HEAD 失败尝试 GET (有些服务器不响应 HEAD)
		req, _ = http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		req.Header.Set("User-Agent", userAgent)
		resp, err = client.Do(req)
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
