package unit

import (
	"fmt"
	"testing"
	"time"

	"iptv-aggregator/services"
	"iptv-aggregator/tests"
)

// TestParseM3U_StandardFormat tests parsing of standard M3U format
func TestParseM3U_StandardFormat(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV 1" tvg-logo="http://example.com/logo1.png" group-title="央视",CCTV 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="cctv2" tvg-name="CCTV 2" tvg-logo="http://example.com/logo2.png" group-title="央视",CCTV 2
http://example.com/stream2.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels")
}

// TestParseM3U_ChannelAttributeExtraction tests extraction of channel attributes
func TestParseM3U_ChannelAttributeExtraction(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="test-id-123" tvg-name="Test Channel" tvg-logo="http://example.com/test.png" group-title="Test Group",Test Channel Name
http://example.com/test.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 1, len(channels), "Should parse 1 channel")

	channel := channels[0]
	tests.AssertEqual(t, "test-id-123", channel.TvgID, "TvgID should be extracted correctly")
	tests.AssertEqual(t, "Test Channel", channel.TvgName, "TvgName should be extracted correctly")
	tests.AssertEqual(t, "http://example.com/test.png", channel.TvgLogo, "TvgLogo should be extracted correctly")
	tests.AssertEqual(t, "Test Group", channel.GroupTitle, "GroupTitle should be extracted correctly")
	tests.AssertEqual(t, "Test Channel Name", channel.Name, "Channel name should be extracted correctly")
}

// TestParseM3U_MultipleChannels tests parsing of multiple channels
func TestParseM3U_MultipleChannels(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group B",Channel 3
http://example.com/stream3.m3u8
#EXTINF:-1 tvg-id="ch4" tvg-name="Channel 4" tvg-logo="http://example.com/logo4.png" group-title="Group B",Channel 4
http://example.com/stream4.m3u8
#EXTINF:-1 tvg-id="ch5" tvg-name="Channel 5" tvg-logo="http://example.com/logo5.png" group-title="Group C",Channel 5
http://example.com/stream5.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 5, len(channels), "Should parse 5 channels")

	// Verify each channel
	expectedChannels := []struct {
		tvgID      string
		tvgName    string
		groupTitle string
		name       string
	}{
		{"ch1", "Channel 1", "Group A", "Channel 1"},
		{"ch2", "Channel 2", "Group A", "Channel 2"},
		{"ch3", "Channel 3", "Group B", "Channel 3"},
		{"ch4", "Channel 4", "Group B", "Channel 4"},
		{"ch5", "Channel 5", "Group C", "Channel 5"},
	}

	for i, expected := range expectedChannels {
		tests.AssertEqual(t, expected.tvgID, channels[i].TvgID, fmt.Sprintf("Channel %d TvgID mismatch", i))
		tests.AssertEqual(t, expected.tvgName, channels[i].TvgName, fmt.Sprintf("Channel %d TvgName mismatch", i))
		tests.AssertEqual(t, expected.groupTitle, channels[i].GroupTitle, fmt.Sprintf("Channel %d GroupTitle mismatch", i))
		tests.AssertEqual(t, expected.name, channels[i].Name, fmt.Sprintf("Channel %d Name mismatch", i))
	}
}

// TestParseM3U_ChannelURLs tests that channel URLs are correctly assigned
func TestParseM3U_ChannelURLs(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	sourceURL := "http://example.com/playlist.m3u"
	channels, err := parser.ParseM3U(m3uContent, sourceURL)
	tests.AssertNoError(t, err, "ParseM3U should not return error")

	// Verify URLs
	tests.AssertEqual(t, 1, len(channels[0].URLs), "Channel 1 should have 1 URL")
	tests.AssertEqual(t, "http://example.com/stream1.m3u8", channels[0].URLs[0], "Channel 1 URL should match")
	tests.AssertEqual(t, sourceURL, channels[0].SourceURLs["http://example.com/stream1.m3u8"], "Channel 1 source URL should match")

	tests.AssertEqual(t, 1, len(channels[1].URLs), "Channel 2 should have 1 URL")
	tests.AssertEqual(t, "http://example.com/stream2.m3u8", channels[1].URLs[0], "Channel 2 URL should match")
	tests.AssertEqual(t, sourceURL, channels[1].SourceURLs["http://example.com/stream2.m3u8"], "Channel 2 source URL should match")
}

// TestParseM3U_EmptyContent tests parsing of empty M3U content
func TestParseM3U_EmptyContent(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for empty content")
	tests.AssertEqual(t, 0, len(channels), "Should parse 0 channels from empty content")
}

// TestParseM3U_MissingAttributes tests parsing with missing optional attributes
func TestParseM3U_MissingAttributes(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1,Channel Without Attributes
http://example.com/stream.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2",Channel 2
http://example.com/stream2.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels")

	// First channel should have empty attributes
	tests.AssertEqual(t, "", channels[0].TvgID, "Channel 1 TvgID should be empty")
	tests.AssertEqual(t, "", channels[0].TvgName, "Channel 1 TvgName should be empty")
	tests.AssertEqual(t, "", channels[0].TvgLogo, "Channel 1 TvgLogo should be empty")
	tests.AssertEqual(t, "", channels[0].GroupTitle, "Channel 1 GroupTitle should be empty")

	// Second channel should have partial attributes
	tests.AssertEqual(t, "ch2", channels[1].TvgID, "Channel 2 TvgID should be extracted")
	tests.AssertEqual(t, "Channel 2", channels[1].TvgName, "Channel 2 TvgName should be extracted")
	tests.AssertEqual(t, "", channels[1].TvgLogo, "Channel 2 TvgLogo should be empty")
}

// TestParseM3U_SkipsComments tests that comments are properly skipped
func TestParseM3U_SkipsComments(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
# This is a comment
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
# Another comment
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels, ignoring comments")
}

// TestParseM3U_HandlesWhitespace tests that whitespace is properly handled
func TestParseM3U_HandlesWhitespace(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U

#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
  http://example.com/stream1.m3u8  

#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8

`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels")
	tests.AssertEqual(t, "http://example.com/stream1.m3u8", channels[0].URLs[0], "URL should be trimmed")
}

// TestParseM3U_ChannelCreation tests that channels are properly created with correct model
func TestParseM3U_ChannelCreation(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="test-id" tvg-name="Test Channel" tvg-logo="http://example.com/logo.png" group-title="Test Group",Test Channel
http://example.com/stream.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 1, len(channels), "Should parse 1 channel")

	channel := channels[0]

	// Verify channel is a proper Channel model
	tests.AssertNotNil(t, channel, "Channel should not be nil")
	tests.AssertNotEqual(t, "", channel.ID, "Channel ID should not be empty")
	tests.AssertEqual(t, "Test Channel", channel.Name, "Channel name should match")
	tests.AssertEqual(t, "Test Group", channel.GroupTitle, "Channel group should match")
	tests.AssertEqual(t, "test-id", channel.TvgID, "Channel tvg-id should match")
	tests.AssertEqual(t, "Test Channel", channel.TvgName, "Channel tvg-name should match")
	tests.AssertEqual(t, "http://example.com/logo.png", channel.TvgLogo, "Channel logo should match")

	// Verify timestamps are set
	tests.AssertNotNil(t, channel.CreatedAt, "CreatedAt should be set")
	tests.AssertNotNil(t, channel.UpdatedAt, "UpdatedAt should be set")

	// Verify test results are initialized
	tests.AssertNotNil(t, channel.TestResults, "TestResults should be initialized")
	tests.AssertEqual(t, "untested", channel.TestResults.Status, "Initial status should be untested")
}

// TestParseM3U_SpecialCharactersInNames tests parsing with special characters
func TestParseM3U_SpecialCharactersInNames(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="频道 & 测试" tvg-logo="http://example.com/logo.png" group-title="分组 (测试)",频道 & 测试
http://example.com/stream.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 1, len(channels), "Should parse 1 channel")
	tests.AssertEqual(t, "频道 & 测试", channels[0].TvgName, "Should handle special characters in tvg-name")
	tests.AssertEqual(t, "分组 (测试)", channels[0].GroupTitle, "Should handle special characters in group-title")
}

// TestParseM3U_LargePlaylist tests parsing of a larger playlist
func TestParseM3U_LargePlaylist(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	// Build a large M3U content with 100 channels
	m3uContent := "#EXTM3U\n"
	for i := 1; i <= 100; i++ {
		channelNum := fmt.Sprintf("%d", i)
		channelID := "ch" + channelNum
		channelName := "Channel " + channelNum
		m3uContent += "#EXTINF:-1 tvg-id=\"" + channelID + "\" tvg-name=\"" + channelName + "\" tvg-logo=\"http://example.com/logo" + channelNum + ".png\" group-title=\"Group A\",Channel " + channelNum + "\n"
		m3uContent += "http://example.com/stream" + channelNum + ".m3u8\n"
	}

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 100, len(channels), "Should parse 100 channels")

	// Verify first channel
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel TvgID should match")
	tests.AssertEqual(t, "Channel 1", channels[0].Name, "First channel name should match")
}

// TestParseM3U_InvalidChannelLine tests handling of invalid channel lines
func TestParseM3U_InvalidChannelLine(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Should parse valid channels and skip invalid ones
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 valid channels")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch3", channels[1].TvgID, "Second channel should be ch3")
}

// TestParseM3U_MalformedExtinfLine tests handling of malformed EXTINF lines
func TestParseM3U_MalformedExtinfLine(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:invalid format without comma
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for malformed EXTINF")
	// Should skip malformed lines and parse valid channels
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 valid channels, skipping malformed EXTINF")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch3", channels[1].TvgID, "Second channel should be ch3")
}

// TestParseM3U_MissingChannelName tests handling of EXTINF without channel name
func TestParseM3U_MissingChannelName(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Should skip channel with empty name
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels with valid names")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch3", channels[1].TvgID, "Second channel should be ch3")
}

// TestParseM3U_URLWithoutExtinf tests handling of URL without preceding EXTINF
func TestParseM3U_URLWithoutExtinf(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
http://example.com/orphan-stream.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Should skip orphan URL and parse valid channels
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels, skipping orphan URL")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch2", channels[1].TvgID, "Second channel should be ch2")
}

// TestParseM3U_EmptyFile tests parsing of completely empty file
func TestParseM3U_EmptyFile(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := ""

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for empty file")
	tests.AssertEqual(t, 0, len(channels), "Should parse 0 channels from empty file")
}

// TestParseM3U_OnlyHeader tests parsing of file with only M3U header
func TestParseM3U_OnlyHeader(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for header-only file")
	tests.AssertEqual(t, 0, len(channels), "Should parse 0 channels from header-only file")
}

// TestParseM3U_OnlyComments tests parsing of file with only comments
func TestParseM3U_OnlyComments(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
# Comment 1
# Comment 2
# Comment 3
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for comment-only file")
	tests.AssertEqual(t, 0, len(channels), "Should parse 0 channels from comment-only file")
}

// TestParseM3U_InvalidURLFormat tests handling of invalid URL formats
func TestParseM3U_InvalidURLFormat(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
not-a-valid-url
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Parser should accept all URLs (validation happens later)
	tests.AssertEqual(t, 3, len(channels), "Should parse 3 channels")
	tests.AssertEqual(t, "not-a-valid-url", channels[1].URLs[0], "Should accept invalid URL format")
}

// TestParseM3U_MissingRequiredFields tests handling of missing required fields
func TestParseM3U_MissingRequiredFields(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Should skip channel with missing name (required field)
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels with required fields")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch3", channels[1].TvgID, "Second channel should be ch3")
}

// TestParseM3U_MalformedAttributeValues tests handling of malformed attribute values
func TestParseM3U_MalformedAttributeValues(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id=ch2 tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
#EXTINF:-1 tvg-id="ch3" tvg-name="Channel 3" tvg-logo="http://example.com/logo3.png" group-title="Group A",Channel 3
http://example.com/stream3.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Should parse all channels, but malformed attribute may not be extracted
	tests.AssertEqual(t, 3, len(channels), "Should parse 3 channels")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel tvg-id should be extracted")
	// Second channel's tvg-id may be empty due to malformed attribute
	tests.AssertEqual(t, "ch3", channels[2].TvgID, "Third channel tvg-id should be extracted")
}

// TestParseM3U_DuplicateChannels tests handling of duplicate channels
func TestParseM3U_DuplicateChannels(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	// Parser should parse all channels (deduplication happens at aggregation level)
	tests.AssertEqual(t, 3, len(channels), "Should parse all 3 channels including duplicates")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch1", channels[1].TvgID, "Second channel should also be ch1")
	tests.AssertEqual(t, "ch2", channels[2].TvgID, "Third channel should be ch2")
}

// TestParseM3U_VeryLongChannelName tests handling of very long channel names
func TestParseM3U_VeryLongChannelName(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	longName := ""
	for i := 0; i < 500; i++ {
		longName += "A"
	}

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",` + longName + `
http://example.com/stream1.m3u8
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error for long channel name")
	tests.AssertEqual(t, 1, len(channels), "Should parse 1 channel")
	tests.AssertEqual(t, longName, channels[0].Name, "Should handle very long channel name")
}

// TestParseM3U_SpecialCharactersInURL tests handling of special characters in URLs
func TestParseM3U_SpecialCharactersInURL(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)

	m3uContent := `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream?id=123&token=abc%20def&name=频道
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream#fragment
`

	channels, err := parser.ParseM3U(m3uContent, "http://example.com/playlist.m3u")
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels")
	tests.AssertEqual(t, "http://example.com/stream?id=123&token=abc%20def&name=频道", channels[0].URLs[0], "Should handle special characters in URL")
	tests.AssertEqual(t, "http://example.com/stream#fragment", channels[1].URLs[0], "Should handle URL fragments")
}
