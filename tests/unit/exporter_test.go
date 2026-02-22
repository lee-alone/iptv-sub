package unit

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"iptv-aggregator/models"
	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestExportM3U_ValidFormat tests exporting channels to valid M3U format
func TestExportM3U_ValidFormat(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:         "1",
			Name:       "Channel 1",
			TvgID:      "tvg1",
			TvgName:    "TVG Channel 1",
			TvgLogo:    "http://logo1.png",
			GroupTitle: "Group 1",
			URLs:       []string{"http://url1.m3u8"},
		},
	}

	filePath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "ExportM3U should not return error")
	tests.AssertTrue(t, tests.FileExists(filePath), "Exported file should exist")

	content := tests.ReadTestFile(t, filePath)
	tests.AssertStringContains(t, content, "#EXTM3U", "M3U file should start with #EXTM3U header")
	tests.AssertStringContains(t, content, "#EXTINF", "M3U file should contain EXTINF line")
	tests.AssertStringContains(t, content, "Channel 1", "M3U file should contain channel name")
}

// TestExportM3U_OnlineChannelFilter tests filtering to export only online channels
func TestExportM3U_OnlineChannelFilter(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	onlineChannel := &models.Channel{
		ID:   "1",
		Name: "Online Channel",
		URLs: []string{"http://url1.m3u8"},
		TestResults: &models.TestResult{
			Status:     "online",
			WorkingURL: "http://url1.m3u8",
		},
	}

	offlineChannel := &models.Channel{
		ID:   "2",
		Name: "Offline Channel",
		URLs: []string{"http://url2.m3u8"},
		TestResults: &models.TestResult{
			Status: "offline",
		},
	}

	channels := []*models.Channel{onlineChannel, offlineChannel}

	filePath, err := exporter.ExportM3U(channels, true)
	tests.AssertNoError(t, err, "ExportM3U with onlyWorking=true should not return error")

	content := tests.ReadTestFile(t, filePath)
	tests.AssertStringContains(t, content, "Online Channel", "M3U should contain online channel")
	tests.AssertFalse(t, strings.Contains(content, "Offline Channel"), "M3U should not contain offline channel")
}

// TestExportM3U_FileGeneration tests that M3U file is properly generated
func TestExportM3U_FileGeneration(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:   "1",
			Name: "Test Channel",
			URLs: []string{"http://test.m3u8"},
		},
	}

	filePath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	// Check file exists
	tests.AssertTrue(t, tests.FileExists(filePath), "File should exist")

	// Check file is in exports directory
	tests.AssertTrue(t, strings.Contains(filePath, "exports"), "File should be in exports directory")

	// Check file has .m3u extension
	tests.AssertTrue(t, strings.HasSuffix(filePath, ".m3u"), "File should have .m3u extension")
}

// TestExportM3U_MultipleChannels tests exporting multiple channels
func TestExportM3U_MultipleChannels(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := make([]*models.Channel, 10)
	for i := 0; i < 10; i++ {
		channels[i] = &models.Channel{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("Channel %d", i),
			URLs: []string{fmt.Sprintf("http://url%d.m3u8", i)},
		}
	}

	filePath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	content := tests.ReadTestFile(t, filePath)
	for i := 0; i < 10; i++ {
		channelName := fmt.Sprintf("Channel %d", i)
		tests.AssertStringContains(t, content, channelName, fmt.Sprintf("M3U should contain %s", channelName))
	}
}

// TestExportM3U_EmptyChannelList tests exporting empty channel list
func TestExportM3U_EmptyChannelList(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	filePath, err := exporter.ExportM3U([]*models.Channel{}, false)
	tests.AssertNoError(t, err, "ExportM3U with empty list should succeed")

	content := tests.ReadTestFile(t, filePath)
	tests.AssertStringContains(t, content, "#EXTM3U", "M3U file should still have header")
}

// TestExportM3U_ChannelAttributes tests that channel attributes are properly exported
func TestExportM3U_ChannelAttributes(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channel := &models.Channel{
		ID:         "test-id",
		Name:       "Test Channel",
		TvgID:      "tvg-123",
		TvgName:    "TVG Name",
		TvgLogo:    "http://logo.png",
		GroupTitle: "Test Group",
		URLs:       []string{"http://test.m3u8"},
	}

	filePath, err := exporter.ExportM3U([]*models.Channel{channel}, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	content := tests.ReadTestFile(t, filePath)
	tests.AssertStringContains(t, content, "tvg-id=\"tvg-123\"", "M3U should contain tvg-id")
	tests.AssertStringContains(t, content, "tvg-name=\"TVG Name\"", "M3U should contain tvg-name")
	tests.AssertStringContains(t, content, "tvg-logo=\"http://logo.png\"", "M3U should contain tvg-logo")
	tests.AssertStringContains(t, content, "group-title=\"Test Group\"", "M3U should contain group-title")
}

// TestExportJSON_ValidFormat tests exporting channels to valid JSON format
func TestExportJSON_ValidFormat(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:   "1",
			Name: "Channel 1",
			URLs: []string{"http://url1.m3u8"},
		},
	}

	filePath, err := exporter.ExportJSON(channels, false)
	tests.AssertNoError(t, err, "ExportJSON should not return error")
	tests.AssertTrue(t, tests.FileExists(filePath), "Exported file should exist")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	err = json.Unmarshal([]byte(content), &data)
	tests.AssertNoError(t, err, "JSON should be valid")
}

// TestExportJSON_DataCompleteness tests that exported JSON contains all channel data
func TestExportJSON_DataCompleteness(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channel := &models.Channel{
		ID:         "test-id",
		Name:       "Test Channel",
		TvgID:      "tvg-123",
		TvgName:    "TVG Name",
		TvgLogo:    "http://logo.png",
		GroupTitle: "Test Group",
		URLs:       []string{"http://url1.m3u8", "http://url2.m3u8"},
	}

	filePath, err := exporter.ExportJSON([]*models.Channel{channel}, false)
	tests.AssertNoError(t, err, "ExportJSON should succeed")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	err = json.Unmarshal([]byte(content), &data)
	tests.AssertNoError(t, err, "JSON should be valid")

	tests.AssertEqual(t, 1, len(data), "JSON should contain one channel")
	tests.AssertEqual(t, "test-id", data[0]["id"], "Channel ID should match")
	tests.AssertEqual(t, "Test Channel", data[0]["name"], "Channel name should match")
}

// TestExportJSON_OnlineChannelFilter tests filtering to export only online channels in JSON
func TestExportJSON_OnlineChannelFilter(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	onlineChannel := &models.Channel{
		ID:   "1",
		Name: "Online Channel",
		URLs: []string{"http://url1.m3u8"},
		TestResults: &models.TestResult{
			Status: "online",
		},
	}

	offlineChannel := &models.Channel{
		ID:   "2",
		Name: "Offline Channel",
		URLs: []string{"http://url2.m3u8"},
		TestResults: &models.TestResult{
			Status: "offline",
		},
	}

	channels := []*models.Channel{onlineChannel, offlineChannel}

	filePath, err := exporter.ExportJSON(channels, true)
	tests.AssertNoError(t, err, "ExportJSON with onlyWorking=true should succeed")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	json.Unmarshal([]byte(content), &data)

	tests.AssertEqual(t, 1, len(data), "JSON should contain only online channel")
	tests.AssertEqual(t, "Online Channel", data[0]["name"], "JSON should contain online channel")
}

// TestExportJSON_FileGeneration tests that JSON file is properly generated
func TestExportJSON_FileGeneration(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:   "1",
			Name: "Test Channel",
			URLs: []string{"http://test.m3u8"},
		},
	}

	filePath, err := exporter.ExportJSON(channels, false)
	tests.AssertNoError(t, err, "ExportJSON should succeed")

	// Check file exists
	tests.AssertTrue(t, tests.FileExists(filePath), "File should exist")

	// Check file is in exports directory
	tests.AssertTrue(t, strings.Contains(filePath, "exports"), "File should be in exports directory")

	// Check file has .json extension
	tests.AssertTrue(t, strings.HasSuffix(filePath, ".json"), "File should have .json extension")
}

// TestExportJSON_MultipleChannels tests exporting multiple channels to JSON
func TestExportJSON_MultipleChannels(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := make([]*models.Channel, 5)
	for i := 0; i < 5; i++ {
		channels[i] = &models.Channel{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("Channel %d", i),
			URLs: []string{fmt.Sprintf("http://url%d.m3u8", i)},
		}
	}

	filePath, err := exporter.ExportJSON(channels, false)
	tests.AssertNoError(t, err, "ExportJSON should succeed")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	json.Unmarshal([]byte(content), &data)

	tests.AssertEqual(t, 5, len(data), "JSON should contain all channels")
}

// TestExportJSON_EmptyChannelList tests exporting empty channel list to JSON
func TestExportJSON_EmptyChannelList(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	filePath, err := exporter.ExportJSON([]*models.Channel{}, false)
	tests.AssertNoError(t, err, "ExportJSON with empty list should succeed")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	err = json.Unmarshal([]byte(content), &data)
	tests.AssertNoError(t, err, "JSON should be valid")
	tests.AssertEqual(t, 0, len(data), "JSON should be empty array")
}

// TestExport_TimestampInFilename tests that exported files have timestamp in filename
func TestExport_TimestampInFilename(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:   "1",
			Name: "Test Channel",
			URLs: []string{"http://test.m3u8"},
		},
	}

	filePath, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	filename := filepath.Base(filePath)
	// Check filename contains timestamp pattern (YYYYMMDD_HHMMSS)
	tests.AssertTrue(t, strings.Contains(filename, "iptv_export_"), "Filename should contain prefix")
	tests.AssertTrue(t, strings.HasSuffix(filename, ".m3u"), "Filename should have .m3u extension")
}

// TestExport_ExportsDirectoryCreation tests that exports directory is created if not exists
func TestExport_ExportsDirectoryCreation(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channels := []*models.Channel{
		{
			ID:   "1",
			Name: "Test Channel",
			URLs: []string{"http://test.m3u8"},
		},
	}

	_, err := exporter.ExportM3U(channels, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	exportsDir := filepath.Join(tmpDir, "exports")
	tests.AssertTrue(t, tests.DirExists(exportsDir), "Exports directory should be created")
}

// TestExportM3U_ChannelWithMultipleURLs tests exporting channel with multiple URLs
func TestExportM3U_ChannelWithMultipleURLs(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channel := &models.Channel{
		ID:   "1",
		Name: "Multi-URL Channel",
		URLs: []string{"http://url1.m3u8", "http://url2.m3u8", "http://url3.m3u8"},
		TestResults: &models.TestResult{
			WorkingURL: "http://url2.m3u8",
		},
	}

	filePath, err := exporter.ExportM3U([]*models.Channel{channel}, false)
	tests.AssertNoError(t, err, "ExportM3U should succeed")

	content := tests.ReadTestFile(t, filePath)
	// Should use the working URL
	tests.AssertStringContains(t, content, "http://url2.m3u8", "M3U should contain working URL")
}

// TestExportJSON_ChannelWithMultipleURLs tests exporting channel with multiple URLs to JSON
func TestExportJSON_ChannelWithMultipleURLs(t *testing.T) {
	tmpDir := tests.CreateTempDir(t)
	defer tests.CleanupTempDir(t, tmpDir)

	exporter := services.NewChannelExporter(tmpDir)

	channel := &models.Channel{
		ID:   "1",
		Name: "Multi-URL Channel",
		URLs: []string{"http://url1.m3u8", "http://url2.m3u8"},
	}

	filePath, err := exporter.ExportJSON([]*models.Channel{channel}, false)
	tests.AssertNoError(t, err, "ExportJSON should succeed")

	content := tests.ReadTestFile(t, filePath)
	var data []map[string]interface{}
	json.Unmarshal([]byte(content), &data)

	tests.AssertEqual(t, 1, len(data), "JSON should contain one channel")
	urls := data[0]["urls"].([]interface{})
	tests.AssertEqual(t, 2, len(urls), "Channel should have 2 URLs")
}
