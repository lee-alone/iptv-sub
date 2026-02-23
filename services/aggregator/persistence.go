package aggregator

import (
	"encoding/json"
	"os"

	"iptv-aggregator/models"
)

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

	data, err := json.MarshalIndent(channels, "", " ")
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
