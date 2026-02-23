package aggregator

import (
	"strings"

	"iptv-aggregator/models"
)

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
		similarity := StringSimilarity(ch.Name, name)
		if similarity >= threshold && similarity > bestSimilarity {
			bestMatch = ch
			bestSimilarity = similarity
		}
	}

	return bestMatch
}
