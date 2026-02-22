package services

import (
	"iptv-aggregator/models"
)

// TestResult 频道测试结果
type TestResult struct {
	Channel      *models.Channel
	WorkingURL   string
	ResponseTime int64
	IsOnline     bool
	Error        error
}

// ChannelTester 处理频道测试的具体实现
type ChannelTester struct {
	streamTester *StreamTester
}

// NewChannelTester 创建新的频道测试器
func NewChannelTester(streamTester *StreamTester) *ChannelTester {
	return &ChannelTester{
		streamTester: streamTester,
	}
}

// TestChannel 测试单个频道，返回测试结果
func (ct *ChannelTester) TestChannel(channel *models.Channel, testAllSources bool) *TestResult {
	result := &TestResult{
		Channel: channel,
	}

	if testAllSources {
		// 测试所有 URL
		for _, url := range channel.URLs {
			ct.streamTester.hostCooldown.Wait(url)
			release := ct.streamTester.domainLimiter.Acquire(url)

			online, rt, err := ct.streamTester.TestStream(url)
			ct.streamTester.hostCooldown.Record(url)

			if err == nil && online {
				ct.streamTester.adaptiveLimiter.RecordSuccess()
				result.WorkingURL = url
				result.ResponseTime = rt
				result.IsOnline = true
				release()
				break
			} else if err != nil {
				if isRateLimitError(err) {
					ct.streamTester.adaptiveLimiter.RecordError()
					ct.streamTester.logger.Warn("Rate limit detected for URL: %s, error: %v", url, err)
				}
				result.Error = err
			}

			release()
		}
	} else {
		// 只测试第一个 URL
		if len(channel.URLs) > 0 {
			url := channel.URLs[0]
			ct.streamTester.hostCooldown.Wait(url)
			release := ct.streamTester.domainLimiter.Acquire(url)

			online, rt, err := ct.streamTester.TestStream(url)
			ct.streamTester.hostCooldown.Record(url)

			if err == nil && online {
				ct.streamTester.adaptiveLimiter.RecordSuccess()
				result.WorkingURL = url
				result.ResponseTime = rt
				result.IsOnline = true
			} else {
				if isRateLimitError(err) {
					ct.streamTester.adaptiveLimiter.RecordError()
					ct.streamTester.logger.Warn("Rate limit detected for URL: %s, error: %v", url, err)
				}
				result.Error = err
			}

			release()
		}
	}

	// 更新频道的测试结果
	status := "offline"
	if result.IsOnline {
		status = "online"
	}

	channel.UpdateTestResult(status, result.WorkingURL, result.ResponseTime)
	if result.Error != nil && !result.IsOnline {
		channel.TestResults.Details = result.Error.Error()
	}

	return result
}

// DeduplicateChannels 去重频道 - 避免重复测试相同的 URL
func (ct *ChannelTester) DeduplicateChannels(channels []*models.Channel) []*models.Channel {
	deduped := make(map[string]bool)
	uniqueChannels := make([]*models.Channel, 0, len(channels))

	for _, ch := range channels {
		if len(ch.URLs) > 0 && !deduped[ch.URLs[0]] {
			deduped[ch.URLs[0]] = true
			uniqueChannels = append(uniqueChannels, ch)
		}
	}

	if len(uniqueChannels) < len(channels) {
		ct.streamTester.logger.Info("Deduplicated channels: %d -> %d", len(channels), len(uniqueChannels))
	}

	return uniqueChannels
}
