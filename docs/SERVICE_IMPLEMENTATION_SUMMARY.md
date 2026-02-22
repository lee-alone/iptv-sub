# Linux 系统服务管理实现总结

## 概述

已成功实现 IPTV Aggregator 的 Linux systemd 服务管理功能，允许用户将程序安装为系统服务，并通过标准的 systemctl 命令进行管理。

## 实现内容

### 1. 核心模块

#### `utils/service_linux.go`
- 仅在 Linux 平台编译（使用 `// +build linux`）
- 实现 `ServiceManager` 结构体，提供以下功能：
  - `Install()` - 安装服务
  - `Uninstall()` - 卸载服务
  - `Start()` - 启动服务
  - `Stop()` - 停止服务
  - `Restart()` - 重启服务
  - `Status()` - 查看服务状态
  - `HandleServiceCommand()` - 统一的命令处理入口

#### `utils/service_notsupported.go`
- 在非 Linux 平台编译（使用 `// +build !linux`）
- 提供友好的错误提示

### 2. 主程序修改

#### `main.go`
- 添加 `-s` 命令行参数用于服务管理
- 在 flag.Parse() 后优先处理服务命令
- 服务命令处理后立即退出，不启动主程序

### 3. 构建系统更新

#### `Makefile`
- 添加 `build-linux` 目标（别名为 `build-debian`）
- 更新 `build-all` 目标以包含 Linux 编译
- 更新帮助信息，显示服务管理命令

## 功能特性

### 安装流程

```bash
sudo ./iptv-aggregator -s install [-config /path/to/config.json]
```

安装时会：
1. 检查 root 权限
2. 创建 `/opt/iptv-aggregator` 目录
3. 复制可执行文件到安装目录
4. 复制配置文件（如指定）
5. 生成 systemd 服务文件
6. 重新加载 systemd 配置
7. 启用服务自启动

### 卸载流程

```bash
sudo ./iptv-aggregator -s uninstall
```

卸载时会：
1. 检查 root 权限
2. 停止正在运行的服务
3. 禁用服务自启动
4. 删除 systemd 服务文件
5. 重新加载 systemd 配置
6. 删除安装目录

### 服务控制

```bash
sudo ./iptv-aggregator -s start      # 启动
sudo ./iptv-aggregator -s stop       # 停止
sudo ./iptv-aggregator -s restart    # 重启
sudo ./iptv-aggregator -s status     # 查看状态
```

## 文件结构

```
项目根目录/
├── main.go                          # 修改：添加服务管理支持
├── utils/
│   ├── service_linux.go             # 新增：Linux 服务管理
│   └── service_notsupported.go      # 新增：非 Linux 平台占位符
├── Makefile                         # 修改：添加 build-linux 目标
└── docs/
    ├── SERVICE_MANAGEMENT.md        # 新增：详细使用文档
    ├── SERVICE_QUICK_REFERENCE.md   # 新增：快速参考
    └── SERVICE_IMPLEMENTATION_SUMMARY.md  # 本文件
```

## 生成的 systemd 服务文件

位置：`/etc/systemd/system/iptv-aggregator.service`

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

## 使用流程

### 第一次安装

```bash
# 1. 编译程序
make build-linux

# 2. 安装服务
sudo ./build/linux/iptv-aggregator -s install

# 3. 启动服务
sudo systemctl start iptv-aggregator

# 4. 查看状态
sudo systemctl status iptv-aggregator

# 5. 查看日志
sudo journalctl -u iptv-aggregator -f
```

### 日常管理

```bash
# 查看状态
sudo systemctl status iptv-aggregator

# 查看日志
sudo journalctl -u iptv-aggregator -f

# 重启服务
sudo systemctl restart iptv-aggregator

# 停止服务
sudo systemctl stop iptv-aggregator

# 启动服务
sudo systemctl start iptv-aggregator
```

### 卸载

```bash
sudo ./build/linux/iptv-aggregator -s uninstall
```

## 技术细节

### 平台特定编译

使用 Go 的 build tags 实现平台特定代码：

- `service_linux.go` - 仅在 Linux 平台编译
- `service_notsupported.go` - 在非 Linux 平台编译

这样可以在编译时自动选择正确的实现，避免运行时错误。

### 权限管理

- 所有服务管理操作都需要 root 权限
- 程序会检查当前用户是否为 root
- 如果不是 root，会显示友好的错误提示

### 日志处理

- 服务日志输出到 systemd journal
- 可以通过 `journalctl` 查看
- 支持实时跟踪、历史查询等功能

### 自动重启

- 服务配置了 `Restart=on-failure`
- 如果服务异常退出，会自动重启
- 重启间隔为 5 秒

## 安全考虑

1. **权限检查** - 所有操作都需要 root 权限
2. **文件权限** - 可执行文件设置为 755
3. **配置保护** - 配置文件在安装目录中
4. **日志审计** - 所有操作都记录在 systemd journal 中

## 兼容性

- ✅ Linux（systemd）
- ❌ Windows
- ❌ macOS
- ❌ 其他 Unix 系统（不支持 systemd）

## 测试建议

1. **安装测试**
   ```bash
   sudo ./build/linux/iptv-aggregator -s install
   sudo systemctl status iptv-aggregator
   ```

2. **启动/停止测试**
   ```bash
   sudo systemctl stop iptv-aggregator
   sudo systemctl start iptv-aggregator
   ```

3. **日志测试**
   ```bash
   sudo journalctl -u iptv-aggregator -n 50
   ```

4. **卸载测试**
   ```bash
   sudo ./build/linux/iptv-aggregator -s uninstall
   ls /opt/iptv-aggregator  # 应该不存在
   ```

## 后续改进建议

1. **用户权限** - 可以创建专用用户（如 `iptv:iptv`）来运行服务，而不是 root
2. **配置管理** - 支持从命令行参数覆盖配置文件中的设置
3. **监控集成** - 集成 Prometheus 或其他监控系统
4. **日志轮转** - 配置 logrotate 来管理日志文件
5. **健康检查** - 添加 systemd 的 ExecHealthCheck 功能
6. **多实例** - 支持运行多个实例（不同端口）

## 相关文档

- [SERVICE_MANAGEMENT.md](SERVICE_MANAGEMENT.md) - 详细使用文档
- [SERVICE_QUICK_REFERENCE.md](SERVICE_QUICK_REFERENCE.md) - 快速参考卡片
