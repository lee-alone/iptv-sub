# Debian 兼容性说明

## 概述

本文档说明 IPTV Aggregator 的 Linux 服务管理功能在 Debian 系统上的兼容性和实现细节。

## 兼容性确认

### ✅ 完全支持的系统

- Debian 10 (Buster) 及以上
- Ubuntu 18.04 LTS 及以上
- 其他基于 systemd 的 Linux 发行版

### ✅ 支持的功能

| 功能 | 状态 | 说明 |
|------|------|------|
| 安装服务 | ✅ | 使用 systemd 安装 |
| 启动服务 | ✅ | 使用 systemctl start |
| 停止服务 | ✅ | 使用 systemctl stop |
| 重启服务 | ✅ | 使用 systemctl restart |
| 查看状态 | ✅ | 使用 systemctl status |
| 查看日志 | ✅ | 使用 journalctl |
| 开机自启 | ✅ | 自动启用 |
| 卸载服务 | ✅ | 完全清理 |

## 实现细节

### 1. 权限管理

**要求**: root 权限

```bash
# 正确方式
sudo ./iptv-aggregator -s install

# 错误方式（会被拒绝）
./iptv-aggregator -s install
```

**实现**:
- 使用 `os/user` 包检查当前用户 UID
- 如果 UID 不为 0，返回错误提示
- 错误消息明确指导用户使用 `sudo`

### 2. 文件系统操作

**安装路径**: `/opt/iptv-aggregator`

```
/opt/iptv-aggregator/
├── iptv-aggregator          # 可执行文件（755 权限）
├── config.json              # 配置文件（644 权限）
└── data/                    # 数据目录（如果存在）
```

**实现**:
- 使用 `os.MkdirAll()` 创建目录
- 使用 `os.Chmod()` 设置正确的文件权限
- 使用 `os.WriteFile()` 创建服务文件

### 3. systemd 集成

**服务文件位置**: `/etc/systemd/system/iptv-aggregator.service`

**关键配置**:

```ini
[Unit]
Description=IPTV M3U Aggregator Service
After=network.target              # 在网络启动后启动

[Service]
Type=simple                       # 简单服务类型
User=root                         # 以 root 用户运行
WorkingDirectory=/opt/iptv-aggregator
ExecStart=/opt/iptv-aggregator/iptv-aggregator -config config.json
Restart=on-failure                # 失败时自动重启
RestartSec=5                      # 重启延迟 5 秒
StandardOutput=journal            # 输出到 systemd journal
StandardError=journal             # 错误输出到 systemd journal

[Install]
WantedBy=multi-user.target        # 多用户模式下启动
```

**实现**:
- 使用 `exec.Command()` 执行 systemctl 命令
- 使用 `os.WriteFile()` 创建服务文件
- 自动调用 `systemctl daemon-reload` 重新加载配置

### 4. 配置文件处理

**改进点**:

1. **自动复制配置文件**
   - 如果指定了自定义配置，复制到安装目录
   - 如果没有指定，尝试复制当前目录的 `config.json`
   - 如果都不存在，显示警告但继续安装

2. **配置路径管理**
   - 服务文件中的 `ExecStart` 使用相对路径
   - 配置文件始终在 `/opt/iptv-aggregator/` 目录中
   - 支持自定义配置文件名

**实现**:

```go
// 如果指定了自定义配置
if sm.configPath != "" && sm.configPath != "config.json" {
    configFile := filepath.Base(sm.configPath)
    destConfig := filepath.Join(ServiceDir, configFile)
    copyFile(sm.configPath, destConfig)
} else {
    // 尝试复制默认配置
    if _, err := os.Stat("config.json"); err == nil {
        copyFile("config.json", filepath.Join(ServiceDir, "config.json"))
    }
}
```

### 5. 日志管理

**日志输出**: systemd journal

```bash
# 查看实时日志
sudo journalctl -u iptv-aggregator -f

# 查看最近 50 行
sudo journalctl -u iptv-aggregator -n 50

# 查看特定时间范围
sudo journalctl -u iptv-aggregator --since "1 hour ago"

# 查看错误日志
sudo journalctl -u iptv-aggregator -p err
```

**优势**:
- 集中管理所有系统日志
- 自动日志轮转
- 支持高级查询和过滤
- 与系统监控工具集成

## 测试验证

### 快速验证

```bash
# 1. 编译
make build-linux

# 2. 安装
cd build/linux
sudo ./iptv-aggregator -s install

# 3. 验证
sudo systemctl status iptv-aggregator

# 4. 查看日志
sudo journalctl -u iptv-aggregator -n 20

# 5. 卸载
sudo ./iptv-aggregator -s uninstall
```

### 详细测试

参考 [DEBIAN_TESTING_CHECKLIST.md](DEBIAN_TESTING_CHECKLIST.md)

## 常见问题

### Q1: 为什么需要 root 权限？

**A**: 因为需要：
- 创建 `/opt` 目录
- 写入 `/etc/systemd/system/` 目录
- 执行 `systemctl` 命令
- 管理系统服务

这些操作都需要 root 权限。

### Q2: 能否以非 root 用户运行服务？

**A**: 可以，但需要修改服务文件：

```ini
[Service]
User=iptv-user    # 改为非 root 用户
```

但这需要：
1. 创建专用用户
2. 设置正确的文件权限
3. 确保用户有权访问数据目录

### Q3: 如何修改服务配置？

**A**: 编辑服务文件后需要重新加载：

```bash
# 编辑服务文件
sudo nano /etc/systemd/system/iptv-aggregator.service

# 重新加载配置
sudo systemctl daemon-reload

# 重启服务
sudo systemctl restart iptv-aggregator
```

### Q4: 如何查看服务启动失败的原因？

**A**: 查看详细日志：

```bash
# 查看最近的日志
sudo journalctl -u iptv-aggregator -n 100

# 查看错误日志
sudo journalctl -u iptv-aggregator -p err

# 手动运行程序测试
/opt/iptv-aggregator/iptv-aggregator -config /opt/iptv-aggregator/config.json
```

### Q5: 能否在多个端口上运行多个实例？

**A**: 可以，需要：
1. 创建多个服务文件（如 `iptv-aggregator-8081.service`）
2. 在每个服务文件中指定不同的端口
3. 分别管理每个实例

## 性能考虑

### 内存使用

- 基础内存占用: ~50-100 MB
- 取决于频道数量和测试配置

### CPU 使用

- 空闲时: < 1%
- 测试时: 取决于并发数和网络速度

### 磁盘使用

- 程序: ~20 MB
- 配置和数据: 取决于频道数量

## 安全建议

### 1. 定期备份

```bash
sudo tar -czf iptv-aggregator-backup.tar.gz /opt/iptv-aggregator/
```

### 2. 监控日志

```bash
# 定期检查错误
sudo journalctl -u iptv-aggregator -p err --since "1 day ago"
```

### 3. 限制访问

```bash
# 检查文件权限
ls -la /opt/iptv-aggregator/

# 限制配置文件访问
sudo chmod 600 /opt/iptv-aggregator/config.json
```

### 4. 定期更新

```bash
# 停止服务
sudo systemctl stop iptv-aggregator

# 更新程序
sudo cp new-iptv-aggregator /opt/iptv-aggregator/iptv-aggregator

# 启动服务
sudo systemctl start iptv-aggregator
```

## 故障恢复

### 服务无法启动

```bash
# 1. 检查日志
sudo journalctl -u iptv-aggregator -n 50

# 2. 检查配置文件
cat /opt/iptv-aggregator/config.json

# 3. 手动测试
/opt/iptv-aggregator/iptv-aggregator -config /opt/iptv-aggregator/config.json

# 4. 检查端口占用
sudo netstat -tlnp | grep 8080
```

### 服务崩溃

```bash
# 1. 查看崩溃日志
sudo journalctl -u iptv-aggregator -n 100

# 2. 检查磁盘空间
df -h /opt/iptv-aggregator/

# 3. 检查内存
free -h

# 4. 重启服务
sudo systemctl restart iptv-aggregator
```

## 相关文档

- [SERVICE_MANAGEMENT.md](SERVICE_MANAGEMENT.md) - 详细使用文档
- [SERVICE_QUICK_REFERENCE.md](SERVICE_QUICK_REFERENCE.md) - 快速参考
- [DEBIAN_TESTING_CHECKLIST.md](DEBIAN_TESTING_CHECKLIST.md) - 测试清单
