package aggregator

import (
	"fmt"
	"time"

	"iptv-aggregator/models"
)

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
