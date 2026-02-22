package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"iptv-aggregator/models"
)

// SubscriptionManager 订阅源管理器
type SubscriptionManager struct {
	subscriptions map[string]*models.Subscription
	mu            sync.RWMutex
	dataDir       string
	filePath      string
}

// NewSubscriptionManager 创建新的订阅源管理器
func NewSubscriptionManager(dataDir string) *SubscriptionManager {
	sm := &SubscriptionManager{
		subscriptions: make(map[string]*models.Subscription),
		dataDir:       dataDir,
		filePath:      filepath.Join(dataDir, "subscriptions.json"),
	}

	// 确保数据目录存在
	os.MkdirAll(dataDir, 0755)

	// 加载订阅源
	sm.Load()

	return sm
}

// AddSubscription 添加订阅源
func (sm *SubscriptionManager) AddSubscription(url, name string, enabled bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.subscriptions[url]; exists {
		return fmt.Errorf("subscription already exists: %s", url)
	}

	sub := models.NewSubscription(url, name, enabled)
	sm.subscriptions[url] = sub

	return sm.save()
}

// GetSubscription 获取订阅源
func (sm *SubscriptionManager) GetSubscription(url string) *models.Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.subscriptions[url]
}

// GetAllSubscriptions 获取所有订阅源
func (sm *SubscriptionManager) GetAllSubscriptions() []*models.Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := make([]*models.Subscription, 0, len(sm.subscriptions))
	for _, sub := range sm.subscriptions {
		subs = append(subs, sub)
	}

	return subs
}

// UpdateSubscription 更新订阅源
func (sm *SubscriptionManager) UpdateSubscription(oldURL, newURL, name string, enabled bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[oldURL]
	if !exists {
		return fmt.Errorf("subscription not found: %s", oldURL)
	}

	// 如果 URL 改变，需要删除旧的并添加新的
	if oldURL != newURL {
		if _, exists := sm.subscriptions[newURL]; exists {
			return fmt.Errorf("subscription already exists: %s", newURL)
		}

		delete(sm.subscriptions, oldURL)
		sub.URL = newURL
	}

	sub.Name = name
	sub.Enabled = enabled
	sm.subscriptions[newURL] = sub

	return sm.save()
}

// RemoveSubscription 删除订阅源
func (sm *SubscriptionManager) RemoveSubscription(url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.subscriptions[url]; !exists {
		return fmt.Errorf("subscription not found: %s", url)
	}

	delete(sm.subscriptions, url)

	return sm.save()
}

// UpdateSubscriptionStatus 更新订阅源状态
func (sm *SubscriptionManager) UpdateSubscriptionStatus(url, status string, channelCount int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[url]
	if !exists {
		return fmt.Errorf("subscription not found: %s", url)
	}

	sub.UpdateStatus(status, channelCount)

	return sm.save()
}

// Load 从文件加载订阅源
func (sm *SubscriptionManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 如果文件不存在，返回空列表
	if _, err := os.Stat(sm.filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read subscriptions file: %w", err)
	}

	var subs []*models.Subscription
	if err := json.Unmarshal(data, &subs); err != nil {
		return fmt.Errorf("failed to parse subscriptions file: %w", err)
	}

	sm.subscriptions = make(map[string]*models.Subscription)
	for _, sub := range subs {
		sm.subscriptions[sub.URL] = sub
	}

	return nil
}

// save 保存订阅源到文件
func (sm *SubscriptionManager) save() error {
	subs := make([]*models.Subscription, 0, len(sm.subscriptions))
	for _, sub := range sm.subscriptions {
		subs = append(subs, sub)
	}

	data, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions: %w", err)
	}

	// 原子写入
	tmpFile := sm.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write subscriptions file: %w", err)
	}

	if err := os.Rename(tmpFile, sm.filePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename subscriptions file: %w", err)
	}

	return nil
}
