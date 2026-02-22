package unit

import (
	"testing"

	"github.com/yourusername/iptv-aggregator/models"
	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestChannelDeduplication_ExactMatch tests exact channel deduplication
func TestChannelDeduplication_ExactMatch(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create two identical channels
	ch1 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	channels := []*models.Channel{ch1, ch2}

	added, updated, _, err := aggregator.AggregateChannels(channels, "tvg_id", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should update 1 channel (merge)")

	// Verify merged channel has both URLs
	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 1, len(allChannels), "Should have 1 channel after deduplication")
	tests.AssertEqual(t, 2, len(allChannels[0].URLs), "Merged channel should have 2 URLs")
}

// TestChannelDeduplication_ByTvgID tests deduplication by tvg_id matching mode
func TestChannelDeduplication_ByTvgID(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create channels with same tvg_id but different names
	ch1 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("中央电视台 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	channels := []*models.Channel{ch1, ch2}

	added, updated, _, err := aggregator.AggregateChannels(channels, "tvg_id", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should update 1 channel (merge by tvg_id)")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 1, len(allChannels), "Should have 1 channel after deduplication")
	tests.AssertEqual(t, 2, len(allChannels[0].URLs), "Merged channel should have 2 URLs")
}

// TestChannelDeduplication_ByName tests deduplication by name matching mode
func TestChannelDeduplication_ByName(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create channels with similar names but different tvg_id
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	channels := []*models.Channel{ch1, ch2}

	added, updated, _, err := aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should update 1 channel (merge by name similarity)")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 1, len(allChannels), "Should have 1 channel after deduplication")
	tests.AssertEqual(t, 2, len(allChannels[0].URLs), "Merged channel should have 2 URLs")
}

// TestChannelDeduplication_ByBoth tests deduplication by both tvg_id and name
func TestChannelDeduplication_ByBoth(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create channels with same tvg_id
	ch1 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// Create channel with similar name but different tvg_id
	ch3 := models.NewChannel("CCTV1", "央视", "id3", "CCTV 1", "http://logo.png", "http://stream3.m3u8", "http://source3.m3u")

	channels := []*models.Channel{ch1, ch2, ch3}

	added, updated, _, err := aggregator.AggregateChannels(channels, "both", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 2, updated, "Should update 2 channels (merge by tvg_id and name)")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 1, len(allChannels), "Should have 1 channel after deduplication")
	tests.AssertEqual(t, 3, len(allChannels[0].URLs), "Merged channel should have 3 URLs")
}

// TestChannelDeduplication_NoMatch tests channels that don't match
func TestChannelDeduplication_NoMatch(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create completely different channels
	ch1 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 2", "央视", "cctv2", "CCTV 2", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")
	ch3 := models.NewChannel("CCTV 3", "央视", "cctv3", "CCTV 3", "http://logo.png", "http://stream3.m3u8", "http://source3.m3u")

	channels := []*models.Channel{ch1, ch2, ch3}

	added, updated, _, err := aggregator.AggregateChannels(channels, "tvg_id", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 3, added, "Should add 3 channels")
	tests.AssertEqual(t, 0, updated, "Should update 0 channels")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 channels")
}

// TestChannelDeduplication_EmptyChannels tests aggregation with empty channel list
func TestChannelDeduplication_EmptyChannels(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	channels := []*models.Channel{}

	added, updated, _, err := aggregator.AggregateChannels(channels, "tvg_id", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 0, added, "Should add 0 channels")
	tests.AssertEqual(t, 0, updated, "Should update 0 channels")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 0, len(allChannels), "Should have 0 channels")
}

// TestChannelDeduplication_MultipleRounds tests deduplication across multiple aggregation rounds
func TestChannelDeduplication_MultipleRounds(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// First round: add 2 channels
	ch1 := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 2", "央视", "cctv2", "CCTV 2", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added1, _, _, err1 := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "tvg_id", 0.8)
	tests.AssertNoError(t, err1, "First aggregation should not return error")
	tests.AssertEqual(t, 2, added1, "First round should add 2 channels")

	// Second round: add duplicate of ch1 and a new channel
	ch1Dup := models.NewChannel("CCTV 1", "央视", "cctv1", "CCTV 1", "http://logo.png", "http://stream1b.m3u8", "http://source1b.m3u")
	ch3 := models.NewChannel("CCTV 3", "央视", "cctv3", "CCTV 3", "http://logo.png", "http://stream3.m3u8", "http://source3.m3u")

	added2, updated2, _, err2 := aggregator.AggregateChannels([]*models.Channel{ch1Dup, ch3}, "tvg_id", 0.8)
	tests.AssertNoError(t, err2, "Second aggregation should not return error")
	tests.AssertEqual(t, 1, added2, "Second round should add 1 new channel")
	tests.AssertEqual(t, 1, updated2, "Second round should update 1 channel")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 3, len(allChannels), "Should have 3 channels total")

	// Find CCTV 1 and verify it has 2 URLs
	var cctv1 *models.Channel
	for _, ch := range allChannels {
		if ch.TvgID == "cctv1" {
			cctv1 = ch
			break
		}
	}
	tests.AssertNotNil(t, cctv1, "CCTV 1 should exist")
	tests.AssertEqual(t, 2, len(cctv1.URLs), "CCTV 1 should have 2 URLs after merge")
}

// TestSimilarity_IdenticalStrings tests similarity of identical strings
func TestSimilarity_IdenticalStrings(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// Test with name matching mode
	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge identical names")
}

// TestSimilarity_HighSimilarity tests similarity with high similarity threshold
func TestSimilarity_HighSimilarity(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// "CCTV 1" vs "CCTV1" - very similar
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// With threshold 0.8, should match
	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge similar names with threshold 0.8")
}

// TestSimilarity_LowSimilarity tests similarity with low similarity threshold
func TestSimilarity_LowSimilarity(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// "CCTV 1" vs "CCTV 3" - more different
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 3", "央视", "id2", "CCTV 3", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// With threshold 0.9, should not match
	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.9)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 2, added, "Should add 2 channels")
	tests.AssertEqual(t, 0, updated, "Should not merge dissimilar names with high threshold")
}

// TestSimilarity_ThresholdBoundary tests similarity at threshold boundary
func TestSimilarity_ThresholdBoundary(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create channels with names that are exactly at the threshold
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// Test with different thresholds
	// With threshold 0.9, might not match
	_, _, _, err1 := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.9)
	tests.AssertNoError(t, err1, "AggregateChannels should not return error")

	// With threshold 0.5, should definitely match
	aggregator2 := services.NewChannelAggregator("")
	ch3 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch4 := models.NewChannel("CCTV1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")
	added2, updated2, _, err2 := aggregator2.AggregateChannels([]*models.Channel{ch3, ch4}, "name", 0.5)
	tests.AssertNoError(t, err2, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added2, "Should add 1 channel with low threshold")
	tests.AssertEqual(t, 1, updated2, "Should merge with low threshold")
}

// TestSimilarity_EmptyStrings tests similarity with empty strings
func TestSimilarity_EmptyStrings(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Channel with empty name
	ch1 := models.NewChannel("", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 2, added, "Should add 2 channels (empty name doesn't match)")
	tests.AssertEqual(t, 0, updated, "Should not merge empty name with non-empty")
}

// TestSimilarity_SpecialCharacters tests similarity with special characters
func TestSimilarity_SpecialCharacters(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Channels with special characters
	ch1 := models.NewChannel("CCTV & 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV & 1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge identical names with special characters")
}

// TestSimilarity_ChineseCharacters tests similarity with Chinese characters
func TestSimilarity_ChineseCharacters(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Channels with Chinese characters
	ch1 := models.NewChannel("中央电视台 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("中央电视台1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge similar Chinese names")
}

// TestSimilarity_CaseSensitivity tests similarity with different cases
func TestSimilarity_CaseSensitivity(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Channels with different cases
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("cctv 1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// Levenshtein distance treats case as different, so similarity is lower
	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	// With threshold 0.8, case difference may prevent merge
	tests.AssertEqual(t, 2, added, "Should add 2 channels")
	tests.AssertEqual(t, 0, updated, "Should not merge due to case difference with threshold 0.8")
}

// TestSimilarity_LongStrings tests similarity with long strings
func TestSimilarity_LongStrings(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	longName1 := "This is a very long channel name with many words and characters"
	longName2 := "This is a very long channel name with many words and characters"

	ch1 := models.NewChannel(longName1, "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel(longName2, "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge identical long names")
}

// TestSimilarity_OneCharacterDifference tests similarity with one character difference
func TestSimilarity_OneCharacterDifference(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// "CCTV 1" vs "CCTV 2" - one character difference
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 2", "央视", "id2", "CCTV 2", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	// With threshold 0.8, should not match (similarity ~0.83)
	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.85)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 2, added, "Should add 2 channels")
	tests.AssertEqual(t, 0, updated, "Should not merge with high threshold")
}

// TestSimilarity_ZeroThreshold tests similarity with zero threshold
func TestSimilarity_ZeroThreshold(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Any two channels should match with threshold 0
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 2", "央视", "id2", "CCTV 2", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 0.0)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge with zero threshold")
}

// TestSimilarity_OneThreshold tests similarity with threshold of 1.0
func TestSimilarity_OneThreshold(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Only identical channels should match with threshold 1.0
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV 1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")

	added, updated, _, err := aggregator.AggregateChannels([]*models.Channel{ch1, ch2}, "name", 1.0)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 1, added, "Should add 1 channel")
	tests.AssertEqual(t, 1, updated, "Should merge identical names with threshold 1.0")
}

// TestSimilarity_MultipleChannelsWithThreshold tests similarity with multiple channels and threshold
func TestSimilarity_MultipleChannelsWithThreshold(t *testing.T) {
	aggregator := services.NewChannelAggregator("")

	// Create multiple channels with varying similarity
	ch1 := models.NewChannel("CCTV 1", "央视", "id1", "CCTV 1", "http://logo.png", "http://stream1.m3u8", "http://source1.m3u")
	ch2 := models.NewChannel("CCTV1", "央视", "id2", "CCTV 1", "http://logo.png", "http://stream2.m3u8", "http://source2.m3u")
	ch3 := models.NewChannel("CCTV 2", "央视", "id3", "CCTV 2", "http://logo.png", "http://stream3.m3u8", "http://source3.m3u")
	ch4 := models.NewChannel("CCTV2", "央视", "id4", "CCTV 2", "http://logo.png", "http://stream4.m3u8", "http://source4.m3u")

	channels := []*models.Channel{ch1, ch2, ch3, ch4}

	added, updated, _, err := aggregator.AggregateChannels(channels, "name", 0.8)
	tests.AssertNoError(t, err, "AggregateChannels should not return error")
	tests.AssertEqual(t, 2, added, "Should add 2 channels")
	tests.AssertEqual(t, 2, updated, "Should update 2 channels")

	allChannels := aggregator.GetAllChannels()
	tests.AssertEqual(t, 2, len(allChannels), "Should have 2 channels after deduplication")
}
