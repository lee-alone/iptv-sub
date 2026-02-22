package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"iptv-aggregator/models"
	"iptv-aggregator/utils"
)

// M3UParser M3U 文件解析器
type M3UParser struct {
	timeout time.Duration
	client  *http.Client
	logger  *utils.Logger
}

// NewM3UParser 创建新的 M3U 解析器
func NewM3UParser(timeout time.Duration) *M3UParser {
	return &M3UParser{
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: utils.NewLogger(),
	}
}

// FetchM3U 从 URL 获取 M3U 文件
func (p *M3UParser) FetchM3U(url string) (string, error) {
	p.logger.Debug("Fetching M3U from %s", url)
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch M3U: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch M3U: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read M3U content: %w", err)
	}

	return string(body), nil
}

// ParseM3U 解析 M3U 文件内容
func (p *M3UParser) ParseM3U(content string, sourceURL string) ([]*models.Channel, error) {
	var channels []*models.Channel
	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentInfo string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#EXTINF")) {
			continue
		}

		// 解析频道信息行
		if strings.HasPrefix(line, "#EXTINF:") {
			currentInfo = line
			continue
		}

		// 解析 URL 行
		if line != "" && !strings.HasPrefix(line, "#") && currentInfo != "" {
			channel := p.parseChannelLine(currentInfo, line, sourceURL)
			if channel != nil {
				channels = append(channels, channel)
			}
			currentInfo = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse M3U: %w", err)
	}

	return channels, nil
}

// parseChannelLine 解析单个频道行
func (p *M3UParser) parseChannelLine(infoLine, urlLine, sourceURL string) *models.Channel {
	// 解析 EXTINF 行
	// 格式: #EXTINF:-1 tvg-id="..." tvg-name="..." tvg-logo="..." group-title="...",频道名称
	info := strings.TrimPrefix(infoLine, "#EXTINF:")
	parts := strings.Split(info, ",")

	if len(parts) < 2 {
		return nil
	}

	// 提取属性
	attrs := parts[0]
	name := strings.TrimSpace(parts[1])

	if name == "" {
		return nil
	}

	tvgID := p.extractAttr(attrs, "tvg-id")
	tvgName := p.extractAttr(attrs, "tvg-name")
	tvgLogo := p.extractAttr(attrs, "tvg-logo")
	groupTitle := p.extractAttr(attrs, "group-title")

	return models.NewChannel(name, groupTitle, tvgID, tvgName, tvgLogo, urlLine, sourceURL)
}

// extractAttr 从属性字符串中提取属性值
func (p *M3UParser) extractAttr(attrs, attrName string) string {
	pattern := fmt.Sprintf(`%s="([^"]*)"`, attrName)
	start := strings.Index(attrs, pattern[:len(attrName)+2])
	if start == -1 {
		return ""
	}

	// 简单的属性提取
	searchStr := fmt.Sprintf(`%s="`, attrName)
	idx := strings.Index(attrs, searchStr)
	if idx == -1 {
		return ""
	}

	start = idx + len(searchStr)
	end := strings.Index(attrs[start:], `"`)
	if end == -1 {
		return ""
	}

	return attrs[start : start+end]
}
