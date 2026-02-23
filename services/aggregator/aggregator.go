package aggregator

import (
	"os"
	"path/filepath"
	"sync"

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
