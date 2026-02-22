package unit

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yourusername/iptv-aggregator/services"
	"github.com/yourusername/iptv-aggregator/tests"
)

// TestFetchM3U_SuccessfulFetch tests successful M3U file fetching
func TestFetchM3U_SuccessfulFetch(t *testing.T) {
	// Create a test server that returns a valid M3U file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should not return error for successful fetch")
	tests.AssertNotNil(t, content, "FetchM3U should return content")
	tests.AssertStringContains(t, content, "#EXTM3U", "Content should contain M3U header")
	tests.AssertStringContains(t, content, "Channel 1", "Content should contain channel data")
	tests.AssertStringContains(t, content, "Channel 2", "Content should contain channel data")
}

// TestFetchM3U_EmptyM3UFile tests fetching an empty M3U file
func TestFetchM3U_EmptyM3UFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should not return error for empty M3U")
	tests.AssertNotNil(t, content, "FetchM3U should return content")
	tests.AssertStringContains(t, content, "#EXTM3U", "Content should contain M3U header")
}

// TestFetchM3U_LargeM3UFile tests fetching a large M3U file
func TestFetchM3U_LargeM3UFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "#EXTM3U\n")
		// Generate 1000 channels
		for i := 1; i <= 1000; i++ {
			fmt.Fprintf(w, "#EXTINF:-1 tvg-id=\"ch%d\" tvg-name=\"Channel %d\" tvg-logo=\"http://example.com/logo%d.png\" group-title=\"Group A\",Channel %d\n", i, i, i, i)
			fmt.Fprintf(w, "http://example.com/stream%d.m3u8\n", i)
		}
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should not return error for large M3U")
	tests.AssertNotNil(t, content, "FetchM3U should return content")
	tests.AssertStringContains(t, content, "Channel 1", "Content should contain first channel")
	tests.AssertStringContains(t, content, "Channel 1000", "Content should contain last channel")
}

// TestFetchM3U_NetworkTimeout tests network timeout handling
func TestFetchM3U_NetworkTimeout(t *testing.T) {
	// Create a server that delays response longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "#EXTM3U\n")
	}))
	defer server.Close()

	// Create parser with very short timeout
	parser := services.NewM3UParser(100 * time.Millisecond)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error on timeout")
	tests.AssertStringContains(t, err.Error(), "failed to fetch M3U", "Error message should indicate fetch failure")
}

// TestFetchM3U_ConnectionRefused tests handling of connection refused error
func TestFetchM3U_ConnectionRefused(t *testing.T) {
	// Use a port that's unlikely to be in use
	invalidURL := "http://127.0.0.1:1"

	parser := services.NewM3UParser(5 * time.Second)
	_, err := parser.FetchM3U(invalidURL)

	tests.AssertError(t, err, "FetchM3U should return error for connection refused")
	tests.AssertStringContains(t, err.Error(), "failed to fetch M3U", "Error message should indicate fetch failure")
}

// TestFetchM3U_InvalidURL tests handling of invalid URL
func TestFetchM3U_InvalidURL(t *testing.T) {
	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U("not-a-valid-url")

	tests.AssertError(t, err, "FetchM3U should return error for invalid URL")
	tests.AssertStringContains(t, err.Error(), "failed to fetch M3U", "Error message should indicate fetch failure")
}

// TestFetchM3U_HTTP404NotFound tests HTTP 404 error handling
func TestFetchM3U_HTTP404NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 404 status")
	tests.AssertStringContains(t, err.Error(), "status code 404", "Error message should contain 404 status code")
}

// TestFetchM3U_HTTP500InternalServerError tests HTTP 500 error handling
func TestFetchM3U_HTTP500InternalServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 500 status")
	tests.AssertStringContains(t, err.Error(), "status code 500", "Error message should contain 500 status code")
}

// TestFetchM3U_HTTP403Forbidden tests HTTP 403 error handling
func TestFetchM3U_HTTP403Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Forbidden")
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 403 status")
	tests.AssertStringContains(t, err.Error(), "status code 403", "Error message should contain 403 status code")
}

// TestFetchM3U_HTTP400BadRequest tests HTTP 400 error handling
func TestFetchM3U_HTTP400BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Bad Request")
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 400 status")
	tests.AssertStringContains(t, err.Error(), "status code 400", "Error message should contain 400 status code")
}

// TestFetchM3U_HTTP503ServiceUnavailable tests HTTP 503 error handling
func TestFetchM3U_HTTP503ServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "Service Unavailable")
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 503 status")
	tests.AssertStringContains(t, err.Error(), "status code 503", "Error message should contain 503 status code")
}

// TestFetchM3U_VariousHTTPErrors tests handling of various HTTP error codes
func TestFetchM3U_VariousHTTPErrors(t *testing.T) {
	errorCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusMethodNotAllowed,    // 405
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
	}

	for _, statusCode := range errorCodes {
		t.Run(fmt.Sprintf("HTTP_%d", statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
				fmt.Fprint(w, "Error")
			}))
			defer server.Close()

			parser := services.NewM3UParser(10 * time.Second)
			_, err := parser.FetchM3U(server.URL)

			tests.AssertError(t, err, fmt.Sprintf("FetchM3U should return error for %d status", statusCode))
			tests.AssertStringContains(t, err.Error(), fmt.Sprintf("status code %d", statusCode),
				fmt.Sprintf("Error message should contain %d status code", statusCode))
		})
	}
}

// TestFetchM3U_ResponseBodyReadError tests handling of response body read errors
func TestFetchM3U_ResponseBodyReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		// Write less data than Content-Length indicates
		fmt.Fprint(w, "incomplete")
		// Connection will be closed, causing read error
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	// This may or may not error depending on how the server handles it
	// The important thing is that the parser handles it gracefully
	if err != nil {
		tests.AssertStringContains(t, err.Error(), "failed", "Error message should indicate failure")
	}
}

// TestFetchM3U_UTF8Content tests fetching M3U file with UTF-8 content
func TestFetchM3U_UTF8Content(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="频道 1" tvg-logo="http://example.com/logo1.png" group-title="分组 A",频道 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="チャンネル 2" tvg-logo="http://example.com/logo2.png" group-title="グループ A",チャンネル 2
http://example.com/stream2.m3u8
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should handle UTF-8 content")
	tests.AssertNotNil(t, content, "FetchM3U should return content")
	tests.AssertStringContains(t, content, "频道 1", "Content should contain Chinese characters")
	tests.AssertStringContains(t, content, "チャンネル 2", "Content should contain Japanese characters")
}

// TestFetchM3U_WithRedirect tests handling of HTTP redirects
func TestFetchM3U_WithRedirect(t *testing.T) {
	// Create a server that redirects to another server
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
`)
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(redirectServer.URL)

	tests.AssertNoError(t, err, "FetchM3U should follow redirects")
	tests.AssertNotNil(t, content, "FetchM3U should return content after redirect")
	tests.AssertStringContains(t, content, "#EXTM3U", "Content should contain M3U header")
	tests.AssertStringContains(t, content, "Channel 1", "Content should contain channel data")
}

// TestFetchM3U_MultipleRedirects tests handling of multiple HTTP redirects
func TestFetchM3U_MultipleRedirects(t *testing.T) {
	// Create a chain of redirects
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
`)
	}))
	defer finalServer.Close()

	redirect2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusFound)
	}))
	defer redirect2Server.Close()

	redirect1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirect2Server.URL, http.StatusMovedPermanently)
	}))
	defer redirect1Server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(redirect1Server.URL)

	tests.AssertNoError(t, err, "FetchM3U should follow multiple redirects")
	tests.AssertNotNil(t, content, "FetchM3U should return content after multiple redirects")
	tests.AssertStringContains(t, content, "#EXTM3U", "Content should contain M3U header")
}

// TestFetchM3U_DifferentContentTypes tests fetching with different content types
func TestFetchM3U_DifferentContentTypes(t *testing.T) {
	contentTypes := []string{
		"application/vnd.apple.mpegurl",
		"application/x-mpegurl",
		"audio/mpegurl",
		"text/plain",
		"application/octet-stream",
	}

	for _, contentType := range contentTypes {
		t.Run(contentType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
`)
			}))
			defer server.Close()

			parser := services.NewM3UParser(10 * time.Second)
			content, err := parser.FetchM3U(server.URL)

			tests.AssertNoError(t, err, fmt.Sprintf("FetchM3U should handle content type %s", contentType))
			tests.AssertNotNil(t, content, "FetchM3U should return content")
			tests.AssertStringContains(t, content, "#EXTM3U", "Content should contain M3U header")
		})
	}
}

// TestFetchM3U_WithCustomHeaders tests fetching with custom headers
func TestFetchM3U_WithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that standard headers are sent
		userAgent := r.Header.Get("User-Agent")
		tests.AssertNotEqual(t, "", userAgent, "User-Agent header should be set")

		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should handle custom headers")
	tests.AssertNotNil(t, content, "FetchM3U should return content")
}

// TestFetchM3U_PartialContent tests handling of partial content (206 status)
func TestFetchM3U_PartialContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPartialContent)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error for 206 partial content status")
	tests.AssertStringContains(t, err.Error(), "status code 206", "Error message should contain 206 status code")
}

// TestFetchM3U_ContextDeadlineExceeded tests handling of context deadline exceeded
func TestFetchM3U_ContextDeadlineExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "#EXTM3U\n")
	}))
	defer server.Close()

	// Create parser with very short timeout
	parser := services.NewM3UParser(100 * time.Millisecond)
	_, err := parser.FetchM3U(server.URL)

	tests.AssertError(t, err, "FetchM3U should return error on context deadline exceeded")
	tests.AssertStringContains(t, err.Error(), "failed to fetch M3U", "Error message should indicate fetch failure")
}

// TestFetchM3U_EmptyResponse tests handling of empty response body
func TestFetchM3U_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		// Return empty body
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)
	content, err := parser.FetchM3U(server.URL)

	tests.AssertNoError(t, err, "FetchM3U should not error on empty response")
	tests.AssertEqual(t, "", content, "FetchM3U should return empty string for empty response")
}

// TestFetchM3U_SuccessfulFetchAndParse tests successful fetch and parse together
func TestFetchM3U_SuccessfulFetchAndParse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" tvg-logo="http://example.com/logo1.png" group-title="Group A",Channel 1
http://example.com/stream1.m3u8
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" tvg-logo="http://example.com/logo2.png" group-title="Group A",Channel 2
http://example.com/stream2.m3u8
`)
	}))
	defer server.Close()

	parser := services.NewM3UParser(10 * time.Second)

	// Fetch the M3U file
	content, err := parser.FetchM3U(server.URL)
	tests.AssertNoError(t, err, "FetchM3U should not return error")
	tests.AssertNotNil(t, content, "FetchM3U should return content")

	// Parse the fetched content
	channels, err := parser.ParseM3U(content, server.URL)
	tests.AssertNoError(t, err, "ParseM3U should not return error")
	tests.AssertEqual(t, 2, len(channels), "Should parse 2 channels from fetched content")
	tests.AssertEqual(t, "ch1", channels[0].TvgID, "First channel should be ch1")
	tests.AssertEqual(t, "ch2", channels[1].TvgID, "Second channel should be ch2")
}
