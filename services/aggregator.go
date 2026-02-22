package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"iptv-aggregator/models"
	"iptv-aggregator/utils"
)

// ChannelAggregator 频道聚合器
type ChannelAggregator struct {
	dataDir    string
	filePath   string
	channels   map[string]*models.Channel
	tvgIDIndex map[string]*models.Channel   // tvg-id 索引，用于O(1)查找
	nameIndex  map[string][]*models.Channel // 名称索引，用于快速查找相似频道
	mu         sync.RWMutex
	logger     *utils.Logger
}

// NewChannelAggregator 创建新的频道聚合器
func NewChannelAggregator(dataDir string) *ChannelAggregator {
	ca := &ChannelAggregator{
		dataDir:    dataDir,
		filePath:   filepath.Join(dataDir, "channels.json"),
		channels:   make(map[string]*models.Channel),
		tvgIDIndex: make(map[string]*models.Channel),
		nameIndex:  make(map[string][]*models.Channel),
		logger:     utils.NewLogger(),
	}

	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)

	// 加载频道数据
	ca.Load()

	return ca
}

// AggregateChannels 聚合频道 - 优化版本，使用索引加速查询
func (ca *ChannelAggregator) AggregateChannels(newChannels []*models.Channel, matchBy string, similarityThreshold float64) (int, int, int, error) {
	ca.logger.Debug("Aggregating %d new channels (matchBy: %s, threshold: %.2f)", len(newChannels), matchBy, similarityThreshold)
	if len(newChannels) == 0 {
		return 0, 0, 0, nil
	}

	ca.mu.Lock()
	defer ca.mu.Unlock()

	added := 0
	updated := 0
	skipped := 0

	for _, newCh := range newChannels {
		matched := false
		var existingCh *models.Channel

		// 根据匹配模式查找现有频道
		switch matchBy {
		case "tvg_id":
			// 优化：使用tvg-id索引，O(1)查找
			if newCh.TvgID != "" {
				existingCh = ca.tvgIDIndex[newCh.TvgID]
				if existingCh != nil {
					matched = true
				}
			}
		case "name":
			// 优化：先在名称索引中查找，避免全量遍历
			existingCh = ca.findChannelByNameSimilarity(newCh.Name, similarityThreshold)
			if existingCh != nil {
				matched = true
			}
		case "both":
			// 优先使用tvg-id精确匹配
			if newCh.TvgID != "" {
				existingCh = ca.tvgIDIndex[newCh.TvgID]
				if existingCh != nil {
					matched = true
				}
			}
			// 如果tvg-id未匹配，尝试名称相似度匹配
			if !matched {
				existingCh = ca.findChannelByNameSimilarity(newCh.Name, similarityThreshold)
				if existingCh != nil {
					matched = true
				}
			}
		}

		if matched && existingCh != nil {
			// 合并频道
			ca.mergeChannels(existingCh, newCh)
			updated++
		} else {
			// 添加新频道
			ca.channels[newCh.ID] = newCh
			ca.addToIndexes(newCh)
			added++
		}
	}

	if added > 0 || updated > 0 {
		ca.save()
	}

	return added, updated, skipped, nil
}

// addToIndexes 将频道添加到索引中
func (ca *ChannelAggregator) addToIndexes(ch *models.Channel) {
	// 添加到tvg-id索引
	if ch.TvgID != "" {
		ca.tvgIDIndex[ch.TvgID] = ch
	}

	// 添加到名称索引
	if ch.Name != "" {
		ca.nameIndex[ch.Name] = append(ca.nameIndex[ch.Name], ch)
	}
}

// removeFromIndexes 从索引中移除频道
func (ca *ChannelAggregator) removeFromIndexes(ch *models.Channel) {
	// 从tvg-id索引移除
	if ch.TvgID != "" {
		delete(ca.tvgIDIndex, ch.TvgID)
	}

	// 从名称索引移除
	if ch.Name != "" {
		channels := ca.nameIndex[ch.Name]
		for i, c := range channels {
			if c.ID == ch.ID {
				ca.nameIndex[ch.Name] = append(channels[:i], channels[i+1:]...)
				if len(ca.nameIndex[ch.Name]) == 0 {
					delete(ca.nameIndex, ch.Name)
				}
				break
			}
		}
	}
}

// findChannelByNameSimilarity 根据名称相似度查找频道
// 多层优化策略：精确匹配 → 不区分大小写 → 快速过滤 → 相似度计算
func (ca *ChannelAggregator) findChannelByNameSimilarity(name string, threshold float64) *models.Channel {
	// 第一步：精确匹配 - O(1)
	if channels, exists := ca.nameIndex[name]; exists && len(channels) > 0 {
		return channels[0]
	}

	// 第二步：不区分大小写匹配 - O(n)，但通常很快就能找到
	lowerName := strings.ToLower(name)
	for key, channels := range ca.nameIndex {
		if strings.ToLower(key) == lowerName && len(channels) > 0 {
			return channels[0]
		}
	}

	// 第三步：相似度匹配 - 仅对候选频道计算
	// 使用快速过滤减少计算量
	var bestMatch *models.Channel
	var bestSimilarity float64
	nameLen := len(name)

	for _, ch := range ca.channels {
		chNameLen := len(ch.Name)

		// 快速过滤1：长度差异过大则跳过（避免不必要的Levenshtein计算）
		// 如果长度差异超过50%，相似度不可能达到threshold
		if chNameLen < nameLen/2 || chNameLen > nameLen*2 {
			continue
		}

		// 快速过滤2：首字母不同且threshold很高则跳过
		if threshold > 0.8 && len(name) > 0 && len(ch.Name) > 0 {
			if !strings.EqualFold(name[:1], ch.Name[:1]) {
				continue
			}
		}

		// 计算相似度
		similarity := ca.stringSimilarity(ch.Name, name)
		if similarity >= threshold && similarity > bestSimilarity {
			bestMatch = ch
			bestSimilarity = similarity
		}
	}

	return bestMatch
}

// copyTestResult 深拷贝测试结果，避免指针共享
func copyTestResult(tr *models.TestResult) *models.TestResult {
	if tr == nil {
		return nil
	}
	return &models.TestResult{
		Status:       tr.Status,
		WorkingURL:   tr.WorkingURL,
		ResponseTime: tr.ResponseTime,
		TestedAt:     tr.TestedAt,
		Details:      tr.Details,
	}
}

// mergeChannels 合并两个频道 - 改进版，保留最佳测试结果
func (ca *ChannelAggregator) mergeChannels(existing, new *models.Channel) {
	// 记录聚合前的状态
	existingWasWorking := existing.IsOnline()
	newIsWorking := new.IsOnline()

	// 添加新的 URL
	for _, url := range new.URLs {
		existing.AddURL(url, new.SourceURLs[url])
	}

	// 更新元数据（如果新的更完整）
	if new.TvgName != "" && existing.TvgName == "" {
		existing.TvgName = new.TvgName
	}
	if new.TvgLogo != "" && existing.TvgLogo == "" {
		existing.TvgLogo = new.TvgLogo
	}
	if new.GroupTitle != "" && existing.GroupTitle == "" {
		existing.GroupTitle = new.GroupTitle
	}

	// 智能合并测试结果：保留最佳的工作状态
	if new.TestResults != nil {
		if existing.TestResults == nil {
			// 深拷贝，避免指针共享
			existing.TestResults = copyTestResult(new.TestResults)
		} else if newIsWorking && !existingWasWorking {
			// 新频道可用，现有频道不可用 → 更新为新频道的结果（深拷贝）
			existing.TestResults = copyTestResult(new.TestResults)
		} else if existingWasWorking && newIsWorking {
			// 两者都可用 → 保留响应时间更快的
			if existing.TestResults.ResponseTime == 0 && new.TestResults.ResponseTime > 0 {
				// 现有的是0ms，新的是真实值 → 使用新的
				existing.TestResults = copyTestResult(new.TestResults)
			} else if existing.TestResults.ResponseTime > 0 && new.TestResults.ResponseTime > 0 {
				// 两者都是真实值 → 选择更短的
				if new.TestResults.ResponseTime < existing.TestResults.ResponseTime {
					existing.TestResults = copyTestResult(new.TestResults)
				}
			}
			// 其他情况保留现有的测试结果
		}
		// 其他情况保留现有的测试结果
	}

	existing.UpdatedAt = time.Now()
}

// stringSimilarity 计算字符串相似度（Levenshtein 距离）
func (ca *ChannelAggregator) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	len1 := len(s1)
	len2 := len(s2)

	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// 使用 Levenshtein 距离
	d := make([][]int, len1+1)
	for i := range d {
		d[i] = make([]int, len2+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			d[i][j] = min(
				d[i-1][j]+1,
				min(d[i][j-1]+1, d[i-1][j-1]+cost),
			)
		}
	}

	distance := d[len1][len2]
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// GetAllChannels 获取所有频道
func (ca *ChannelAggregator) GetAllChannels() []*models.Channel {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	channels := make([]*models.Channel, 0, len(ca.channels))
	for _, ch := range ca.channels {
		channels = append(channels, ch)
	}

	return channels
}

// GetChannelCountBySource 获取指定来源的频道数量
func (ca *ChannelAggregator) GetChannelCountBySource(sourceURL string) int {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	count := 0
	for _, ch := range ca.channels {
		for _, subURL := range ch.SourceURLs {
			if subURL == sourceURL {
				count++
				break
			}
		}
	}
	return count
}

// GetChannelByID 根据 ID 获取频道
func (ca *ChannelAggregator) GetChannelByID(id string) *models.Channel {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	return ca.channels[id]
}

// GetOnlineChannels 获取在线频道
func (ca *ChannelAggregator) GetOnlineChannels() []*models.Channel {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var online []*models.Channel
	for _, ch := range ca.channels {
		if ch.IsOnline() {
			online = append(online, ch)
		}
	}

	return online
}

// GetOfflineChannels 获取离线频道
func (ca *ChannelAggregator) GetOfflineChannels() []*models.Channel {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var offline []*models.Channel
	for _, ch := range ca.channels {
		if ch.IsOffline() {
			offline = append(offline, ch)
		}
	}

	return offline
}

// GetUntestedChannels 获取未测试频道
func (ca *ChannelAggregator) GetUntestedChannels() []*models.Channel {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var untested []*models.Channel
	for _, ch := range ca.channels {
		if ch.IsUntested() {
			untested = append(untested, ch)
		}
	}

	return untested
}

// HasTestResults 检查是否有有效的测试结果
func (ca *ChannelAggregator) HasTestResults() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	for _, ch := range ca.channels {
		if ch.TestResults != nil && ch.TestResults.Status != "untested" {
			return true
		}
	}

	return false
}

// GetLastTestTime 获取最后测试时间
func (ca *ChannelAggregator) GetLastTestTime() time.Time {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	var lastTime time.Time

	for _, ch := range ca.channels {
		if ch.TestResults != nil && !ch.TestResults.TestedAt.IsZero() {
			if ch.TestResults.TestedAt.After(lastTime) {
				lastTime = ch.TestResults.TestedAt
			}
		}
	}

	return lastTime
}

// ClearChannels 清空所有频道
func (ca *ChannelAggregator) ClearChannels() {
	ca.mu.Lock()
	ca.channels = make(map[string]*models.Channel)
	ca.tvgIDIndex = make(map[string]*models.Channel)
	ca.nameIndex = make(map[string][]*models.Channel)
	ca.mu.Unlock()

	ca.save()
}

// RemoveChannel 删除单个频道
func (ca *ChannelAggregator) RemoveChannel(id string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ch, exists := ca.channels[id]
	if !exists {
		return fmt.Errorf("channel not found: %s", id)
	}

	delete(ca.channels, id)
	ca.removeFromIndexes(ch)
	return ca.save()
}

// ResetTestResults 重置所有频道的测试结果
func (ca *ChannelAggregator) ResetTestResults() {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	now := time.Now()
	for _, ch := range ca.channels {
		if ch.TestResults == nil {
			ch.TestResults = &models.TestResult{}
		}
		ch.TestResults.Status = "untested"
		ch.TestResults.WorkingURL = ""
		ch.TestResults.ResponseTime = 0
		ch.TestResults.TestedAt = now
		ch.TestResults.Details = "Reset before test"
	}
	ca.save()
}

// RemoveOldChannels 删除长时间离线或未更新的频道
func (ca *ChannelAggregator) RemoveOldChannels(maxAge time.Duration) int {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, ch := range ca.channels {
		// 检查频道是否离线超过 maxAge
		if ch.TestResults != nil && ch.TestResults.Status == "offline" {
			if now.Sub(ch.TestResults.TestedAt) > maxAge {
				delete(ca.channels, id)
				ca.removeFromIndexes(ch)
				removed++
			}
		}
	}

	if removed > 0 {
		ca.save()
	}

	return removed
}

// Load 从文件加载频道
func (ca *ChannelAggregator) Load() error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if _, err := os.Stat(ca.filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(ca.filePath)
	if err != nil {
		return err
	}

	var channelList []*models.Channel
	if err := json.Unmarshal(data, &channelList); err != nil {
		return err
	}

	ca.channels = make(map[string]*models.Channel)
	ca.tvgIDIndex = make(map[string]*models.Channel)
	ca.nameIndex = make(map[string][]*models.Channel)

	for _, ch := range channelList {
		ca.channels[ch.ID] = ch
		ca.addToIndexes(ch)
	}

	return nil
}

// save 保存频道到文件 (内部使用，不加锁)
func (ca *ChannelAggregator) save() error {
	channels := make([]*models.Channel, 0, len(ca.channels))
	for _, ch := range ca.channels {
		channels = append(channels, ch)
	}

	data, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		return err
	}

	// 原子写入
	tmpFile := ca.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, ca.filePath); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

// Save 公开的保存方法
func (ca *ChannelAggregator) Save() error {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.save()
}

// min 返回两个整数中的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
