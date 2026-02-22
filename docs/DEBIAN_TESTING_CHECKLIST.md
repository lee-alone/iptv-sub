# Debian 系统测试清单

本文档用于在 Debian/Ubuntu 系统上验证服务安装功能是否正常工作。

## 前置条件

- Debian 10+ 或 Ubuntu 18.04+
- root 权限（使用 `sudo`）
- Go 1.16+ 或预编译的二进制文件

## 测试环境准备

### 1. 编译程序

```bash
# 克隆或进入项目目录
cd /path/to/iptv-aggregator

# 编译 Linux 版本
make build-linux

# 验证编译成功
ls -la build/linux/iptv-aggregator
```

### 2. 验证编译环境

```bash
# 检查 systemd 是否可用
systemctl --version

# 检查 journalctl 是否可用
journalctl --version

# 检查当前用户
whoami

# 检查 /opt 目录是否存在
ls -ld /opt
```

## 测试步骤

### 测试 1: 基本安装

**目标**: 验证 `-s install` 命令能否成功安装服务

```bash
# 1. 进入编译输出目录
cd build/linux

# 2. 执行安装命令
sudo ./iptv-aggregator -s install

# 3. 验证安装结果
# 应该看到类似输出：
# ✓ Service installed successfully!
#   Service file: /etc/systemd/system/iptv-aggregator.service
#   Install path: /opt/iptv-aggregator
```

**验证清单**:
- [ ] 命令执行成功（退出码为 0）
- [ ] 显示安装成功消息
- [ ] `/opt/iptv-aggregator` 目录已创建
- [ ] `/etc/systemd/system/iptv-aggregator.service` 文件已创建

### 测试 2: 验证安装文件

**目标**: 确认所有必要的文件都已正确复制

```bash
# 1. 检查安装目录
ls -la /opt/iptv-aggregator/

# 应该看到：
# -rwxr-xr-x iptv-aggregator
# -rw-r--r-- config.json (如果存在)

# 2. 检查可执行文件权限
file /opt/iptv-aggregator/iptv-aggregator

# 应该显示：ELF 64-bit LSB executable

# 3. 检查服务文件
cat /etc/systemd/system/iptv-aggregator.service

# 应该看到 [Unit], [Service], [Install] 部分
```

**验证清单**:
- [ ] 可执行文件存在且权限为 755
- [ ] 配置文件存在（如果指定了）
- [ ] 服务文件格式正确
- [ ] 服务文件包含正确的 ExecStart 路径

### 测试 3: 启动服务

**目标**: 验证服务能否正常启动

```bash
# 1. 启动服务
sudo systemctl start iptv-aggregator

# 2. 检查服务状态
sudo systemctl status iptv-aggregator

# 应该显示：
# ● iptv-aggregator.service - IPTV M3U Aggregator Service
#    Loaded: loaded (/etc/systemd/system/iptv-aggregator.service; enabled; vendor preset: enabled)
#    Active: active (running) since ...

# 3. 验证进程是否运行
ps aux | grep iptv-aggregator

# 应该看到运行中的进程
```

**验证清单**:
- [ ] 服务启动成功
- [ ] 服务状态显示 "active (running)"
- [ ] 进程在进程列表中可见
- [ ] 没有错误消息

### 测试 4: 查看日志

**目标**: 验证日志系统正常工作

```bash
# 1. 查看最近的日志
sudo journalctl -u iptv-aggregator -n 50

# 2. 实时跟踪日志
sudo journalctl -u iptv-aggregator -f

# 应该看到服务启动的日志信息
```

**验证清单**:
- [ ] 日志能够正常查看
- [ ] 日志包含启动信息
- [ ] 没有错误或警告信息

### 测试 5: 停止服务

**目标**: 验证服务能否正常停止

```bash
# 1. 停止服务
sudo systemctl stop iptv-aggregator

# 2. 检查服务状态
sudo systemctl status iptv-aggregator

# 应该显示：
# ● iptv-aggregator.service - IPTV M3U Aggregator Service
#    Loaded: loaded (/etc/systemd/system/iptv-aggregator.service; enabled; vendor preset: enabled)
#    Active: inactive (dead) since ...

# 3. 验证进程已停止
ps aux | grep iptv-aggregator

# 应该只看到 grep 命令本身
```

**验证清单**:
- [ ] 服务停止成功
- [ ] 服务状态显示 "inactive (dead)"
- [ ] 进程已从进程列表中移除

### 测试 6: 重启服务

**目标**: 验证服务能否正常重启

```bash
# 1. 重启服务
sudo systemctl restart iptv-aggregator

# 2. 检查服务状态
sudo systemctl status iptv-aggregator

# 应该显示 "active (running)"

# 3. 查看重启日志
sudo journalctl -u iptv-aggregator -n 20
```

**验证清单**:
- [ ] 服务重启成功
- [ ] 服务状态显示 "active (running)"
- [ ] 日志中有重启记录

### 测试 7: 开机自启

**目标**: 验证服务已启用开机自启

```bash
# 1. 检查服务是否启用
sudo systemctl is-enabled iptv-aggregator

# 应该输出：enabled

# 2. 查看服务文件中的 [Install] 部分
grep -A 2 "\[Install\]" /etc/systemd/system/iptv-aggregator.service

# 应该显示：
# [Install]
# WantedBy=multi-user.target
```

**验证清单**:
- [ ] 服务已启用
- [ ] 服务文件包含正确的 WantedBy 配置

### 测试 8: 卸载服务

**目标**: 验证服务能否正常卸载

```bash
# 1. 执行卸载命令
sudo ./iptv-aggregator -s uninstall

# 2. 验证卸载结果
# 应该看到：
# ✓ Service uninstalled successfully!

# 3. 验证文件已删除
ls /opt/iptv-aggregator 2>&1

# 应该显示：cannot access '/opt/iptv-aggregator': No such file or directory

# 4. 验证服务文件已删除
ls /etc/systemd/system/iptv-aggregator.service 2>&1

# 应该显示：cannot access: No such file or directory

# 5. 验证服务已禁用
sudo systemctl is-enabled iptv-aggregator 2>&1

# 应该显示：disabled 或 Unit iptv-aggregator.service could not be found
```

**验证清单**:
- [ ] 卸载命令执行成功
- [ ] `/opt/iptv-aggregator` 目录已删除
- [ ] 服务文件已删除
- [ ] 服务已禁用

### 测试 9: 自定义配置文件安装

**目标**: 验证能否使用自定义配置文件安装

```bash
# 1. 创建自定义配置文件
cp config.json custom-config.json
# 编辑 custom-config.json（可选）

# 2. 使用自定义配置安装
sudo ./iptv-aggregator -s install -config custom-config.json

# 3. 验证配置文件已复制
ls -la /opt/iptv-aggregator/

# 应该看到 custom-config.json

# 4. 检查服务文件中的配置路径
grep ExecStart /etc/systemd/system/iptv-aggregator.service

# 应该显示：
# ExecStart=/opt/iptv-aggregator/iptv-aggregator -config custom-config.json

# 5. 卸载测试
sudo ./iptv-aggregator -s uninstall
```

**验证清单**:
- [ ] 自定义配置文件已复制
- [ ] 服务文件中的配置路径正确
- [ ] 服务能够正常启动

### 测试 10: 权限检查

**目标**: 验证权限检查是否正常工作

```bash
# 1. 尝试不使用 sudo 安装（应该失败）
./iptv-aggregator -s install

# 应该显示错误：
# this operation requires root privileges. Please run with sudo

# 2. 验证错误消息清晰
# 应该提示用户使用 sudo
```

**验证清单**:
- [ ] 非 root 用户无法执行安装
- [ ] 错误消息清晰明了
- [ ] 提示用户使用 sudo

## 故障排查

### 问题 1: 权限被拒绝

```bash
# 症状：Permission denied 错误

# 解决方案：
sudo ./iptv-aggregator -s install
```

### 问题 2: systemctl 命令不存在

```bash
# 症状：systemctl: command not found

# 解决方案：
# 确保系统使用 systemd
systemctl --version

# 如果不可用，可能需要升级系统或使用其他 init 系统
```

### 问题 3: 服务无法启动

```bash
# 症状：Active: failed

# 解决方案：
# 1. 查看详细错误
sudo journalctl -u iptv-aggregator -n 100

# 2. 检查配置文件
cat /opt/iptv-aggregator/config.json

# 3. 手动运行程序测试
/opt/iptv-aggregator/iptv-aggregator -config /opt/iptv-aggregator/config.json
```

### 问题 4: 无法创建 /opt 目录

```bash
# 症状：failed to create service directory

# 解决方案：
# 检查 /opt 目录权限
ls -ld /opt

# 如果 /opt 不存在，创建它
sudo mkdir -p /opt
sudo chmod 755 /opt
```

## 性能测试

### 内存使用

```bash
# 启动服务后检查内存使用
ps aux | grep iptv-aggregator

# 或使用 systemd-cgtop
systemd-cgtop
```

### CPU 使用

```bash
# 监控 CPU 使用
top -p $(pgrep iptv-aggregator)
```

## 清理

完成测试后，清理测试环境：

```bash
# 1. 卸载服务
sudo ./iptv-aggregator -s uninstall

# 2. 删除编译输出
make clean

# 3. 删除临时文件
rm -f custom-config.json
```

## 测试报告模板

```
测试日期: ____________________
测试系统: ____________________
Debian 版本: ____________________
Go 版本: ____________________
测试人员: ____________________

测试结果:
- [ ] 测试 1: 基本安装 - PASS / FAIL
- [ ] 测试 2: 验证安装文件 - PASS / FAIL
- [ ] 测试 3: 启动服务 - PASS / FAIL
- [ ] 测试 4: 查看日志 - PASS / FAIL
- [ ] 测试 5: 停止服务 - PASS / FAIL
- [ ] 测试 6: 重启服务 - PASS / FAIL
- [ ] 测试 7: 开机自启 - PASS / FAIL
- [ ] 测试 8: 卸载服务 - PASS / FAIL
- [ ] 测试 9: 自定义配置 - PASS / FAIL
- [ ] 测试 10: 权限检查 - PASS / FAIL

总体结果: PASS / FAIL

备注:
____________________________________________________________________
____________________________________________________________________
```

## 相关文档

- [SERVICE_MANAGEMENT.md](SERVICE_MANAGEMENT.md) - 详细使用文档
- [SERVICE_QUICK_REFERENCE.md](SERVICE_QUICK_REFERENCE.md) - 快速参考
