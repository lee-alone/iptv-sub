# 聚合优化测试指南

## 测试场景

### 场景1：URL去重验证

**目标：** 验证AddURL的O(1)性能

**测试代码：**
```go
func TestURLDeduplication(t *testing.T) {
    ch := models.NewChannel("CCTV-1", "News", "123", "", "", "http://url1.com", "source1")
    
    // 添加相同URL
    ch.AddURL("http://url1.com", "source1")
    if len(ch.URLs) != 1 {
        t.Errorf("Expected 1 URL, got %d", len(ch.URLs))
    }
    
    // 添加不同URL
    ch.AddURL("http://url2.com", "source2")
    if len(ch.URLs) != 2 {
        t.Errorf("Expected 2 URLs, got %d", len(ch.URLs))
    }
}
```

**预期结果：** 重复URL被正确过滤

---

### 场景2：tvg-id索引验证

**目标：** 验证tvg-id索引的O(1)查找

**测试代码：**
```go
func TestTvgIDIndexing(t *testing.T) {
    agg := services.NewChannelAggregator("./test_data")
    
    // 创建测试频道
    ch1 := models.NewChannel("CCTV-1", "News", "cctv1", "", "", "http://url1.com", "source1")
    ch2 := models.NewChannel("CCTV-2", "News", "cctv2", "", "", "http://url2.com", "source2")
    
    channels := []*models.Channel{ch1, ch2}
    agg.AggregateChannels(channels, "tvg_id", 0.0)
    
    // 验证索引
    found := agg.GetChannelByID("cctv1")
    if found == nil || found.Name != "CCTV-1" {
        t.Errorf("tvg-id index lookup failed")
    }
}
```

**预期结果：** 快速查找到正确的频道

---

### 场景3：名称相似度匹配

**目标：** 验证名称匹配的多层优化

**测试代码：**
```go
func TestNameSimilarityMatching(t *testing.T) {
    agg := services.NewChannelAggregator("./test_data")
    
    // 创建现有频道
    existing := models.NewChannel("CCTV-1", "News", "", "", "", "http://url1.com", "source1")
    agg.channels["ch1"] = existing
    agg.addToIndexes(existing)
    
    // 创建新频道（名称略有不同）
    new := models.NewChannel("CCTV-1 HD", "News", "", "", "", "http://url2.com", "source2")
    
    channels := []*models.Channel{new}
    added, updated, _, _ := agg.AggregateChannels(channels, "name", 0.75)
    
    if updated != 1 {
        t.Errorf("Expected 1 update, got %d", updated)
    }
}
```

**预期结果：** 相似名称被正确匹配

---

### 场景4：智能合并策略

**目标：** 验证测试结果的智能保留

**测试代码：**
```go
func TestSmartMerging(t *testing.T) {
    // 现有频道：不可用
    existing := models.NewChannel("CCTV-1", "News", "123", "", "", "http://url1.com", "source1")
    existing.UpdateTestResult("offline", "", 0)
    
    // 新频道：可用
    new := models.NewChannel("CCTV-1", "News", "123", "", "", "http://url2.com", "source2")
    new.UpdateTestResult("online", "http://url2.com", 100)
    
    agg := services.NewChannelAggregator("./test_data")
    agg.mergeChannels(existing, new)
    
    // 验证：应该保留新频道的可用状态
    if existing.TestResults.Status != "online" {
        t.Errorf("Expected online status, got %s", existing.TestResults.Status)
    }
    if existing.TestResults.WorkingURL != "http://url2.com" {
        t.Errorf("Expected working URL to be updated")
    }
}
```

**预期结果：** 合并后保留最佳的测试结果

---

### 场景5：性能基准测试

**目标：** 验证性能提升

**测试代码：**
```go
func BenchmarkAggregation(b *testing.B) {
    agg := services.NewChannelAggregator("./test_data")
    
    // 创建10000个现有频道
    for i := 0; i < 10000; i++ {
        ch := models.NewChannel(
            fmt.Sprintf("Channel-%d", i),
            "Group",
            fmt.Sprintf("id-%d", i),
            "",
            "",
            fmt.Sprintf("http://url%d.com", i),
            "source",
        )
        agg.channels[ch.ID] = ch
        agg.addToIndexes(ch)
    }
    
    // 创建1000个新频道
    newChannels := make([]*models.Channel, 1000)
    for i := 0; i < 1000; i++ {
        newChannels[i] = models.NewChannel(
            fmt.Sprintf("Channel-%d", 10000+i),
            "Group",
            fmt.Sprintf("id-%d", 10000+i),
            "",
            "",
            fmt.Sprintf("http://url%d.com", 10000+i),
            "source",
        )
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        agg.AggregateChannels(newChannels, "tvg_id", 0.0)
    }
}
```

**预期结果：** 
- tvg-id匹配：< 50ms
- 名称匹配：< 500ms

---

## 手动测试步骤

### 1. 准备测试数据

创建两个M3U文件：

**source1.m3u:**
```
#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV-1" group-title="News",CCTV-1
http://stream1.com/cctv1
#EXTINF:-1 tvg-id="cctv2" tvg-name="CCTV-2" group-title="News",CCTV-2
http://stream1.com/cctv2
```

**source2.m3u:**
```
#EXTM3U
#EXTINF:-1 tvg-id="cctv1" tvg-name="CCTV-1 HD" group-title="News",CCTV-1 HD
http://stream2.com/cctv1
#EXTINF:-1 tvg-id="cctv3" tvg-name="CCTV-3" group-title="Entertainment",CCTV-3
http://stream2.com/cctv3
```

### 2. 测试tvg-id匹配

```bash
# 配置：match_by = "tvg_id"
# 预期：3个频道（cctv1, cctv2, cctv3）
# 实际URL数：cctv1有2个URL（合并）
```

### 3. 测试名称匹配

```bash
# 配置：match_by = "name", threshold = 0.7
# 预期：3个频道
# 实际URL数：CCTV-1和CCTV-1 HD应该合并
```

### 4. 测试性能

```bash
# 使用大量源（100+）
# 监控聚合时间
# 预期：< 1秒完成
```

---

## 性能指标

### 目标指标

| 操作 | 目标 | 实际 | 状态 |
|------|------|------|------|
| tvg-id匹配 (1000新+10000现有) | < 50ms | ? | ⏳ |
| 名称匹配 (1000新+10000现有) | < 500ms | ? | ⏳ |
| URL去重 (1000个URL) | < 10ms | ? | ⏳ |
| 总聚合时间 | < 1s | ? | ⏳ |

### 监控指标

- 聚合时间
- 内存使用
- 索引大小
- 重复率

---

## 故障排查

### 问题1：聚合速度未达预期

**检查项：**
1. 是否使用了tvg-id匹配？
2. 索引是否正确初始化？
3. 是否有大量名称相似度计算？

**解决方案：**
```go
// 启用日志
fmt.Printf("Aggregating %d channels\n", len(newChannels))
fmt.Printf("tvgIDIndex size: %d\n", len(ca.tvgIDIndex))
fmt.Printf("nameIndex size: %d\n", len(ca.nameIndex))
```

---

### 问题2：频道重复

**检查项：**
1. 相似度阈值是否过低？
2. 是否有多个相同tvg-id？

**解决方案：**
```go
// 检查重复
duplicates := make(map[string]int)
for _, ch := range agg.GetAllChannels() {
    duplicates[ch.TvgID]++
}
for id, count := range duplicates {
    if count > 1 {
        fmt.Printf("Duplicate tvg-id: %s (%d times)\n", id, count)
    }
}
```

---

## 持续监控

### 建议的监控指标

1. **聚合性能**
   - 平均聚合时间
   - P95/P99聚合时间
   - 内存峰值

2. **数据质量**
   - 重复率
   - 可用URL比例
   - 测试通过率

3. **索引效率**
   - 索引命中率
   - 索引大小
   - 索引重建时间
