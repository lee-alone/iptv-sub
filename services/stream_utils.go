package services

import (
	"net/url"
	"regexp"
	"strings"
)

// extractTSURLs 提取 M3U8 中的 TS URL，支持 Master Playlist 和 Media Playlist
func extractTSURLs(baseURL string, m3u8Text string) []string {
	var tsURLs []string
	tsReg := regexp.MustCompile(`^[^#?]+\.ts([?#].*)?$`)

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}

	lines := strings.Split(m3u8Text, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是 TS 分片
		isTS := strings.HasSuffix(strings.ToLower(line), ".ts") ||
			tsReg.MatchString(line) ||
			strings.Contains(strings.ToLower(line), ".ts")

		// 兼容无扩展名的分片
		if !isTS && !strings.Contains(line, "://") {
			isTS = true
		}

		// 检查是否是 Master Playlist 中的媒体播放列表 URL（EXT-X-STREAM-INF 后的行）
		if !isTS && i > 0 {
			prevLine := strings.TrimSpace(lines[i-1])
			if strings.HasPrefix(prevLine, "#EXT-X-STREAM-INF") {
				// 这是一个媒体播放列表 URL，可能是 .m3u8 或其他格式
				isTS = true
			}
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

// isRateLimitError 检测是否是限流错误（只检测明确的限流错误）
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 只检测明确的限流错误，避免误判网络错误
	return strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "Too Many Requests") ||
		strings.Contains(errStr, "Forbidden")
}
