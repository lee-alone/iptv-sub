package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/iptv-aggregator/models"
	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestIntegrationCompleteAggregationWorkflow tests the complete aggregation workflow
func TestIntegrationCompleteAggregationWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create test M3U content
	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	m3uFile := filepath.Join(tmpDir, "test.m3u")
	tests.CreateTestFile(t, tmpDir, "test.m3u", m3uContent)

	// Initialize services
	subscriptionMgr := services.NewSubscriptionManager(tmpDir)
	parser := services.NewM3UParser(10 * time.Second)
	aggregator := services.NewChannelAggregator(tmpDir)
	exporter := services.NewChannelExporter(tmpDir)

	// Step 1: Add subscription
	subscriptionURL := "file://" + m3uFile
	err := subscriptionMgr.AddSubscription(subscriptionURL, "Test Subscription", true)
	tests.AssertNoError(t, err, "Should add subscription")

	// Step 2: Parse M3U
	content, err := parser.FetchM3U(subscriptionURL)
	tests.AssertNoError(t, err, "Should fetch M3U")

	channels, err := parser.ParseM3U(content, subscriptionURL)
	tests.AssertNoError(t, err, "Should parse M3U")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels")

	// Step 3: Aggregate channels
	_, _, _, err = aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 2, len(allChannels), "Should have 2 channels")

	// Step 4: Export to M3U
	m3uPath, err := exporter.ExportM3U(allChannels, false)
	tests.AssertNoError(t, err, "Should export M3U")
	tests.AssertTrue(t, tests.FileExists(m3uPath), "Exported M3U should exist")

	// Step 5: Export to JSON
	jsonPath, err := exporter.ExportJSON(allChannels, false)
	tests.AssertNoError(t, err, "Should export JSON")
	tests.AssertTrue(t, tests.FileExists(jsonPath), "Exported JSON should exist")

	t.Logf("✓ Complete aggregation workflow test passed")
}

// TestIntegrationSubscriptionManagement tests subscription management
func TestIntegrationSubscriptionManagement(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	subscriptionMgr := services.NewSubscriptionManager(tmpDir)

	// Add subscription
	url1 := "http://example.com/playlist1.m3u"
	err := subscriptionMgr.AddSubscription(url1, "Subscription 1", true)
	tests.AssertNoError(t, err, "Should add subscription")

	// Verify subscription was added
	subs := subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have 1 subscription")

	// Add another subscription
	url2 := "http://example.com/playlist2.m3u"
	err = subscriptionMgr.AddSubscription(url2, "Subscription 2", false)
	tests.AssertNoError(t, err, "Should add second subscription")

	subs = subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 2, len(subs), "Should have 2 subscriptions")

	// Remove subscription
	err = subscriptionMgr.RemoveSubscription(url1)
	tests.AssertNoError(t, err, "Should remove subscription")

	subs = subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have 1 subscription after removal")

	t.Logf("✓ Subscription management test passed")
}

// TestIntegrationChannelQuery tests channel query functionality
func TestIntegrationChannelQuery(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	aggregator := services.NewChannelAggregator(tmpDir)

	// Create test channels
	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "Online Channel",
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
			Name:       "Offline Channel",
			URLs:       []string{"http://example.com/stream2.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream2.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "offline",
				ResponseTime: 0,
				TestedAt:     time.Now(),
			},
		},
	}

	// Aggregate channels
	_, _, _, err := aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels")

	// Get all channels
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 2, len(allChannels), "Should have 2 channels")

	// Get channel by ID
	ch := aggregator.GetChannelByID("ch1")
	tests.AssertNotNil(t, ch, "Should retrieve channel by ID")
	tests.AssertEqual(t, "Online Channel", ch.Name, "Channel name should match")

	// Get online channels
	onlineChannels := aggregator.GetOnlineChannels()
	tests.AssertEqual(t, 1, len(onlineChannels), "Should have 1 online channel")

	t.Logf("✓ Channel query test passed")
}

// TestIntegrationExportWorkflow tests export functionality
func TestIntegrationExportWorkflow(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	// Create test channels
	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "Channel 1",
			TvgID:      "ch1",
			GroupTitle: "Group A",
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
			Name:       "Channel 2",
			TvgID:      "ch2",
			GroupTitle: "Group A",
			URLs:       []string{"http://example.com/stream2.m3u8"},
			SourceURLs: map[string]string{"http://example.com/stream2.m3u8": "http://example.com/playlist.m3u"},
			TestResults: &models.TestResult{
				Status:       "offline",
				ResponseTime: 0,
				TestedAt:     time.Now(),
			},
		},
	}

	// Export all channels to M3U
	m3uPath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "Should export M3U")
	tests.AssertTrue(t, tests.FileExists(m3uPath), "M3U file should exist")

	// Export only online channels to M3U
	m3uOnlinePath, err := exporter.ExportM3U(channels, true)
	tests.AssertNoError(t, err, "Should export online M3U")
	tests.AssertTrue(t, tests.FileExists(m3uOnlinePath), "Online M3U file should exist")

	// Export to JSON
	jsonPath, err := exporter.ExportJSON(channels, false)
	tests.AssertNoError(t, err, "Should export JSON")
	tests.AssertTrue(t, tests.FileExists(jsonPath), "JSON file should exist")

	t.Logf("✓ Export workflow test passed")
}
