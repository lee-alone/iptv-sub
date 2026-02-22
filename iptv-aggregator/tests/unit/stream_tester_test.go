package unit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/yourusername/iptv-aggregator/models"
	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestTestStream_SuccessfulStream tests successful stream testing
func TestTestStream_SuccessfulStream(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for successful stream")
	tests.AssertTrue(t, online, "TestStream should return online=true for successful stream")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
	tests.AssertTrue(t, responseTime < 5000, "TestStream response time should be less than 5 seconds")
}

// TestTestStream_FailedStream tests failed stream testing
func TestTestStream_FailedStream(t *testing.T) {
	// Create a test server that returns a 404 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for failed stream (404)")
	tests.AssertFalse(t, online, "TestStream should return online=false for 404 response")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time even for failed stream")
}

// TestTestStream_ResponseTime tests response time measurement
func TestTestStream_ResponseTime(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error")
	tests.AssertTrue(t, online, "TestStream should return online=true")
	tests.AssertTrue(t, responseTime >= 100, "TestStream response time should be at least 100ms")
	tests.AssertTrue(t, responseTime < 1000, "TestStream response time should be less than 1 second")
}

// TestTestStream_Timeout tests stream testing timeout
func TestTestStream_Timeout(t *testing.T) {
	// Create a test server that delays response longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tester := services.NewStreamTester(100*time.Millisecond, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertError(t, err, "TestStream should return error on timeout")
	tests.AssertFalse(t, online, "TestStream should return online=false on timeout")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time even on timeout")
}

// TestTestStream_ConnectionRefused tests connection refused error
func TestTestStream_ConnectionRefused(t *testing.T) {
	// Use a port that's unlikely to be in use
	invalidURL := "http://127.0.0.1:1"

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(invalidURL)

	tests.AssertError(t, err, "TestStream should return error for connection refused")
	tests.AssertFalse(t, online, "TestStream should return online=false for connection refused")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_InvalidURL tests invalid URL handling
func TestTestStream_InvalidURL(t *testing.T) {
	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream("not-a-valid-url")

	tests.AssertError(t, err, "TestStream should return error for invalid URL")
	tests.AssertFalse(t, online, "TestStream should return online=false for invalid URL")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_HTTP500Error tests HTTP 500 error handling
func TestTestStream_HTTP500Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for 500 status")
	tests.AssertFalse(t, online, "TestStream should return online=false for 500 status")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_HTTP200Success tests HTTP 200 success
func TestTestStream_HTTP200Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for 200 status")
	tests.AssertTrue(t, online, "TestStream should return online=true for 200 status")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_HTTP301Redirect tests HTTP 301 redirect handling
func TestTestStream_HTTP301Redirect(t *testing.T) {
	// Create a final server
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer finalServer.Close()

	// Create a redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(redirectServer.URL)

	tests.AssertNoError(t, err, "TestStream should handle redirects")
	tests.AssertTrue(t, online, "TestStream should return online=true after redirect")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_HTTP403Forbidden tests HTTP 403 forbidden
func TestTestStream_HTTP403Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for 403 status")
	tests.AssertFalse(t, online, "TestStream should return online=false for 403 status")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestTestStream_HTTP206PartialContent tests HTTP 206 partial content
func TestTestStream_HTTP206PartialContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
	}))
	defer server.Close()

	tester := services.NewStreamTester(5*time.Second, 5)
	online, responseTime, err := tester.TestStream(server.URL)

	tests.AssertNoError(t, err, "TestStream should not return error for 206 status")
	tests.AssertTrue(t, online, "TestStream should return online=true for 206 status (2xx-3xx range)")
	tests.AssertTrue(t, responseTime >= 0, "TestStream should return non-negative response time")
}

// TestBatchTest_MultipleChannels tests batch testing with multiple channels
func TestBatchTest_MultipleChannels(t *testing.T) {
	// Create test servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server3.Close()

	// Create channels
	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server1.URL, "http://source1.com"),
		models.NewChannel("Channel 2", "Group A", "ch2", "Channel 2", "", server2.URL, "http://source1.com"),
		models.NewChannel("Channel 3", "Group B", "ch3", "Channel 3", "", server3.URL, "http://source2.com"),
	}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 3, len(result), "BatchTest should return all channels")
	tests.AssertTrue(t, result[0].IsOnline(), "First channel should be online")
	tests.AssertTrue(t, result[1].IsOnline(), "Second channel should be online")
	tests.AssertTrue(t, result[2].IsOffline(), "Third channel should be offline")
}

// TestBatchTest_ConcurrencyLimit tests batch testing respects concurrency limit
func TestBatchTest_ConcurrencyLimit(t *testing.T) {
	// Track concurrent requests
	maxConcurrent := 0
	currentConcurrent := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		currentConcurrent++
		if currentConcurrent > maxConcurrent {
			maxConcurrent = currentConcurrent
		}
		mu.Unlock()

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		currentConcurrent--
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create 10 channels
	channels := make([]*models.Channel, 10)
	for i := 0; i < 10; i++ {
		channels[i] = models.NewChannel(
			fmt.Sprintf("Channel %d", i+1),
			"Group A",
			fmt.Sprintf("ch%d", i+1),
			fmt.Sprintf("Channel %d", i+1),
			"",
			server.URL,
			"http://source.com",
		)
	}

	// Test with maxWorkers=3
	tester := services.NewStreamTester(5*time.Second, 3)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 10, len(result), "BatchTest should return all channels")
	tests.AssertTrue(t, maxConcurrent <= 3, fmt.Sprintf("Concurrent requests should not exceed 3, got %d", maxConcurrent))
}

// TestBatchTest_EmptyChannelList tests batch testing with empty channel list
func TestBatchTest_EmptyChannelList(t *testing.T) {
	channels := []*models.Channel{}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error for empty list")
	tests.AssertEqual(t, 0, len(result), "BatchTest should return empty list")
}

// TestBatchTest_SingleChannel tests batch testing with single channel
func TestBatchTest_SingleChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return single channel")
	tests.AssertTrue(t, result[0].IsOnline(), "Channel should be online")
}

// TestBatchTest_TestAllSources tests batch testing with testAllSources=true
func TestBatchTest_TestAllSources(t *testing.T) {
	// Create servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Create channel with multiple URLs
	channel := models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server1.URL, "http://source1.com")
	channel.AddURL(server2.URL, "http://source2.com")

	channels := []*models.Channel{channel}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, true)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return single channel")
	tests.AssertTrue(t, result[0].IsOnline(), "Channel should be online (second URL works)")
	tests.AssertEqual(t, server2.URL, result[0].TestResults.WorkingURL, "Working URL should be the second server")
}

// TestBatchTest_TestFirstSourceOnly tests batch testing with testAllSources=false
func TestBatchTest_TestFirstSourceOnly(t *testing.T) {
	// Create servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Create channel with multiple URLs
	channel := models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server1.URL, "http://source1.com")
	channel.AddURL(server2.URL, "http://source2.com")

	channels := []*models.Channel{channel}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return single channel")
	tests.AssertTrue(t, result[0].IsOffline(), "Channel should be offline (only first URL tested)")
}

// TestBatchTest_TimeoutHandling tests batch testing timeout handling
func TestBatchTest_TimeoutHandling(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	tester := services.NewStreamTester(100*time.Millisecond, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error (timeout is handled internally)")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return channel")
	tests.AssertTrue(t, result[0].IsOffline(), "Channel should be offline due to timeout")
}

// TestBatchTest_MixedResults tests batch testing with mixed online/offline results
func TestBatchTest_MixedResults(t *testing.T) {
	// Create servers with different responses
	servers := make([]*httptest.Server, 5)
	for i := 0; i < 5; i++ {
		statusCode := http.StatusOK
		if i%2 == 0 {
			statusCode = http.StatusNotFound
		}
		code := statusCode // Capture for closure
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))
		defer servers[i].Close()
	}

	// Create channels
	channels := make([]*models.Channel, 5)
	for i := 0; i < 5; i++ {
		channels[i] = models.NewChannel(
			fmt.Sprintf("Channel %d", i+1),
			"Group A",
			fmt.Sprintf("ch%d", i+1),
			fmt.Sprintf("Channel %d", i+1),
			"",
			servers[i].URL,
			"http://source.com",
		)
	}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 5, len(result), "BatchTest should return all channels")

	// Check results
	onlineCount := 0
	offlineCount := 0
	for _, ch := range result {
		if ch.IsOnline() {
			onlineCount++
		} else if ch.IsOffline() {
			offlineCount++
		}
	}

	tests.AssertEqual(t, 2, onlineCount, "Should have 2 online channels")
	tests.AssertEqual(t, 3, offlineCount, "Should have 3 offline channels")
}

// TestBatchTest_LargeChannelList tests batch testing with large channel list
func TestBatchTest_LargeChannelList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create 100 channels
	channels := make([]*models.Channel, 100)
	for i := 0; i < 100; i++ {
		channels[i] = models.NewChannel(
			fmt.Sprintf("Channel %d", i+1),
			"Group A",
			fmt.Sprintf("ch%d", i+1),
			fmt.Sprintf("Channel %d", i+1),
			"",
			server.URL,
			"http://source.com",
		)
	}

	tester := services.NewStreamTester(5*time.Second, 10)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error for large list")
	tests.AssertEqual(t, 100, len(result), "BatchTest should return all 100 channels")

	// Check all are online
	for i, ch := range result {
		tests.AssertTrue(t, ch.IsOnline(), fmt.Sprintf("Channel %d should be online", i+1))
	}
}

// TestBatchTest_ChannelWithNoURLs tests batch testing with channel that has no URLs
func TestBatchTest_ChannelWithNoURLs(t *testing.T) {
	// Create a channel with no URLs
	channel := &models.Channel{
		ID:         "ch1",
		Name:       "Channel 1",
		GroupTitle: "Group A",
		TvgID:      "ch1",
		URLs:       []string{},
		SourceURLs: make(map[string]string),
	}

	channels := []*models.Channel{channel}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return channel")
	// Channel with no URLs will have TestResults set to offline status
	tests.AssertTrue(t, result[0].IsOffline() || result[0].IsUntested(), "Channel with no URLs should be offline or untested")
}

// TestBatchTest_ResponseTimeRecorded tests that response time is recorded
func TestBatchTest_ResponseTimeRecorded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return channel")
	tests.AssertTrue(t, result[0].TestResults.ResponseTime >= 50, "Response time should be recorded")
	tests.AssertTrue(t, result[0].TestResults.ResponseTime < 1000, "Response time should be reasonable")
}

// TestBatchTest_WorkingURLRecorded tests that working URL is recorded
func TestBatchTest_WorkingURLRecorded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertEqual(t, 1, len(result), "BatchTest should return channel")
	tests.AssertEqual(t, server.URL, result[0].TestResults.WorkingURL, "Working URL should be recorded")
}

// TestBatchTest_StatusUpdated tests that channel status is updated
func TestBatchTest_StatusUpdated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	// Initially should be untested
	tests.AssertTrue(t, channels[0].IsUntested(), "Channel should be untested initially")

	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertTrue(t, result[0].IsOnline(), "Channel should be online after batch test")
	tests.AssertEqual(t, "online", result[0].TestResults.Status, "Channel status should be 'online'")
}

// TestBatchTest_TestedAtTimestamp tests that tested timestamp is set
func TestBatchTest_TestedAtTimestamp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	channels := []*models.Channel{
		models.NewChannel("Channel 1", "Group A", "ch1", "Channel 1", "", server.URL, "http://source.com"),
	}

	beforeTest := time.Now().Add(-1 * time.Second) // Add buffer for clock precision
	tester := services.NewStreamTester(5*time.Second, 5)
	result, err := tester.BatchTest(channels, false)
	afterTest := time.Now().Add(1 * time.Second) // Add buffer for clock precision

	tests.AssertNoError(t, err, "BatchTest should not return error")
	tests.AssertTrue(t, result[0].TestResults.TestedAt.After(beforeTest), "TestedAt should be after test start")
	tests.AssertTrue(t, result[0].TestResults.TestedAt.Before(afterTest), "TestedAt should be before test end")
}
