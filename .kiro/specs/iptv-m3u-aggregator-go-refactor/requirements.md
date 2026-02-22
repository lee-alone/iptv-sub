# 需求文档：IPTV M3U 聚合器 Go 重构

## 简介

本文档规定了将现有的基于 Python 的 IPTV M3U 聚合平台重构为 Go 的需求。重构保持所有现有功能，同时改进性能、并发处理和部署特性。该系统聚合多个 IPTV M3U 订阅源，对频道进行去重，测试流可用性，并提供 Web UI 和 API 接口用于管理和消费。

## 术语表

- **M3U 格式**：多媒体文件的播放列表格式，通常用于带有 EXTINF 元数据的 IPTV 流
- **订阅源**：指向包含 IPTV 频道定义的 M3U 文件的 URL
- **频道**：单个 IPTV 流条目，包含元数据（名称、分组、logo、URL）
- **聚合**：将来自多个源的频道合并并去重的过程
- **流测试**：验证频道 URL 可用性和响应性的过程
- **深度检查**：检测 HLS 流循环和卡顿的高级流测试
- **工作 URL**：已验证为可访问和响应的频道 URL
- **调度器**：在配置的时间间隔执行周期性任务（更新、测试）的组件
- **Web UI**：用于管理订阅和查看频道的 HTML/CSS/JavaScript 界面
- **API**：用于以编程方式访问聚合频道和管理功能的 RESTful 端点
- **导出**：从聚合频道生成 M3U 或 JSON 文件的过程
- **持久化**：将订阅、频道和配置数据存储到磁盘

## 需求

### 需求 1：订阅源管理

**用户故事**：作为平台管理员，我想管理 IPTV 订阅源，以便我可以控制哪些源被聚合并维护频道数据库。

#### 验收标准

1. 当管理员添加新的订阅源 URL 时，系统应存储它及其元数据（名称、启用状态）并持久化到磁盘
2. 当管理员检索订阅列表时，系统应返回所有存储的订阅及其当前状态（活跃、失败、未测试）
3. 当管理员更新订阅源 URL 或元数据时，系统应修改存储的订阅并持久化更改
4. 当管理员删除订阅源时，系统应从存储中删除它并防止从该源进行进一步聚合
5. 当添加或修改订阅源时，系统应在接受前验证 URL 格式
6. 当系统启动时，系统应从磁盘加载所有持久化的订阅到内存

### 需求 2：M3U 文件解析和频道提取

**用户故事**：作为系统组件，我想从远程源解析 M3U 文件，以便我可以提取频道元数据用于聚合。

#### 验收标准

1. 当系统从订阅源 URL 获取 M3U 文件时，解析器应使用可配置的超时时间（默认 30 秒）检索内容
2. 当解析器接收 M3U 内容时，解析器应提取所有 EXTINF 元数据行和频道 URL
3. 当解析 EXTINF 行时，解析器应提取 tvg-id、tvg-name、tvg-logo、group-title 和频道名称属性
4. 当提取频道 URL 时，解析器应将其与其源订阅源 URL 关联以进行跟踪
5. 当解析器遇到格式错误的 M3U 内容时，解析器应跳过无效条目并继续处理有效条目
6. 当解析完成时，解析器应返回包含所有提取元数据的频道对象列表

### Requirement 3: Channel Aggregation and Deduplication

**User Story:** As a system component, I want to aggregate channels from multiple sources and deduplicate them, so that users see a unified channel list without redundancy.

#### Acceptance Criteria

1. WHEN the Aggregator receives channels from multiple sources, THE Aggregator SHALL combine them into a single list
2. WHEN duplicate channels are detected (by name or tvg-id based on configuration), THE Aggregator SHALL merge them and preserve all available URLs
3. WHEN merging duplicate channels, THE Aggregator SHALL calculate name similarity using configurable threshold (default 0.85)
4. WHEN aggregation completes, THE Aggregator SHALL persist the unified channel list to disk
5. WHEN the system starts, THE Aggregator SHALL load the persisted channel list from disk
6. WHEN channels are aggregated, THE Aggregator SHALL track which source each URL came from for debugging

### Requirement 4: Stream Availability Testing

**User Story:** As a system component, I want to test channel stream URLs for availability, so that users can identify working channels and the system can maintain quality metrics.

#### Acceptance Criteria

1. WHEN the StreamTester receives a batch of channels, THE StreamTester SHALL test each URL concurrently (up to max_workers limit, default 10)
2. WHEN testing a stream URL, THE StreamTester SHALL attempt connection with a configurable timeout (default 5 seconds)
3. WHEN a stream URL responds successfully, THE StreamTester SHALL mark it as 'online' and record the response time
4. WHEN a stream URL fails to respond or times out, THE StreamTester SHALL mark it as 'offline'
5. WHEN deep_check is enabled, THE StreamTester SHALL perform advanced HLS stream validation to detect loops and stalls
6. WHEN deep_check detects an HLS stream loop, THE StreamTester SHALL mark the stream as 'offline' despite initial connectivity
7. WHEN a channel has multiple URLs, THE StreamTester SHALL test all URLs and identify the working one
8. WHEN testing completes, THE StreamTester SHALL persist test results to the channel data
9. WHEN the system is configured with test_all_sources=true, THE StreamTester SHALL test all URLs for each channel; otherwise test only primary URL

### Requirement 5: Periodic Task Scheduling

**User Story:** As a system component, I want to schedule periodic tasks for subscription updates and stream testing, so that the channel database stays current and quality metrics are maintained.

#### Acceptance Criteria

1. WHEN the system starts, THE Scheduler SHALL initialize configured periodic tasks without executing them immediately
2. WHEN a periodic task is scheduled, THE Scheduler SHALL execute it at the configured interval (hours, minutes, or seconds)
3. WHEN an update task executes, THE Scheduler SHALL fetch all enabled subscriptions, parse M3U files, and aggregate channels
4. WHEN a test task executes, THE Scheduler SHALL test all channels and persist results
5. WHEN a task is manually triggered, THE Scheduler SHALL execute it immediately regardless of schedule
6. WHEN the system shuts down, THE Scheduler SHALL gracefully stop all running tasks

### Requirement 6: Channel Data Export

**User Story:** As a user, I want to export aggregated channels in standard formats, so that I can use them with IPTV players and other applications.

#### Acceptance Criteria

1. WHEN a user requests M3U export, THE Exporter SHALL generate a valid M3U file with all channel metadata
2. WHEN a user requests JSON export, THE Exporter SHALL generate a JSON file with complete channel data
3. WHEN export is requested with only_working=true, THE Exporter SHALL include only channels marked as 'online'
4. WHEN export completes, THE Exporter SHALL save the file to disk with a timestamped filename
5. WHEN a user requests the export list, THE Exporter SHALL return all previously generated export files with metadata
6. WHEN a user deletes an export file, THE Exporter SHALL remove it from disk

### Requirement 7: Web User Interface

**User Story:** As a user, I want to interact with the system through a web interface, so that I can manage subscriptions and view channel information without command-line access.

#### Acceptance Criteria

1. WHEN a user accesses the web interface, THE System SHALL display a dashboard with aggregation statistics (total subscriptions, channels, groups, online/offline counts)
2. WHEN a user navigates to the subscriptions page, THE System SHALL display all subscriptions with their status and channel counts
3. WHEN a user navigates to the channels page, THE System SHALL display all channels with filtering by group, status (online/offline/untested), and search by name
4. WHEN a user clicks to test a single channel, THE System SHALL execute the test and update the UI with results
5. WHEN a user clicks to test all channels, THE System SHALL initiate batch testing and provide progress feedback
6. WHEN a user clicks to update subscriptions, THE System SHALL initiate the update process and provide status feedback
7. WHEN a user navigates to the export page, THE System SHALL display available export files and allow creation of new exports
8. WHEN a user navigates to settings, THE System SHALL display all configurable parameters and allow modification

### Requirement 8: RESTful API Endpoints

**User Story:** As a developer, I want to access aggregated channels programmatically, so that I can integrate the system with IPTV players and other applications.

#### Acceptance Criteria

1. WHEN a client requests GET /api/playlist.m3u, THE API SHALL return a valid M3U file containing only online channels
2. WHEN a client requests GET /api/channels, THE API SHALL return a JSON array of all channels with optional filtering by group or status
3. WHEN a client requests GET /api/subscriptions, THE API SHALL return a JSON array of all subscriptions with their status
4. WHEN a client requests GET /api/stats, THE API SHALL return aggregation statistics (total channels, online count, offline count, etc.)
5. WHEN a client requests POST /api/subscriptions, THE API SHALL create a new subscription and return the created object
6. WHEN a client requests DELETE /api/subscriptions/{id}, THE API SHALL remove the subscription and return success status

### Requirement 9: Configuration Management

**User Story:** As a system administrator, I want to configure system parameters, so that I can customize behavior for different deployment scenarios.

#### Acceptance Criteria

1. WHEN the system starts, THE System SHALL load configuration from a config.json file if it exists
2. WHEN configuration is missing, THE System SHALL use sensible defaults and create the config file
3. WHEN a user modifies settings through the web UI, THE System SHALL persist changes to config.json
4. WHEN configuration is updated, THE System SHALL apply changes to running components (scheduler intervals, test timeouts, etc.)
5. THE System SHALL support configuration of: update interval, test interval, stream test timeout, max concurrent workers, deep check settings, aggregation matching strategy, and similarity threshold

### Requirement 10: Data Persistence

**User Story:** As a system component, I want to persist all data to disk, so that the system can recover state after restart and maintain historical data.

#### Acceptance Criteria

1. WHEN subscriptions are added, modified, or deleted, THE System SHALL persist changes to subscriptions.json
2. WHEN channels are aggregated or tested, THE System SHALL persist the channel list to channels.json
3. WHEN configuration is modified, THE System SHALL persist changes to config.json
4. WHEN the system starts, THE System SHALL load all persisted data from disk
5. WHEN data files are corrupted or missing, THE System SHALL initialize with empty data and log the issue
6. THE System SHALL maintain data consistency across concurrent operations

### Requirement 11: Performance and Concurrency

**User Story:** As a system operator, I want the system to handle concurrent operations efficiently, so that multiple users can interact with the system simultaneously without performance degradation.

#### Acceptance Criteria

1. WHEN multiple stream tests are executed, THE System SHALL process them concurrently up to the configured max_workers limit
2. WHEN the web UI requests data while background tasks are running, THE System SHALL serve requests without blocking
3. WHEN multiple users access the web interface simultaneously, THE System SHALL handle requests concurrently
4. WHEN aggregating channels from multiple sources, THE System SHALL fetch and parse sources concurrently
5. THE System SHALL complete testing of 1000 channels within 5 minutes with default configuration (10 concurrent workers, 5-second timeout)

### Requirement 12: Error Handling and Logging

**User Story:** As a system operator, I want comprehensive error handling and logging, so that I can diagnose issues and maintain system reliability.

#### Acceptance Criteria

1. WHEN a subscription fetch fails, THE System SHALL log the error, mark the subscription as failed, and continue processing other subscriptions
2. WHEN a stream test fails, THE System SHALL log the error and mark the stream as offline
3. WHEN configuration loading fails, THE System SHALL log the error and use default configuration
4. WHEN data persistence fails, THE System SHALL log the error and attempt retry
5. THE System SHALL maintain structured logs with timestamp, level, component, and message
6. WHEN an unhandled error occurs, THE System SHALL log it and continue operation without crashing

### Requirement 13: Backward Compatibility with Python Version

**User Story:** As a system operator, I want the Go version to maintain compatibility with existing data formats, so that I can migrate from Python to Go without data loss.

#### Acceptance Criteria

1. WHEN the Go system starts, THE System SHALL read subscriptions.json created by the Python version
2. WHEN the Go system starts, THE System SHALL read channels.json created by the Python version
3. WHEN the Go system starts, THE System SHALL read config.json created by the Python version
4. WHEN the Go system exports data, THE System SHALL generate files in the same format as the Python version
5. WHEN a user migrates from Python to Go, THE System SHALL preserve all subscription and channel data

