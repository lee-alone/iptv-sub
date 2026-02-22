package integration

import (
	"testing"
	"time"

	"iptv-aggregator/models"
	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestIntegrationChannelQueryWorkflow tests the complete channel query workflow
func TestIntegrationChannelQueryWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	aggregator := services.NewChannelAggregator(tmpDir)

	// Create test channels
	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "CCTV 1",
			TvgID:      "cctv1",
			TvgName:    "CCTV 1",
			GroupTitle: "央视",
			URLs:       []string{"http://example.com/stream1.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream1.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "online",
				ResponseTime: 100,
				TestedAt:     time.Now(),
			},
		},
		{
			ID:         "ch2",
			Name:       "CCTV 2",
			TvgID:      "cctv2",
			TvgName:    "CCTV 2",
			GroupTitle: "央视",
			URLs:       []string{"http://example.com/stream2.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream2.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "offline",
				ResponseTime: 0,
				TestedAt:     time.Now(),
			},
		},
		{
			ID:         "ch3",
			Name:       "CCTV 3",
			TvgID:      "cctv3",
			TvgName:    "CCTV 3",
			GroupTitle: "央视",
			URLs:       []string{"http://example.com/stream3.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream3.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "online",
				ResponseTime: 150,
				TestedAt:     time.Now(),
			},
		},
	}

	// Aggregate channels
	_, _, _, err := aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels successfully")

	// Step 1: Get all channels
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 channels")

	// Step 2: Get channel by ID
	ch1 := aggregator.GetChannelByID("ch1")
	tests.AssertNotNil(t, ch1, "Should retrieve channel by ID")
	tests.AssertEqual(t, "CCTV 1", ch1.Name, "Channel name should match")
	tests.AssertEqual(t, "cctv1", ch1.TvgID, "Channel tvg-id should match")

	// Step 3: Get online channels
	onlineChannels := aggregator.GetOnlineChannels()
	tests.AssertEqual(t, 2, len(onlineChannels), "Should have 2 online channels")

	// Verify all online channels have online status
	for _, ch := range onlineChannels {
		tests.AssertEqual(t, "online", ch.TestResults.Status, "Online channel should have online status")
	}

	t.Logf("✓ Channel query workflow test passed")
}

// TestIntegrationChannelStatusFiltering tests filtering channels by status
func TestIntegrationChannelStatusFiltering(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	aggregator := services.NewChannelAggregator(tmpDir)

	// Create channels with different statuses
	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "Online Channel 1",
			URLs:       []string{"http://example.com/stream1.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream1.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "online",
				ResponseTime: 100,
				TestedAt:     time.Now(),
			},
		},
		{
			ID:         "ch2",
			Name:       "Offline Channel 1",
			URLs:       []string{"http://example.com/stream2.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream2.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "offline",
				ResponseTime: 0,
				TestedAt:     time.Now(),
			},
		},
		{
			ID:         "ch3",
			Name:       "Online Channel 2",
			URLs:       []string{"http://example.com/stream3.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream3.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "online",
				ResponseTime: 150,
				TestedAt:     time.Now(),
			},
		},
	}

	// Aggregate channels
	_, _, _, err := aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels successfully")

	// Get all channels
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 channels")

	// Get online channels
	onlineChannels := aggregator.GetOnlineChannels()
	tests.AssertEqual(t, 2, len(onlineChannels), "Should have 2 online channels")

	// Verify all online channels have online status
	for _, ch := range onlineChannels {
		tests.AssertEqual(t, "online", ch.TestResults.Status, "Online channel should have online status")
	}

	t.Logf("✓ Channel status filtering test passed")
}

// TestIntegrationChannelPersistence tests that channels are persisted to disk
func TestIntegrationChannelPersistence(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create first aggregator and add channels
	agg1 := services.NewChannelAggregator(tmpDir)

	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "Channel 1",
			GroupTitle: "Group A",
			URLs:       []string{"http://example.com/stream1.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream1.m3u8": "http://example.com/playlist.m3u"},
		},
		{
			ID:         "ch2",
			Name:       "Channel 2",
			GroupTitle: "Group A",
			URLs:       []string{"http://example.com/stream2.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream2.m3u8": "http://example.com/playlist.m3u"},
		},
	}

	_, _, _, err := agg1.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels successfully")

	ch1 := agg1.GetAllChannels()
	tests.AssertEqual(t, 2, len(ch1), "Should have 2 channels")

	// Create new aggregator instance (simulating app restart)
	agg2 := services.NewChannelAggregator(tmpDir)
	ch2 := agg2.GetAllChannels()

	// Verify channels were persisted
	tests.AssertEqual(t, 2, len(ch2), "Should have 2 channels after reload")
	tests.AssertEqual(t, "Channel 1", ch2[0].Name, "First channel name should match")
	tests.AssertEqual(t, "Channel 2", ch2[1].Name, "Second channel name should match")

	t.Logf("✓ Channel persistence test passed")
}
