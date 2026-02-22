package services

import "fmt"

// StreamTestError 流测试错误类型
type StreamTestError struct {
	Code    string // 错误代码
	Message string // 错误信息
	Err     error  // 原始错误
}

// Error 实现 error 接口
func (e *StreamTestError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is 和 errors.As
func (e *StreamTestError) Unwrap() error {
	return e.Err
}

// 错误代码常量
const (
	ErrCodeRateLimited      = "RATE_LIMITED"      // 被限流
	ErrCodeManifestInvalid  = "MANIFEST_INVALID"  // 播放列表无效
	ErrCodeNoSegments       = "NO_SEGMENTS"       // 未找到分片
	ErrCodeSegmentFailed    = "SEGMENT_FAILED"    // 分片不可访问
	ErrCodeConnectionFailed = "CONNECTION_FAILED" // 连接失败
	ErrCodeTimeout          = "TIMEOUT"           // 超时
	ErrCodeInvalidURL       = "INVALID_URL"       // 无效 URL
	ErrCodeHTTPError        = "HTTP_ERROR"        // HTTP 错误
	ErrCodeRTMPFailed       = "RTMP_FAILED"       // RTMP 连接失败
	ErrCodePlaylistStalled  = "PLAYLIST_STALLED"  // 播放列表停滞
)

// NewStreamTestError 创建新的流测试错误
func NewStreamTestError(code, message string, err error) *StreamTestError {
	return &StreamTestError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsRateLimitedError 检查是否是限流错误
func IsRateLimitedError(err error) bool {
	if err == nil {
		return false
	}
	if ste, ok := err.(*StreamTestError); ok {
		return ste.Code == ErrCodeRateLimited
	}
	return false
}

// IsManifestInvalidError 检查是否是播放列表无效错误
func IsManifestInvalidError(err error) bool {
	if err == nil {
		return false
	}
	if ste, ok := err.(*StreamTestError); ok {
		return ste.Code == ErrCodeManifestInvalid || ste.Code == ErrCodeNoSegments
	}
	return false
}

// IsConnectionError 检查是否是连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if ste, ok := err.(*StreamTestError); ok {
		return ste.Code == ErrCodeConnectionFailed || ste.Code == ErrCodeRTMPFailed
	}
	return false
}
