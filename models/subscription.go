package models

import "time"

// Subscription 订阅源模型
type Subscription struct {
	URL          string    `json:"url"`
	Name         string    `json:"name"`
	Enabled      bool      `json:"enabled"`
	Status       string    `json:"status"` // active, failed, untested
	ChannelCount int       `json:"channel_count"`
	LastUpdated  time.Time `json:"last_updated"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// NewSubscription 创建新的订阅源
func NewSubscription(url, name string, enabled bool) *Subscription {
	return &Subscription{
		URL:          url,
		Name:         name,
		Enabled:      enabled,
		Status:       "untested",
		ChannelCount: 0,
		LastUpdated:  time.Now(),
	}
}

// UpdateStatus 更新订阅源状态
func (s *Subscription) UpdateStatus(status string, channelCount int) {
	s.Status = status
	s.ChannelCount = channelCount
	s.LastUpdated = time.Now()
	s.ErrorMessage = ""
}

// UpdateError 更新订阅源错误信息
func (s *Subscription) UpdateError(errMsg string) {
	s.Status = "failed"
	s.ErrorMessage = errMsg
	s.LastUpdated = time.Now()
}
