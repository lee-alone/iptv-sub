package stream_tester

import (
	"net"
	"net/http"
	"time"
)

// CreateHTTPClient 创建 HTTP 客户端，根据 maxWorkers 自适应连接池大小
func CreateHTTPClient(timeout time.Duration, maxWorkers int) *http.Client {
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
