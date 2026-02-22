# 架构改进建议

## 1. 数据持久化优化方案

### 选项 A: 读写分离 + 缓存
```
读取: 内存缓存 → 文件（仅在启动时）
写入: 内存缓存 → 定期异步写入文件
```

**优点：**
- 减少文件IO操作
- 保持JSON兼容性
- 实现简单

**实现示例：**

```go
type PersistentCache struct {
    data      interface{}
    mu        sync.RWMutex
    filepath  string
    saveTimer *time.Timer
    dirty     bool
}

func (pc *PersistentCache) Save() {
    pc.saveTimer.Stop()
    pc.saveTimer = time.AfterFunc(5*time.Second, pc.doSave)
}

func (pc *PersistentCache) doSave() {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    // 原子写入：先写临时文件，再重命名
    tmpFile := pc.filepath + ".tmp"
    data, _ := json.Marshal(pc.data)
    ioutil.WriteFile(tmpFile, data, 0644)
    os.Rename(tmpFile, pc.filepath)
}
```

### 选项 B: 引入嵌入式数据库（可选）

对于未来的扩展，可以考虑：
- **boltdb** - 纯Go, 单文件, 类似LMDB
- **badger** - 基于LSM树的KV存储
- **sqlite** - 成熟稳定, 支持复杂查询

```go
// 使用boltDB作为存储层
type BoltDBStore struct {
    db *bolt.DB
}

// 保留JSON import/export功能
func (s *BoltDBStore) ExportJSON() ([]byte, error) {
    // ...
}
```

## 2. 并发控制改进

### 当前问题
- 流测试并发度固定
- 不支持动态调整并发数

### 建议：自适应工作池

```go
type AdaptiveWorkerPool struct {
    minWorkers  int
    maxWorkers  int
    activeJobs  int
    workers     chan *worker
    jobQueue    chan Job
    resultQueue chan Result
    
    // 自适应调节
    cpuUsage    float64
    memoryUsage uint64
}

func (p *AdaptiveWorkerPool) adjustWorkers() {
    // 根据系统负载动态调整worker数量
    if p.cpuUsage < 0.7 && p.memoryUsage < 80<<20 {
        // 增加worker
        p.scaleUp()
    } else if p.cpuUsage > 0.9 || p.memoryUsage > 90<<20 {
        // 减少worker
        p.scaleDown()
    }
}
```

## 3. 错误处理增强

### 建议：结构化错误 + 重试机制

```go
type RetryableError struct {
    Err       error
    CanRetry  bool
    Delay     time.Duration
    MaxTries  int
}

func StreamTestWithRetry(url string, config TestConfig) error {
    var lastErr error
    for i := 0; i < config.MaxTries; i++ {
        err := TestStream(url, config.Timeout)
        if err == nil {
            return nil
        }
        
        if IsNetworkError(err) {
            lastErr = &RetryableError{Err: err, CanRetry: true}
            time.Sleep(config.BackoffDelay)
            continue
        }
        
        return err
    }
    return lastErr
}
```

## 4. 监控和可观测性

### 建议：添加 Prometheus 指标

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    streamTestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "stream_test_duration_seconds",
        Help:    "Duration of stream tests",
        Buckets: prometheus.DefBuckets,
    })
    
    channelsOnline = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "channels_online_total",
        Help: "Number of online channels",
    })
    
    subscriptionUpdateStatus = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "subscription_updates_total",
            Help: "Total subscription updates",
        },
        []string{"status"}, // success, failed
    )
)

// 暴露指标端点
app.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

## 5. 增量聚合策略

### 当前问题
- 每次更新都清空所有频道
- 频测试结果的重复工作

### 建议：增量更新

```go
type IncrementalAggregator struct {
    existingChannels map[string]*Channel
    newChannels      map[string]*Channel
    changedChannels  map[string]*Channel
}

func (ia *IncrementalAggregator) Aggregate(newChannels []*Channel) {
    // 只更新变化的频道
    for _, newCh := range newChannels {
        if existing, ok := ia.existingChannels[newCh.ID]; ok {
            // 频道已存在，检查是否需要更新
            if !existing.Equal(newCh) {
                ia.changedChannels[newCh.ID] = newCh
            }
        } else {
            // 新频道
            ia.newChannels[newCh.ID] = newCh
        }
    }
    
    // 只测试变化的频道
    channelsToTest := append(
        maps.Values(ia.newChannels),
        maps.Values(ia.changedChannels)...,
    )
}
```

## 6. 测试策略增强

### 建议：添加模糊测试和压力测试

```go
// 模糊测试：用随机输入测试解析器
func FuzzParseM3U(f *testing.F) {
    // 已知良好种子
    f.Add("#EXTM3U\n#EXTINF:-1 tvg-id=\"1\",Channel\nhttp://test.com/stream")
    
    f.Fuzz(func(t *testing.T, input string) {
        // 解析器应该能优雅处理任何输入而不panic
        channels, err := ParseM3U(input)
        if err != nil {
            // 错误是可接受的，但必须不panic
            return
        }
        // 验证输出结构的完整性
        for _, ch := range channels {
            if ch.Name == "" && ch.URL == "" {
                t.Errorf("Invalid channel: both name and url are empty")
            }
        }
    })
}

// 压力测试：模拟大量并发请求
func BenchmarkConcurrentStreamTests(b *testing.B) {
    urls := generateTestURLs(1000)
    pool := NewWorkerPool(50)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        pool.TestChannels(urls)
    }
}
```

## 7. Docker 容器化增强

### 多阶段构建优化

```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o iptv-aggregator

# 运行阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# 创建非root用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

COPY --from=builder /app/iptv-aggregator .
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates
RUN mkdir -p data && chown -R appuser:appuser /app

USER appuser
EXPOSE 8080
CMD ["./iptv-aggregator"]
```

### Docker Compose 配置

```yaml
version: '3.8'

services:
  iptv-aggregator:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.json:/app/config.json:ro
    environment:
      - TZ=Asia/Shanghai
      - GIN_MODE=release
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/api/stats"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## 8. 配置管理改进

### 建议：支持配置热重载

```go
type ConfigWatcher struct {
    config     *Config
    filePath   string
    notifyChan chan struct{}
    watcher    *fsnotify.Watcher
}

func (cw *ConfigWatcher) Watch() {
    for {
        select {
        case event := <-cw.watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                cw.reloadConfig()
                cw.notifyChan <- struct{}{}
            }
        }
    }
}

// 支持监听配置变化并实时更新
func main() {
    configWatcher := NewConfigWatcher("config.json")
    go configWatcher.Watch()
    
    for range configWatcher.Notify() {
        // 配置已更新，应用新配置
        applyNewConfig(configWatcher.Config())
    }
}
```

## 9. 安全性增强

### 建议：添加认证和限流

```go
// JWT 中间件（可选）
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !validateToken(token) {
            c.AbortWithStatus(401)
            return
        }
        c.Next()
    }
}

// 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
    limiter := rate.NewLimiter(10, 100) // 每秒10个请求，桶大小100
    
    return func(c *gin.Context) {
        if !limiter.Allow() {
            c.AbortWithStatus(429)
            return
        }
        c.Next()
    }
}

// 只对关键API应用这些中间件
api := router.Group("/api")
api.Use(RateLimitMiddleware())
api.Use(AuthMiddleware())
```

## 10. API 版本管理

### 建议：添加API版本化

```go
// 支持多个API版本
v1 := router.Group("/api/v1")
{
    v1.GET("/channels", handlers.GetChannelsV1)
    v1.POST("/subscriptions", handlers.AddSubscriptionV1)
}

v2 := router.Group("/api/v2")
{
    v2.GET("/channels", handlers.GetChannelsV2)  // 包含更多字段
    v2.POST("/subscriptions", handlers.AddSubscriptionV2)
}

// 当前版本别名
router.Any("/api/*path", func(c *gin.Context) {
    c.Redirect(301, "/api/v1"+c.Param("path"))
})
```

## 实施优先级

### 高优先级（必须实现）
1. ✅ 数据持久化优化（读写分离 + 缓存）
2. ✅ 监控指标（Prometheus）
3. ✅ 自适应工作池

### 中优先级（建议实现）
4. ✅ 增量聚合策略
5. ✅ 配置热重载
6. ✅ 安全性增强

### 低优先级（可选）
7. ⚪ API版本化
8. ⚪ 嵌入式数据库支持
9. ⚪ 测试策略增强

## 总结

这些改进建议都是在保持原始功能完整性的前提下，针对性能、可靠性和可维护性的增强。建议在实施时：

1. **先完成基础功能**（按照原计划）
2. **然后逐步添加优化**（优先级从高到低）
3. **每个改进都要有测试**
4. **保持向后兼容性**

这样可以确保项目稳定、高效地迁移到 Go。