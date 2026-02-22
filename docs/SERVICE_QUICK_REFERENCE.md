# 服务管理快速参考

## 安装和卸载

```bash
# 安装服务
sudo ./iptv-aggregator -s install

# 卸载服务
sudo ./iptv-aggregator -s uninstall
```

## 服务控制

```bash
# 启动
sudo systemctl start iptv-aggregator

# 停止
sudo systemctl stop iptv-aggregator

# 重启
sudo systemctl restart iptv-aggregator

# 查看状态
sudo systemctl status iptv-aggregator
```

## 日志查看

```bash
# 实时日志
sudo journalctl -u iptv-aggregator -f

# 最近 50 行
sudo journalctl -u iptv-aggregator -n 50

# 最近 1 小时
sudo journalctl -u iptv-aggregator --since "1 hour ago"
```

## 开机自启

```bash
# 启用
sudo systemctl enable iptv-aggregator

# 禁用
sudo systemctl disable iptv-aggregator

# 查看状态
sudo systemctl is-enabled iptv-aggregator
```

## 配置管理

```bash
# 编辑配置
sudo nano /opt/iptv-aggregator/config.json

# 重启使配置生效
sudo systemctl restart iptv-aggregator
```

## 文件位置

| 项目 | 位置 |
|------|------|
| 程序 | `/opt/iptv-aggregator/iptv-aggregator` |
| 配置 | `/opt/iptv-aggregator/config.json` |
| 数据 | `/opt/iptv-aggregator/data/` |
| 服务文件 | `/etc/systemd/system/iptv-aggregator.service` |
| 日志 | `journalctl -u iptv-aggregator` |

## 故障排查

```bash
# 查看详细错误
sudo journalctl -u iptv-aggregator -n 100

# 检查服务文件
sudo cat /etc/systemd/system/iptv-aggregator.service

# 检查文件权限
ls -la /opt/iptv-aggregator/

# 验证配置文件
cat /opt/iptv-aggregator/config.json
```
