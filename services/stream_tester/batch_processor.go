package stream_tester

import (
	"iptv-aggregator/models"
	"iptv-aggregator/utils"
)

// SyncTestResultsToSkippedChannels 将测试结果同步到被跳过的频道
func SyncTestResultsToSkippedChannels(testedChannels, skippedChannels []*models.Channel, logger *utils.Logger) {
	// 创建 URL -> 测试结果的映射
	urlToResult := make(map[string]*models.TestResult)
	for _, ch := range testedChannels {
		if len(ch.URLs) > 0 && ch.TestResults != nil {
			urlToResult[ch.URLs[0]] = ch.TestResults
		}
	}

	// 同步结果到被跳过的频道
	syncedCount := 0
	for _, ch := range skippedChannels {
		if len(ch.URLs) > 0 {
			if result, exists := urlToResult[ch.URLs[0]]; exists {
				// 深拷贝测试结果，避免指针共享
				ch.TestResults = &models.TestResult{
					Status:       result.Status,
					WorkingURL:   result.WorkingURL,
					ResponseTime: result.ResponseTime,
					TestedAt:     result.TestedAt,
					Details:      result.Details,
				}
				syncedCount++
			}
		}
	}

	if syncedCount > 0 {
		logger.Info("Synced test results to %d skipped channels", syncedCount)
	}
}
