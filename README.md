# IPTV M3U 聚合软件

## 项目概述

这是一个IPTV M3U聚合软件，旨在整合多个IPTV M3U订阅源，对重复频道进行合并，提供基于Web的用户界面进行订阅管理，并包含直播流的测试和周期性更新功能。

## 核心功能

- **订阅源管理**：添加、显示、删除和编辑M3U订阅源
- **M3U解析与聚合**：抓取、解析M3U文件并合并频道
- **流媒体测试**：测试频道URL的可用性
- **周期性更新**：按计划自动更新订阅源
- **导出功能**：将聚合后的频道列表导出为M3U或JSON格式
- **用户界面**：提供Web界面进行管理和操作

## 技术栈

- **后端**：Python 3.x
- **Web框架**：Flask
- **HTTP客户端**：requests
- **调度**：APScheduler
- **前端**：HTML, CSS, JavaScript, Bootstrap

## 安装与运行

1. 安装依赖：
```
pip install -r requirements.txt
```

2. 启动应用：
```
python app.py
```

3. 在浏览器中访问：
```
http://localhost:5000
```

## 项目结构

```
.
├── app.py                  # 应用入口
├── config.py               # 配置文件
├── requirements.txt        # 依赖列表
├── static/                 # 静态文件
│   ├── css/                # CSS样式
│   ├── js/                 # JavaScript文件
│   └── img/                # 图片资源
├── templates/              # HTML模板
└── modules/                # 功能模块
    ├── subscription.py     # 订阅源管理
    ├── parser.py           # M3U解析
    ├── aggregator.py       # 频道聚合
    ├── stream_tester.py    # 流媒体测试
    ├── scheduler.py        # 调度器
    └── exporter.py         # 导出功能
```

## 数据存储

应用数据存储在`data`目录下：
- `subscriptions.json`：订阅源列表
- `channels.json`：聚合后的频道列表
- `config.json`：用户配置

## 许可证

MIT