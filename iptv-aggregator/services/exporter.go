package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/iptv-aggregator/models"
)

// ChannelExporter 频道导出器
type ChannelExporter struct {
	dataDir string
}

// NewChannelExporter 创建新的频道导出器
func NewChannelExporter(dataDir string) *ChannelExporter {
	return &ChannelExporter{
		dataDir: dataDir,
	}
}

// ExportM3U 导出为 M3U 格式
func (ce *ChannelExporter) ExportM3U(channels []*models.Channel, onlyWorking bool) (string, error) {
	var sb strings.Builder

	// 写入 M3U 头
	sb.WriteString("#EXTM3U\n")

	for _, ch := range channels {
		// 如果只导出在线频道，跳过离线频道
		if onlyWorking && !ch.IsOnline() {
			continue
		}

		// 构建 EXTINF 行
		extinf := "#EXTINF:-1"

		if ch.TvgID != "" {
			extinf += fmt.Sprintf(` tvg-id="%s"`, ch.TvgID)
		}
		if ch.TvgName != "" {
			extinf += fmt.Sprintf(` tvg-name="%s"`, ch.TvgName)
		}
		if ch.TvgLogo != "" {
			extinf += fmt.Sprintf(` tvg-logo="%s"`, ch.TvgLogo)
		}
		if ch.GroupTitle != "" {
			extinf += fmt.Sprintf(` group-title="%s"`, ch.GroupTitle)
		}

		extinf += fmt.Sprintf(",%s\n", ch.Name)
		sb.WriteString(extinf)

		// 选择 URL
		url := ce.selectURL(ch)
		if url != "" {
			sb.WriteString(url + "\n")
		}
	}

	// 保存到文件
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("iptv_export_%s.m3u", timestamp)
	filePath := filepath.Join(ce.dataDir, "exports", filename)

	// 确保导出目录存在
	os.MkdirAll(filepath.Dir(filePath), 0755)

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write M3U file: %w", err)
	}

	return filePath, nil
}

// ExportJSON 导出为 JSON 格式
func (ce *ChannelExporter) ExportJSON(channels []*models.Channel, onlyWorking bool) (string, error) {
	var exportChannels []*models.Channel

	for _, ch := range channels {
		// 如果只导出在线频道，跳过离线频道
		if onlyWorking && !ch.IsOnline() {
			continue
		}
		exportChannels = append(exportChannels, ch)
	}

	data, err := json.MarshalIndent(exportChannels, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal channels: %w", err)
	}

	// 保存到文件
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("iptv_export_%s.json", timestamp)
	filePath := filepath.Join(ce.dataDir, "exports", filename)

	// 确保导出目录存在
	os.MkdirAll(filepath.Dir(filePath), 0755)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	return filePath, nil
}

// selectURL 选择最佳 URL
func (ce *ChannelExporter) selectURL(ch *models.Channel) string {
	// 优先使用测试中工作的 URL
	if ch.TestResults != nil && ch.TestResults.WorkingURL != "" {
		return ch.TestResults.WorkingURL
	}

	// 否则使用第一个 URL
	if len(ch.URLs) > 0 {
		return ch.URLs[0]
	}

	return ""
}
