# Linux 系统服务管理

本文档说明如何在 Linux 系统上将 IPTV Aggregator 安装为系统服务。

## 前置要求

- Linux 系统（支持 systemd）
- root 权限（使用 `sudo`）
- 已编译的可执行文件

## 安装目录结构

安装后的目录结构如下：

```
/opt/iptv-aggregator/
├── iptv-aggregator          # 主程序
├── config.json              # 配置文件
└── data/                    # 数据目录（如果指定）
```

systemd 服务文件位置：
```
/etc/systemd/system/iptv-aggregator.service
```

## 命令行选项

### 服务管理命令

```bash
-s install      # 安装为系统服务
-s uninstall    # 卸载系统服务
-s start        # 启动服务
-s stop         # 停止服务
-s restart      # 重启服务
-s status       # 查看服务状态
```

## 使用示例

### 1. 编译程序

```bash
# 使用 Makefile
make build-linux

# 或者使用 go build
go build -o build/iptv-aggregator
```

### 2. 安装服务

```bash
# 基本安装（使用默认配置）
sudo ./build/iptv-aggregator -s install

# 指定自定义配置文件
sudo ./build/iptv-aggregator -s install -config /path/to/custom-config.json
```

安装成功后会显示：
```
✓ Service installed successfully!
  Service file: /etc/systemd/system/iptv-aggregator.service
  Install path: /opt/iptv-aggregator

Next steps:
  Start service:   sudo systemctl start iptv-aggregator
  Check status:    sudo systemctl status iptv-aggregator
  View logs:       sudo journalctl -u iptv-aggregator -f
```

### 3. 启动服务

```bash
sudo systemctl start iptv-aggregator
```

### 4. 查看服务状态

```bash
sudo systemctl status iptv-aggregator
```

### 5. 查看实时日志

```bash
# 查看最近的日志
sudo journalctl -u iptv-aggregator -n 50

# 实时跟踪日志
sudo journalctl -u iptv-aggregator -f

# 查看特定时间范围的日志
sudo journalctl -u iptv-aggregator --since "2 hours ago"
```

### 6. 重启服务

```bash
sudo systemctl restart iptv-aggregator
```

### 7. 停止服务

```bash
sudo systemctl stop iptv-aggregator
```

### 8. 卸载服务

```bash
sudo ./build/iptv-aggregator -s uninstall
```

卸载会：
- 停止正在运行的服务
- 禁用服务自启动
- 删除 systemd 服务文件
- 删除 `/opt/iptv-aggregator` 目录

## 服务文件内容

安装后生成的 systemd 服务文件示例：

```ini
[Unit]
Description=IPTV M3U Aggregator Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/iptv-aggregator
ExecStart=/opt/iptv-aggregator/iptv-aggregator -config config.json
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

## 常见操作

### 设置开机自启

安装时已自动启用开机自启，如需手动启用：

```bash
sudo systemctl enable iptv-aggregator
```

### 禁用开机自启

```bash
sudo systemctl disable iptv-aggregator
```

### 查看服务是否启用

```bash
sudo systemctl is-enabled iptv-aggregator
```

### 查看服务是否运行

```bash
sudo systemctl is-active iptv-aggregator
```

### 重新加载 systemd 配置

如果手动修改了服务文件，需要重新加载：

```bash
sudo systemctl daemon-reload
```

## 故障排查

### 服务无法启动

1. 检查日志：
```bash
sudo journalctl -u iptv-aggregator -n 100
```

2. 检查配置文件是否存在：
```bash
ls -la /opt/iptv-aggregator/
```

3. 检查可执行文件权限：
```bash
ls -la /opt/iptv-aggregator/iptv-aggregator
```

### 权限问题

如果遇到权限问题，确保使用 `sudo`：

```bash
sudo ./build/iptv-aggregator -s install
```

### 端口被占用

如果 8080 端口被占用，可以在配置文件中修改端口，或使用命令行参数：

```bash
# 临时运行在不同端口
./build/iptv-aggregator -port 8081
```

## 配置文件管理

### 修改配置

1. 编辑配置文件：
```bash
sudo nano /opt/iptv-aggregator/config.json
```

2. 重启服务使配置生效：
```bash
sudo systemctl restart iptv-aggregator
```

### 备份配置

```bash
sudo cp /opt/iptv-aggregator/config.json /opt/iptv-aggregator/config.json.bak
```

## 性能监控

### 查看服务资源使用

```bash
# 使用 systemd-cgtop
systemd-cgtop

# 或使用 ps
ps aux | grep iptv-aggregator
```

### 查看服务内存使用

```bash
sudo systemctl status iptv-aggregator
```

## 安全建议

1. **定期备份配置和数据**：
```bash
sudo tar -czf iptv-aggregator-backup.tar.gz /opt/iptv-aggregator/
```

2. **定期检查日志**：
```bash
sudo journalctl -u iptv-aggregator --since "1 day ago" | grep -i error
```

3. **监控磁盘空间**：
```bash
df -h /opt/iptv-aggregator/
```

## 更新程序

1. 编译新版本
2. 停止服务：`sudo systemctl stop iptv-aggregator`
3. 备份旧程序：`sudo cp /opt/iptv-aggregator/iptv-aggregator /opt/iptv-aggregator/iptv-aggregator.bak`
4. 复制新程序：`sudo cp ./build/iptv-aggregator /opt/iptv-aggregator/`
5. 启动服务：`sudo systemctl start iptv-aggregator`
6. 验证：`sudo systemctl status iptv-aggregator`

## 卸载和清理

完全卸载服务：

```bash
# 1. 卸载服务
sudo ./build/iptv-aggregator -s uninstall

# 2. 清理日志（可选）
sudo journalctl --vacuum=time=30d
```

## 平台支持

- ✅ Linux（systemd）
- ❌ Windows（不支持）
- ❌ macOS（不支持）

在非 Linux 系统上使用 `-s` 参数会显示错误提示。
