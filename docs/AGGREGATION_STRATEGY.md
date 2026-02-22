# 聚合策略指南

## 概述

本文档说明如何使用优化后的聚合逻辑来处理不同场景的IPTV源。

## 聚合流程

```
订阅源 → 解析 → 同源URL去重 → 频道聚合 → 测试 → 导出
         (Parser)  (AddURL)    (Aggregate) (Test) (Export)
```

## 匹配策略

### 1. tvg-id 匹配（推荐用于高质量源）

**特点：** 精确匹配，O(1)性能

**适用场景：**
- 来自官方或高质量源的频道
- tvg-id完整且准确的源

**配置：**
```json
{
  "match_by": "tvg_id",
  "similarity_threshold": 0.0
}
```

**性能：** 1000个新频道 + 10000个现有频道 = ~1ms

---

### 2. 名称匹配（推荐用于多源聚合）

**特点：** 支持模糊匹配，自动过滤

**适用场景：**
- 不同源的同一频道名称略有差异
- tvg-id不可靠或缺失

**配置：**
```json
{
  "match_by": "name",
  "similarity_threshold": 0.7
}
```

**阈值建议：**
- 0.9+：严格匹配（如"CCTV-1"和"CCTV-1 HD"不匹配）
- 0.7-0.8：平衡（推荐）
- 0.5-0.7：宽松（可能误匹配）

**性能：** 
- 精确匹配：O(1)
- 模糊匹配：O(n*m*l²)，但通过快速过滤减少90%计算

---

### 3. 双重匹配（推荐用于混合源）

**特点：** 优先tvg-id，备选名称匹配

**适用场景：**
- 混合多个质量不同的源
- 需要最大化匹配准确性

**配置：**
```json
{
  "match_by": "both",
  "similarity_threshold": 0.7
}
```

**匹配顺序：**
1. 先尝试tvg-id精确匹配 → O(1)
2. 失败则尝试名称相似度匹配 → O(n*m*l²)

---

## 聚合结果优化

### 智能合并策略

聚合时自动选择最佳的测试结果：

```
场景1：新频道可用 + 现有频道不可用
  → 更新为新频道的测试结果

场景2：两者都可用
  → 保留响应时间更快的

场景3：两者都不可用
  → 保留现有的测试结果

场景4：新频道未测试
  → 保留现有的测试结果
```

### 备用URL机制

对于保守模式（未来实现），可以保留多个URL：

```go
// 主URL（已测试为可用）
channel.URLs = []string{"http://stream1.com/cctv1"}

// 备用URL（备选方案）
channel.BackupURLs = []string{"http://stream2.com/cctv1"}
```

---

## 实际应用示例

### 示例1：单一高质量源

```json
{
  "subscriptions": [
    {
      "url": "http://example.com/iptv.m3u",
      "name": "Official IPTV",
      "enabled": true
    }
  ],
  "aggregation": {
    "match_by": "tvg_id",
    "similarity_threshold": 0.0
  }
}
```

**预期结果：**
- 聚合速度：极快（O(1)）
- 准确性：极高（精确匹配）
- 重复率：低

---

### 示例2：多个社区源

```json
{
  "subscriptions": [
    {
      "url": "http://source1.com/iptv.m3u",
      "name": "Community Source 1",
      "enabled": true
    },
    {
      "url": "http://source2.com/iptv.m3u",
      "name": "Community Source 2",
      "enabled": true
    }
  ],
  "aggregation": {
    "match_by": "both",
    "similarity_threshold": 0.75
  }
}
```

**预期结果：**
- 聚合速度：快（混合O(1)和O(n*m*l²)）
- 准确性：高（双重验证）
- 重复率：中等

---

### 示例3：混合源（官方+社区）

```json
{
  "subscriptions": [
    {
      "url": "http://official.com/iptv.m3u",
      "name": "Official",
      "enabled": true
    },
    {
      "url": "http://community.com/iptv.m3u",
      "name": "Community",
      "enabled": true
    }
  ],
  "aggregation": {
    "match_by": "both",
    "similarity_threshold": 0.8
  }
}
```

**预期结果：**
- 聚合速度：快
- 准确性：高
- 重复率：低

---

## 性能优化建议

### 1. 索引利用

系统自动维护三个索引：
- `tvgIDIndex`：O(1)查找
- `nameIndex`：O(1)精确匹配
- `urlSet`：O(1)URL去重

**无需手动配置，自动生效**

### 2. 快速过滤

名称匹配时自动应用：
- 长度差异检查（避免"A"和"ABCDEFGH"匹配）
- 首字母检查（高阈值时）

**无需手动配置，自动生效**

### 3. 增量更新

每次聚合只处理新频道，不重新处理现有频道。

**建议：** 定期增量更新而不是全量重新聚合

---

## 故障排查

### 问题1：频道重复率高

**原因：** 相似度阈值过低

**解决方案：**
```json
{
  "similarity_threshold": 0.8  // 提高阈值
}
```

---

### 问题2：频道丢失

**原因：** 相似度阈值过高

**解决方案：**
```json
{
  "similarity_threshold": 0.7  // 降低阈值
}
```

---

### 问题3：聚合速度慢

**原因：** 使用名称匹配处理大量源

**解决方案：**
1. 优先使用tvg-id匹配
2. 分批处理源（先聚合官方源，再聚合社区源）
3. 增加测试并发数

---

## 最佳实践

1. **优先使用tvg-id** - 性能最好，准确性最高
2. **混合源时使用"both"** - 平衡性能和准确性
3. **定期测试** - 保持测试结果最新
4. **监控重复率** - 调整相似度阈值
5. **增量更新** - 避免全量重新聚合

---

## 未来扩展

### 保守模式（计划中）

```json
{
  "aggregation": {
    "mode": "conservative",  // 保留多个URL
    "save_backups": true     // 在M3U中包含备用URL
  }
}
```

### 自定义匹配规则（计划中）

```json
{
  "aggregation": {
    "custom_rules": [
      {
        "pattern": "CCTV-*",
        "match_by": "tvg_id"
      },
      {
        "pattern": "*",
        "match_by": "name",
        "threshold": 0.7
      }
    ]
  }
}
```
