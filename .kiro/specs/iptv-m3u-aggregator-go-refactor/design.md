# Design Document: IPTV M3U Aggregator Go Refactor

## Overview

The IPTV M3U Aggregator is being refactored from Python (Flask + APScheduler) to Go to improve performance, concurrency handling, and deployment characteristics. The Go version maintains feature parity with the Python version while leveraging Go's strengths in concurrent I/O, static compilation, and efficient resource utilization.

### Key Design Goals

1. **Performance**: Reduce latency for stream testing and aggregation through native concurrency
2. **Scalability**: Handle thousands of channels and concurrent requests efficiently
3. **Maintainability**: Clear separation of concerns with well-defined interfaces
4. **Compatibility**: Maintain data format compatibility with Python version for seamless migration
5. **Reliability**: Robust error handling and graceful degradation under failure conditions

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Web UI (HTML/CSS/JS)                    │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    HTTP Server (Gin)                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Web Routes  │  API Routes  │  Static File Handler  │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                  Application Layer                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Subscription Manager  │  Channel Manager           │  │
│  │  Aggregator            │  Stream Tester             │  │
│  │  Exporter              │  Scheduler                 │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                   Data Layer                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  File Storage (JSON)  │  In-Memory Cache            │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Language | Go 1.21+ | Native concurrency, fast compilation, single binary deployment |
| Web Framework | Gin | High performance, minimal overhead, excellent routing |
| HTTP Client | net/http + context | Built-in, efficient, supports timeouts and cancellation |
| Concurrency | goroutines + channels | Lightweight, efficient, native to Go |
| JSON Processing | encoding/json | Built-in, efficient, no external dependencies |
| File I/O | os + io/ioutil | Built-in, efficient |
| Logging | slog (Go 1.21+) | Structured logging, built-in, efficient |
| Task Scheduling | robfig/cron | Lightweight, reliable, well-tested |
| String Matching | github.com/texttheater/golang-levenshtein | Efficient string similarity calculation |
| HLS Parsing | github.com/grafov/m3u8 | Robust M3U8/HLS parsing |

## Components and Interfaces

### 1. Subscription Manager

**Responsibility**: Manage subscription sources (CRUD operations, persistence)

```go
type SubscriptionManager interface {
    AddSubscription(url, name string, enabled bool) error
    GetSubscription(url string) (*Subscription, error)
    GetAllSubscriptions() ([]*Subscription, error)
    UpdateSubscription(oldURL, newURL, name string, enabled bool) error
    RemoveSubscription(url string) error
    UpdateSubscriptionStatus(url, status string, channelCount int) error
    LoadSubscriptions() error
    SaveSubscriptions() error
}

type Subscription struct {
    URL           string    `json:"url"`
    Name          string    `json:"name"`
    Enabled       bool      `json:"enabled"`
    Status        string    `json:"status"` // active, failed, untested
    ChannelCount  int       `json:"channel_count"`
    LastUpdated   time.Time `json:"last_updated"`
    ErrorMessage  string    `json:"error_message,omitempty"`
}
```

### 2. M3U Parser

**Responsibility**: Fetch and parse M3U files from remote sources

```go
type M3UParser interface {
    FetchM3U(url string) (string, error)
    ParseM3U(content string, sourceURL string) ([]*Channel, error)
}

type Channel struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    GroupTitle   string            `json:"group_title"`
    TvgID        string            `json:"tvg_id"`
    TvgName      string            `json:"tvg_name"`
    TvgLogo      string            `json:"tvg_logo"`
    URLs         []string          `json:"urls"`
    SourceURLs   map[string]string `json:"source_urls"` // url -> source subscription URL
    TestResults  *TestResult       `json:"test_results,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
}

type TestResult struct {
    Status      string        `json:"status"` // online, offline, untested
    WorkingURL  string        `json:"working_url,omitempty"`
    ResponseTime int64        `json:"response_time_ms,omitempty"`
    TestedAt    time.Time     `json:"tested_at"`
    Details     string        `json:"details,omitempty"`
}
```

### 3. Channel Aggregator

**Responsibility**: Combine channels from multiple sources and deduplicate

```go
type ChannelAggregator interface {
    AggregateChannels(newChannels []*Channel, matchBy string, similarityThreshold float64) (total, added, updated int, error)
    GetAllChannels() []*Channel
    GetChannelsByGroup(group string) []*Channel
    GetChannelGroups() []string
    LoadChannels() error
    SaveChannels() error
}
```

### 4. Stream Tester

**Responsibility**: Test channel URLs for availability and responsiveness

```go
type StreamTester interface {
    TestStream(url string) (bool, int64, error)
    BatchTest(channels []*Channel, testAllSources bool) ([]*Channel, error)
    SetMaxWorkers(workers int)
    SetTimeout(timeout time.Duration)
}

type StreamTestConfig struct {
    Timeout        time.Duration
    MaxWorkers     int
    DeepCheck      bool
    LoopChecks     int
    LoopInterval   time.Duration
    SegmentWindow  int
}
```

### 5. Scheduler

**Responsibility**: Execute periodic tasks (updates, tests)

```go
type Scheduler interface {
    Start() error
    Stop() error
    AddJob(id string, schedule string, fn func() error) error
    RemoveJob(id string) error
    TriggerJob(id string) error
}
```

### 6. Exporter

**Responsibility**: Generate M3U and JSON exports

```go
type ChannelExporter interface {
    ExportM3U(channels []*Channel, onlyWorking bool) (string, error)
    ExportJSON(channels []*Channel, onlyWorking bool) (string, error)
    SaveExport(filename, content string) error
    GetExportList() ([]ExportFile, error)
    DeleteExport(filename string) error
}

type ExportFile struct {
    Filename    string    `json:"filename"`
    Size        int64     `json:"size"`
    CreatedAt   time.Time `json:"created_at"`
    ChannelCount int      `json:"channel_count"`
}
```

### 7. Configuration Manager

**Responsibility**: Load, validate, and persist configuration

```go
type Config struct {
    // Server
    Port                int           `json:"port"`
    Host                string        `json:"host"`
    
    // Requests
    RequestTimeout      time.Duration `json:"request_timeout"`
    StreamTestTimeout   time.Duration `json:"stream_test_timeout"`
    MaxTestWorkers      int           `json:"max_test_workers"`
    
    // Scheduling
    UpdateInterval      time.Duration `json:"update_interval"`
    TestInterval        time.Duration `json:"test_interval"`
    EnableStreamTest    bool          `json:"enable_stream_test"`
    TestAllSources      bool          `json:"test_all_sources"`
    
    // Aggregation
    MatchBy             string        `json:"match_by"` // name, tvg_id, both
    SimilarityThreshold float64       `json:"similarity_threshold"`
    
    // Deep Check (HLS)
    DeepCheck           bool          `json:"deep_check"`
    LoopChecks          int           `json:"loop_checks"`
    LoopInterval        time.Duration `json:"loop_interval"`
    SegmentWindow       int           `json:"segment_window"`
    
    // Data
    DataDir             string        `json:"data_dir"`
}
```

## Data Models

### JSON Schema for Persistence

**subscriptions.json**:
```json
[
  {
    "url": "https://example.com/iptv.m3u",
    "name": "Example IPTV",
    "enabled": true,
    "status": "active",
    "channel_count": 150,
    "last_updated": "2024-01-15T10:30:00Z",
    "error_message": ""
  }
]
```

**channels.json**:
```json
[
  {
    "id": "unique-channel-id",
    "name": "Channel Name",
    "group_title": "Group Name",
    "tvg_id": "tvg-id-123",
    "tvg_name": "TVG Name",
    "tvg_logo": "https://example.com/logo.png",
    "urls": ["https://stream1.com/channel", "https://stream2.com/channel"],
    "source_urls": {
      "https://stream1.com/channel": "https://example.com/iptv.m3u",
      "https://stream2.com/channel": "https://example.com/iptv2.m3u"
    },
    "test_results": {
      "status": "online",
      "working_url": "https://stream1.com/channel",
      "response_time_ms": 150,
      "tested_at": "2024-01-15T10:30:00Z",
      "details": ""
    },
    "created_at": "2024-01-10T08:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

**config.json**:
```json
{
  "port": 8080,
  "host": "0.0.0.0",
  "request_timeout": 30000000000,
  "stream_test_timeout": 5000000000,
  "max_test_workers": 10,
  "update_interval": 86400000000000,
  "test_interval": 86400000000000,
  "enable_stream_test": true,
  "test_all_sources": false,
  "match_by": "name",
  "similarity_threshold": 0.85,
  "deep_check": true,
  "loop_checks": 3,
  "loop_interval": 4000000000,
  "segment_window": 5,
  "data_dir": "data"
}
```

## Concurrency Strategy

### Stream Testing Concurrency

- Use worker pool pattern with configurable worker count (default 10)
- Each worker processes channels from a shared queue
- Channels are tested concurrently, with results aggregated
- Context cancellation for graceful shutdown

```go
// Pseudo-code
func (st *StreamTester) BatchTest(channels []*Channel) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    jobs := make(chan *Channel, len(channels))
    results := make(chan *Channel, len(channels))
    
    // Start workers
    for i := 0; i < st.maxWorkers; i++ {
        go st.worker(ctx, jobs, results)
    }
    
    // Send jobs
    for _, ch := range channels {
        jobs <- ch
    }
    close(jobs)
    
    // Collect results
    for i := 0; i < len(channels); i++ {
        <-results
    }
}
```

### Web Request Concurrency

- Gin framework handles concurrent requests natively
- Each request processed in its own goroutine
- Shared data access protected by sync.RWMutex for channel/subscription data
- File I/O operations use atomic writes to prevent corruption

### Background Task Concurrency

- Scheduler runs periodic tasks in separate goroutines
- Tasks can run concurrently if not explicitly serialized
- Update and test tasks are serialized to prevent conflicts

## Error Handling

### Error Categories

1. **Network Errors**: Timeout, connection refused, DNS resolution failure
   - Action: Log, mark subscription/stream as failed, continue
   
2. **Parsing Errors**: Malformed M3U, invalid JSON
   - Action: Log, skip invalid entry, continue processing
   
3. **File I/O Errors**: Permission denied, disk full, file not found
   - Action: Log, attempt retry with exponential backoff
   
4. **Configuration Errors**: Invalid config values
   - Action: Log, use defaults, continue
   
5. **Concurrency Errors**: Race conditions, deadlocks
   - Action: Prevent through proper synchronization

### Logging Strategy

- Use structured logging (slog) with levels: DEBUG, INFO, WARN, ERROR
- Log format: timestamp | level | component | message | context
- Log to stdout and optionally to file
- Include request IDs for tracing

## Testing Strategy

### Unit Testing

- Test each component in isolation with mocked dependencies
- Test data parsing (M3U, JSON)
- Test aggregation logic (deduplication, similarity matching)
- Test configuration loading and validation
- Test error handling paths

### Integration Testing

- Test component interactions (parser → aggregator → exporter)
- Test data persistence (load/save cycles)
- Test scheduler task execution
- Test concurrent operations

### Property-Based Testing

Property-based testing validates universal correctness properties across many generated inputs.

### Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.



## Correctness Properties

### Property 1: Subscription Persistence Round Trip

*For any* subscription with valid URL and metadata, when added to the system and then loaded from disk, the loaded subscription should have identical properties to the original.

**Validates: Requirements 1.1, 1.6, 10.1, 10.4**

### Property 2: M3U Parsing Completeness

*For any* valid M3U file content, parsing should extract all EXTINF lines and URLs without loss, and the number of extracted channels should equal the number of EXTINF entries in the source.

**Validates: Requirements 2.2, 2.3, 2.6**

### Property 3: Channel Aggregation Idempotence

*For any* set of channels, aggregating them twice should produce the same result as aggregating them once (same channel count, same deduplication).

**Validates: Requirements 3.1, 3.2, 3.3**

### Property 4: Stream Test Consistency

*For any* channel URL that is marked as 'online', subsequent tests of the same URL within a short time window should also mark it as 'online' (assuming no network changes).

**Validates: Requirements 4.2, 4.3, 4.4**

### Property 5: Export Format Validity

*For any* set of channels, exporting to M3U format and then parsing the exported M3U should produce channels with equivalent metadata to the original (round-trip property).

**Validates: Requirements 6.1, 6.2, 13.4**

### Property 6: Configuration Persistence

*For any* configuration modification, when saved and then loaded, the loaded configuration should have identical values to the saved configuration.

**Validates: Requirements 9.3, 9.4, 10.3**

### Property 7: Concurrent Request Isolation

*For any* concurrent requests to the web API, each request should receive consistent data snapshots without interference from other concurrent requests.

**Validates: Requirements 11.2, 11.3**

### Property 8: Data Consistency After Aggregation

*For any* aggregation operation, the persisted channel data should match the in-memory channel data, ensuring no data loss during save operations.

**Validates: Requirements 3.4, 10.2, 10.6**

### Property 9: Backward Compatibility Data Format

*For any* subscriptions.json, channels.json, or config.json file created by the Python version, the Go version should successfully load and parse it without data loss.

**Validates: Requirements 13.1, 13.2, 13.3**

### Property 10: Scheduler Task Execution

*For any* scheduled task, when triggered manually, it should execute immediately and complete successfully, and when scheduled periodically, it should execute at the configured interval.

**Validates: Requirements 5.2, 5.3, 5.4, 5.5**

## Error Handling

### Network Error Handling

- Implement exponential backoff for retries (1s, 2s, 4s, 8s max)
- Log all network errors with URL and error details
- Mark subscriptions as failed after max retries
- Continue processing other subscriptions

### File I/O Error Handling

- Use atomic writes (write to temp file, then rename)
- Implement file locking to prevent concurrent writes
- Log all I/O errors with file path and error details
- Attempt recovery by reloading from disk

### Concurrency Error Handling

- Use sync.RWMutex for shared data access
- Implement context cancellation for graceful shutdown
- Detect and log deadlocks (timeout-based detection)
- Prevent race conditions through proper synchronization

## Performance Optimization

### Stream Testing Optimization

- Implement connection pooling for HTTP clients
- Use HEAD requests for initial connectivity check
- Implement timeout-based early termination
- Cache DNS lookups
- Batch multiple URLs for same channel

### Memory Optimization

- Stream large M3U files instead of loading entirely into memory
- Implement channel deduplication in-place to reduce memory usage
- Use sync.Pool for temporary objects
- Implement garbage collection hints for large operations

### Disk I/O Optimization

- Batch write operations
- Use atomic writes to prevent corruption
- Implement incremental saves for large datasets
- Cache file modification times to detect external changes

## Deployment Considerations

### Single Binary Deployment

- Compile to single executable with embedded static files
- No external dependencies required
- Easy deployment to Docker, VMs, or bare metal

### Configuration

- Support environment variables for configuration override
- Support command-line flags for configuration
- Support config.json file for persistent configuration
- Precedence: CLI flags > environment variables > config.json > defaults

### Logging

- Structured logging to stdout
- Optional file logging with rotation
- Configurable log level
- Request tracing with correlation IDs

### Monitoring

- Expose metrics endpoint for Prometheus scraping
- Track: request count, latency, error rate, channel count, online percentage
- Health check endpoint for load balancers

## Migration Path from Python

### Phase 1: Parallel Deployment

- Run Go version alongside Python version
- Share data directory (subscriptions.json, channels.json, config.json)
- Validate data consistency between versions

### Phase 2: Cutover

- Stop Python version
- Verify Go version has all data
- Switch traffic to Go version

### Phase 3: Cleanup

- Archive Python version
- Remove Python dependencies

## API Specification

### REST Endpoints

**GET /api/subscriptions**
- Returns: Array of subscriptions
- Query params: none

**POST /api/subscriptions**
- Body: {url, name, enabled}
- Returns: Created subscription

**GET /api/subscriptions/{id}**
- Returns: Single subscription

**PUT /api/subscriptions/{id}**
- Body: {url, name, enabled}
- Returns: Updated subscription

**DELETE /api/subscriptions/{id}**
- Returns: Success status

**GET /api/channels**
- Query params: group, status, search
- Returns: Array of channels

**GET /api/channels/{id}**
- Returns: Single channel

**POST /api/channels/{id}/test**
- Returns: Test result

**GET /api/playlist.m3u**
- Returns: M3U file with online channels

**GET /api/stats**
- Returns: Aggregation statistics

**POST /api/update**
- Triggers subscription update
- Returns: Status

**POST /api/test**
- Triggers stream testing
- Returns: Status

