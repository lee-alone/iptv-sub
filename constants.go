package main

// 服务器配置常量
const (
	DefaultPort = 8080
	DefaultHost = "0.0.0.0"
)

// 文件权限常量
const (
	DirPermission  = 0755
	FilePermission = 0644
)

// 频道状态常量
const (
	ChannelStatusOnline   = "online"
	ChannelStatusOffline  = "offline"
	ChannelStatusUntested = "untested"
)

// 订阅源状态常量
const (
	SubscriptionStatusActive = "active"
	SubscriptionStatusFailed = "failed"
)

// HTTP 相关常量
const (
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	RTMPDefaultPort  = "1935"
)

// 错误消息常量
const (
	ErrRTMPPortUnreachable  = "RTMP port unreachable"
	ErrHTTPError            = "HTTP error"
	ErrM3U8DownloadFailed   = "M3U8 download failed"
	ErrTSSegmentNotFound    = "TS segment not found"
	ErrTSSegmentUnavailable = "TS segment unavailable"
	ErrHLSStalled           = "HLS playlist appears to be stalled or looping"
)
