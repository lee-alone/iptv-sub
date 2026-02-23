package aggregator

import (
	"time"

	"iptv-aggregator/models"
)

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
