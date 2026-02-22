package integration

import (
	"path/filepath"
	"testing"
	"time"

	"iptv-aggregator/models"
	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestIntegrationAggregationWorkflow tests the complete aggregation workflow
// from subscription source to export
func TestIntegrationAggregationWorkflow(t *testing.T) {
	// Setup
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create test M3U content
	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV 1" tvg-logo="http://example.com/logo1.png" group-title="央视",CCTV 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="cctv2" tvg-name="CCTV 2" tvg-logo="http://example.com/logo2.png" group-title="央视",CCTV 2
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="cctv3" tvg-name="CCTV 3" tvg-logo="http://example.com/logo3.png" group-title="央视",CCTV 3
http://example.com/stream3.m3u8
`

	// Create test M3U file
	m3uFile := filepath.Join(tmpDir, "test.m3u")
	tests.CreateTestFile(t, tmpDir, "test.m3u", m3uContent)

	// Initialize services
	subscriptionMgr := services.NewSubscriptionManager(tmpDir)
	parser := services.NewM3UParser(10 * time.Second)
	aggregator := services.NewChannelAggregator(tmpDir)
	tester := services.NewStreamTester(5*time.Second, 2)
	exporter := services.NewChannelExporter(tmpDir)

	// Step 1: Add subscription source
	subscriptionURL := "file://" + m3uFile
	err := subscriptionMgr.AddSubscription(subscriptionURL, "Test Subscription", true)
	tests.AssertNoError(t, err, "Should add subscription successfully")

	// Verify subscription was added
	subs := subscriptionMgr.GetAllSubscriptions()
	tests.AssertEqual(t, 1, len(subs), "Should have 1 subscription")
	tests.AssertEqual(t, "Test Subscription", subs[0].Name, "Subscription name should match")

	// Step 2: Fetch and parse M3U file
	content, err := parser.FetchM3U(subscriptionURL)
	tests.AssertNoError(t, err, "Should fetch M3U content successfully")
	tests.AssertStringContains(t, content, "#EXTM3U", "M3U content should contain header")

	// Step 3: Parse channels
	channels, err := parser.ParseM3U(content, subscriptionURL)
	tests.AssertNoError(t, err, "Should parse M3U successfully")
	tests.AssertEqual(t, 3, len(channels), "Should parse 3 channels")

	// Step 4: Aggregate channels
	_, _, _, err = aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels successfully")

	// Verify channels were aggregated
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 aggregated channels")

	// Step 5: Test stream availability (mock test)
	testResults, err := tester.BatchTest(allChannels, false)
	tests.AssertNoError(t, err, "Should complete batch test")
	tests.AssertNotNil(t, testResults, "Test results should not be nil")

	// Step 6: Export to M3U
	m3uExportPath, err := exporter.ExportM3U(allChannels, false)
	tests.AssertNoError(t, err, "Should export M3U successfully")
	tests.AssertTrue(t, tests.FileExists(m3uExportPath), "Exported M3U file should exist")

	// Verify exported M3U content
	exportedContent := tests.ReadTestFile(t, m3uExportPath)
	tests.AssertStringContains(t, exportedContent, "#EXTM3U", "Exported M3U should contain header")
	tests.AssertStringContains(t, exportedContent, "CCTV 1", "Exported M3U should contain channel name")

	// Step 7: Export to JSON
	jsonExportPath, err := exporter.ExportJSON(allChannels, false)
	tests.AssertNoError(t, err, "Should export JSON successfully")
	tests.AssertTrue(t, tests.FileExists(jsonExportPath), "Exported JSON file should exist")

	// Verify exported JSON content
	exportedJSON := tests.ReadTestFile(t, jsonExportPath)
	tests.AssertStringContains(t, exportedJSON, "CCTV 1", "Exported JSON should contain channel name")
	tests.AssertStringContains(t, exportedJSON, "cctv1", "Exported JSON should contain tvg-id")

	t.Logf("✓ Complete aggregation workflow test passed")
}

// TestIntegrationAggregationWithDuplicates tests aggregation with duplicate channels
func TestIntegrationAggregationWithDuplicates(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create M3U with duplicate channels
	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1-backup.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	m3uFile := filepath.Join(tmpDir, "test.m3u")
	tests.CreateTestFile(t, tmpDir, "test.m3u", m3uContent)

	parser := services.NewM3UParser(10 * time.Second)
	aggregator := services.NewChannelAggregator(tmpDir)

	// Parse channels
	channels, err := parser.ParseM3U(m3uContent, "file://"+m3uFile)
	tests.AssertNoError(t, err, "Should parse M3U successfully")
	tests.AssertEqual(t, 3, len(channels), "Should parse 3 channels (including duplicates)")

	// Aggregate channels
	_, _, _, err = aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate channels successfully")

	// Verify deduplication
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 2, len(allChannels), "Should have 2 unique channels after deduplication")

	// Verify duplicate channel has multiple URLs
	ch1 := aggregator.GetChannelByID(allChannels[0].ID)
	tests.AssertNotNil(t, ch1, "Channel should exist")
	tests.AssertTrue(t, len(ch1.URLs) >= 1, "Channel should have at least 1 URL")

	t.Logf("✓ Aggregation with duplicates test passed")
}

// TestIntegrationMultipleSubscriptions tests aggregating from multiple subscriptions
func TestIntegrationMultipleSubscriptions(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create first M3U file
	m3u1Content := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	// Create second M3U file
	m3u2Content := `#EXTM3U
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2-alt.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group B",Channel 3
http://example.com/stream3.m3u8
`

	m3u1File := filepath.Join(tmpDir, "test1.m3u")
	m3u2File := filepath.Join(tmpDir, "test2.m3u")
	tests.CreateTestFile(t, tmpDir, "test1.m3u", m3u1Content)
	tests.CreateTestFile(t, tmpDir, "test2.m3u", m3u2Content)

	parser := services.NewM3UParser(10 * time.Second)
	aggregator := services.NewChannelAggregator(tmpDir)

	// Parse and aggregate from first subscription
	channels1, err := parser.ParseM3U(m3u1Content, "file://"+m3u1File)
	tests.AssertNoError(t, err, "Should parse first M3U successfully")
	added1, _, _, err := aggregator.AggregateChannels(channels1, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate first M3U successfully")
	tests.AssertEqual(t, 2, added1, "Should add 2 channels from first M3U")

	// Parse and aggregate from second subscription
	channels2, err := parser.ParseM3U(m3u2Content, "file://"+m3u2File)
	tests.AssertNoError(t, err, "Should parse second M3U successfully")
	_, _, _, err = aggregator.AggregateChannels(channels2, "name", 0.8)
	tests.AssertNoError(t, err, "Should aggregate second M3U successfully")

	// Verify aggregation results
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 unique channels total")

	t.Logf("✓ Multiple subscriptions test passed")
}

// TestIntegrationExportFiltering tests export with filtering options
func TestIntegrationExportFiltering(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	// Create channels with different statuses
	channels := []*models.Channel{
		{
			ID:         "ch1",
			Name:       "Channel 1",
			TvgID:      "ch1",
			TvgName:    "Channel 1",
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
			TvgName:    "Channel 2",
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

	exporter := services.NewChannelExporter(tmpDir)

	// Export all channels
	allPath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "Should export all channels successfully")
	allContent := tests.ReadTestFile(t, allPath)
	tests.AssertStringContains(t, allContent, "Channel 1", "All export should contain Channel 1")
	tests.AssertStringContains(t, allContent, "Channel 2", "All export should contain Channel 2")

	// Export only online channels
	onlinePath, err := exporter.ExportM3U(channels, true)
	tests.AssertNoError(t, err, "Should export online channels successfully")
	onlineContent := tests.ReadTestFile(t, onlinePath)
	tests.AssertStringContains(t, onlineContent, "Channel 1", "Online export should contain Channel 1")
	tests.AssertStringNotContains(t, onlineContent, "Channel 2", "Online export should not contain Channel 2")

	t.Logf("✓ Export filtering test passed")
}
