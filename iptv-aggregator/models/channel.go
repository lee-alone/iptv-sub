package models

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// TestResult 测试结果
type TestResult struct {
	Status       string    `json:"status"` // online, offline, untested
	WorkingURL   string    `json:"working_url,omitempty"`
	ResponseTime int64     `json:"response_time_ms,omitempty"`
	TestedAt     time.Time `json:"tested_at"`
	Details      string    `json:"details,omitempty"`
}

// Channel 频道模型
type Channel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	GroupTitle  string            `json:"group_title"`
	TvgID       string            `json:"tvg_id"`
	TvgName     string            `json:"tvg_name"`
	TvgLogo     string            `json:"tvg_logo"`
	URLs        []string          `json:"urls"`
	BackupURLs  []string          `json:"backup_urls,omitempty"` // 备用URLs，保守模式使用
	SourceURLs  map[string]string `json:"source_urls"`           // url -> source subscription URL
	urlSet      map[string]bool   `json:"-"`                     // 内部使用，用于O(1)去重检查
	TestResults *TestResult       `json:"test_results,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewChannel 创建新的频道
func NewChannel(name, groupTitle, tvgID, tvgName, tvgLogo, url, sourceURL string) *Channel {
	id := generateChannelID(name, tvgID)
	now := time.Now()

	urlSet := make(map[string]bool)
	urlSet[url] = true

	return &Channel{
		ID:         id,
		Name:       name,
		GroupTitle: groupTitle,
		TvgID:      tvgID,
		TvgName:    tvgName,
		TvgLogo:    tvgLogo,
		URLs:       []string{url},
		SourceURLs: map[string]string{url: sourceURL},
		urlSet:     urlSet,
		TestResults: &TestResult{
			Status:   "untested",
			TestedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddURL 添加 URL - O(1) 时间复杂度
func (c *Channel) AddURL(url, sourceURL string) {
	// 初始化urlSet（用于从JSON加载的频道）
	if c.urlSet == nil {
		c.urlSet = make(map[string]bool)
		for _, u := range c.URLs {
			c.urlSet[u] = true
		}
	}

	// 检查 URL 是否已存在 - O(1)
	if c.urlSet[url] {
		return
	}

	c.URLs = append(c.URLs, url)
	c.urlSet[url] = true
	if c.SourceURLs == nil {
		c.SourceURLs = make(map[string]string)
	}
	c.SourceURLs[url] = sourceURL
	c.UpdatedAt = time.Now()
}

// AddBackupURL 添加备用 URL - 用于保守模式
func (c *Channel) AddBackupURL(url, sourceURL string) {
	// 避免重复添加
	for _, existingURL := range c.BackupURLs {
		if existingURL == url {
			return
		}
	}

	c.BackupURLs = append(c.BackupURLs, url)
	if c.SourceURLs == nil {
		c.SourceURLs = make(map[string]string)
	}
	c.SourceURLs[url] = sourceURL
	c.UpdatedAt = time.Now()
}

// UpdateTestResult 更新测试结果
func (c *Channel) UpdateTestResult(status, workingURL string, responseTime int64) {
	if c.TestResults == nil {
		c.TestResults = &TestResult{}
	}

	c.TestResults.Status = status
	c.TestResults.WorkingURL = workingURL
	c.TestResults.ResponseTime = responseTime
	c.TestResults.TestedAt = time.Now()
	c.UpdatedAt = time.Now()
}

// generateChannelID 生成频道 ID
func generateChannelID(name, tvgID string) string {
	if tvgID != "" {
		return tvgID
	}

	// 使用名称生成 ID
	hash := sha256.Sum256([]byte(name))
	return fmt.Sprintf("%x", hash)[:16]
}

// IsOnline 检查频道是否在线
func (c *Channel) IsOnline() bool {
	return c.TestResults != nil && c.TestResults.Status == "online"
}

// IsOffline 检查频道是否离线
func (c *Channel) IsOffline() bool {
	return c.TestResults != nil && c.TestResults.Status == "offline"
}

// IsUntested 检查频道是否未测试
func (c *Channel) IsUntested() bool {
	return c.TestResults == nil || c.TestResults.Status == "untested"
}
