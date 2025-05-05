# -*- coding: utf-8 -*-
"""
IPTV M3U 聚合软件

应用入口文件，实现Web界面和核心功能的集成。
"""

import os
import json
import logging
from datetime import datetime
from flask import Flask, render_template, request, jsonify, send_from_directory, redirect, url_for, make_response

# 导入功能模块
from modules.subscription import SubscriptionManager
from modules.parser import M3UParser
from modules.aggregator import ChannelAggregator
from modules.stream_tester import StreamTester
from modules.scheduler import UpdateScheduler
from modules.exporter import ChannelExporter

# 导入配置
from config import Config

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# 创建Flask应用
app = Flask(__name__)

# 为所有请求添加 now 变量
@app.context_processor
def inject_now():
    return {'now': datetime.now()}

# 初始化模块
config = Config()
subscription_manager = SubscriptionManager(data_dir=config.data_dir)
m3u_parser = M3UParser(timeout=config.request_timeout)
channel_aggregator = ChannelAggregator(data_dir=config.data_dir)
stream_tester = StreamTester(timeout=config.stream_test_timeout, max_workers=config.max_test_workers)
scheduler = UpdateScheduler()
channel_exporter = ChannelExporter(data_dir=config.data_dir)

# 确保数据目录存在
if not os.path.exists(config.data_dir):
    os.makedirs(config.data_dir)

# 更新函数
def update_subscriptions():
    """更新所有订阅源"""
    logger.info("开始更新订阅源")
    subscriptions = subscription_manager.get_all_subscriptions()
    all_channels = []
    
    for sub in subscriptions:
        url = sub['url']
        logger.info(f"正在更新订阅源: {url}")
        
        # 抓取M3U文件
        success, content = m3u_parser.fetch_m3u(url)
        
        if success:
            # 解析频道
            channels = m3u_parser.parse_m3u(content, source_url=url)
            all_channels.extend(channels)
            
            # 更新订阅源状态
            subscription_manager.update_subscription_status(
                url, 'active', channel_count=len(channels))
            logger.info(f"订阅源 {url} 更新成功，获取到 {len(channels)} 个频道")
        else:
            # 更新订阅源状态为失败
            subscription_manager.update_subscription_status(url, 'failed')
            logger.error(f"订阅源 {url} 更新失败: {content}")
    
    # 聚合频道
    if all_channels:
        total, added, updated = channel_aggregator.aggregate_channels(
            all_channels, match_by=config.match_by, similarity_threshold=config.similarity_threshold)
        logger.info(f"频道聚合完成: 总计 {total} 个频道，新增 {added} 个，更新 {updated} 个")
    
    return True

# 测试流函数
def test_streams():
    """测试所有频道流"""
    logger.info("开始测试频道流")
    channels = channel_aggregator.get_all_channels()
    
    # 批量测试
    updated_channels = stream_tester.batch_test(channels, test_all_sources=config.test_all_sources)
    
    # 保存测试结果
    channel_aggregator.save_channels()
    
    logger.info("频道流测试完成")
    return True

# 初始化调度器
@app.before_first_request
def init_scheduler():
    """初始化调度器"""
    # 添加订阅更新任务
    scheduler.add_interval_job(
        'update_subscriptions', 
        update_subscriptions, 
        hours=config.update_interval_hours
    )
    
    # 添加流测试任务
    if config.enable_stream_test:
        scheduler.add_interval_job(
            'test_streams', 
            test_streams, 
            hours=config.test_interval_hours
        )
    
    # 启动调度器
    scheduler.start()
    logger.info("调度器已初始化")

# 路由：首页
@app.route('/')
def index():
    """首页"""
    subscriptions = subscription_manager.get_all_subscriptions()
    channels = channel_aggregator.get_all_channels()
    groups = channel_aggregator.get_channel_groups()
    
    # 统计信息
    stats = {
        'total_subscriptions': len(subscriptions),
        'total_channels': len(channels),
        'total_groups': len(groups),
        'online_channels': sum(1 for ch in channels if 'test_results' in ch and ch['test_results'].get('status') == 'online'),
        'offline_channels': sum(1 for ch in channels if 'test_results' in ch and ch['test_results'].get('status') == 'offline'),
        'untested_channels': sum(1 for ch in channels if 'test_results' not in ch or ch['test_results'].get('status') == 'untested')
    }
    
    return render_template('index.html', 
                           subscriptions=subscriptions, 
                           channels=channels, 
                           groups=groups, 
                           stats=stats)

# 路由：订阅源管理
@app.route('/subscriptions')
def subscription_list():
    """订阅源列表"""
    subscriptions = subscription_manager.get_all_subscriptions()
    return render_template('subscriptions.html', subscriptions=subscriptions)

# 路由：添加订阅源
@app.route('/subscriptions/add', methods=['GET', 'POST'])
def add_subscription():
    """添加订阅源"""
    if request.method == 'POST':
        url = request.form.get('url', '').strip()
        name = request.form.get('name', '').strip()
        
        if url:
            success, message = subscription_manager.add_subscription(url, name)
            if success:
                # 立即更新新添加的订阅源
                success, content = m3u_parser.fetch_m3u(url)
                if success:
                    channels = m3u_parser.parse_m3u(content, source_url=url)
                    channel_aggregator.aggregate_channels(channels)
                    subscription_manager.update_subscription_status(url, 'active', channel_count=len(channels))
                
                return redirect(url_for('subscription_list'))
            else:
                return render_template('add_subscription.html', error=message)
        else:
            return render_template('add_subscription.html', error="URL不能为空")
    
    return render_template('add_subscription.html')

# 路由：删除订阅源
@app.route('/subscriptions/delete/<path:url>', methods=['POST'])
def delete_subscription(url):
    """删除订阅源"""
    success, message = subscription_manager.remove_subscription(url)
    return jsonify({'success': success, 'message': message})

# 路由：编辑订阅源
@app.route('/subscriptions/edit/<path:url>', methods=['GET', 'POST'])
def edit_subscription(url):
    """编辑订阅源"""
    subscription = subscription_manager.get_subscription(url)
    
    if not subscription:
        return redirect(url_for('subscription_list'))
    
    if request.method == 'POST':
        new_url = request.form.get('url', '').strip()
        new_name = request.form.get('name', '').strip()
        
        if new_url:
            success, message = subscription_manager.update_subscription(url, new_url, new_name)
            if success:
                return redirect(url_for('subscription_list'))
            else:
                return render_template('edit_subscription.html', 
                                       subscription=subscription, 
                                       error=message)
        else:
            return render_template('edit_subscription.html', 
                                   subscription=subscription, 
                                   error="URL不能为空")
    
    return render_template('edit_subscription.html', subscription=subscription)

# 路由：频道列表
@app.route('/channels')
def channel_list():
    """频道列表"""
    channels = channel_aggregator.get_all_channels()
    groups = channel_aggregator.get_channel_groups()
    
    # 过滤参数
    group = request.args.get('group', '')
    status = request.args.get('status', '')
    query = request.args.get('q', '').lower()
    
    # 应用过滤
    filtered_channels = channels
    
    if group:
        filtered_channels = [ch for ch in filtered_channels if ch['group_title'] == group]
    
    if status:
        if status == 'online':
            filtered_channels = [ch for ch in filtered_channels if 'test_results' in ch and ch['test_results'].get('status') == 'online']
        elif status == 'offline':
            filtered_channels = [ch for ch in filtered_channels if 'test_results' in ch and ch['test_results'].get('status') == 'offline']
        elif status == 'untested':
            filtered_channels = [ch for ch in filtered_channels if 'test_results' not in ch or ch['test_results'].get('status') == 'untested']
    
    if query:
        filtered_channels = [ch for ch in filtered_channels if query in ch['name'].lower()]
    
    return render_template('channels.html', 
                           channels=filtered_channels, 
                           groups=groups, 
                           current_group=group, 
                           current_status=status, 
                           query=query)

# 路由：测试频道
@app.route('/channels/test/<int:channel_id>', methods=['POST'])
def test_channel(channel_id):
    """测试单个频道"""
    channels = channel_aggregator.get_all_channels()
    
    if 0 <= channel_id < len(channels):
        channel = channels[channel_id]
        updated_channel = stream_tester.batch_test([channel], test_all_sources=True)[0]
        channel_aggregator.save_channels()
        
        return jsonify({
            'success': True, 
            'status': updated_channel['test_results'].get('status', 'untested'),
            'message': '测试完成'
        })
    else:
        return jsonify({'success': False, 'message': '频道不存在'})

# 路由：批量测试频道
@app.route('/channels/test-all', methods=['POST'])
def test_all_channels():
    """批量测试所有频道"""
    # 启动测试任务
    scheduler.add_interval_job('test_streams_once', test_streams, seconds=1)
    return jsonify({'success': True, 'message': '测试任务已启动'})

# 路由：手动更新订阅
@app.route('/update', methods=['POST'])
def manual_update():
    """手动更新订阅"""
    # 启动更新任务
    scheduler.add_interval_job('update_subscriptions_once', update_subscriptions, seconds=1)
    return jsonify({'success': True, 'message': '更新任务已启动'})

# 路由：导出
@app.route('/export')
def export_page():
    """导出页面"""
    exports = channel_exporter.get_export_list()
    return render_template('export.html', exports=exports)

# 路由：执行导出
@app.route('/export/create', methods=['POST'])
def create_export():
    """创建导出文件"""
    export_type = request.form.get('type', 'm3u')
    only_working = request.form.get('only_working') == 'on'
    
    channels = channel_aggregator.get_all_channels()
    
    if export_type == 'm3u':
        success, result = channel_exporter.export_m3u(channels, only_working=only_working)
    else:  # json
        success, result = channel_exporter.export_json(channels, only_working=only_working)
    
    if success:
        return redirect(url_for('export_page'))
    else:
        return render_template('export.html', 
                               exports=channel_exporter.get_export_list(), 
                               error=result)

# 路由：下载导出文件
@app.route('/export/download/<path:filename>')
def download_export(filename):
    """下载导出文件"""
    return send_from_directory(os.path.join(config.data_dir, 'exports'), filename, as_attachment=True)

# API路由：获取在线频道M3U
@app.route('/api/playlist.m3u', methods=['GET'])
def api_get_playlist():
    """获取在线频道M3U播放列表
    该接口可供电视盒子等设备直接访问，获取所有在线频道
    """
    channels = channel_aggregator.get_all_channels()
    
    # 仅保留在线频道
    online_channels = [ch for ch in channels 
                      if 'test_results' in ch 
                      and ch['test_results'].get('status') == 'online']
    
    # 生成M3U内容
    content = []
    content.append('#EXTM3U')
    
    for channel in online_channels:
        # 获取工作的URL
        url = channel.get('test_results', {}).get('working_url') or channel.get('url', '')
        if not url:
            continue
            
        # 构建EXTINF行
        extinf = '#EXTINF:-1'
        if channel.get('tvg_id'):
            extinf += f' tvg-id="{channel["tvg_id"]}"'
        if channel.get('tvg_name'):
            extinf += f' tvg-name="{channel["tvg_name"]}"'
        elif channel.get('name'):
            extinf += f' tvg-name="{channel["name"]}"'
        if channel.get('tvg_logo'):
            extinf += f' tvg-logo="{channel["tvg_logo"]}"'
        if channel.get('group_title'):
            extinf += f' group-title="{channel["group_title"]}"'
            
        # 添加频道名称
        extinf += f',{channel["name"]}'
        
        # 添加到内容列表
        content.append(extinf)
        content.append(url)
    
    # 返回M3U文件
    response = make_response('\n'.join(content))
    response.headers['Content-Type'] = 'audio/x-mpegurl'
    response.headers['Content-Disposition'] = 'inline; filename=playlist.m3u'
    return response

# 路由：删除导出文件
@app.route('/export/delete/<path:filename>', methods=['POST'])
def delete_export(filename):
    """删除导出文件"""
    success = channel_exporter.delete_export(filename)
    return jsonify({'success': success})

# 路由：配置页面
@app.route('/settings', methods=['GET', 'POST'])
def settings():
    """配置页面"""
    if request.method == 'POST':
        # 更新配置
        config.update_interval_hours = int(request.form.get('update_interval_hours', 24))
        config.enable_stream_test = request.form.get('enable_stream_test') == 'on'
        config.test_interval_hours = int(request.form.get('test_interval_hours', 24))
        config.match_by = request.form.get('match_by', 'name')
        # 处理相似度阈值：从百分比转换为小数
        similarity_threshold = float(request.form.get('similarity_threshold', 85))
        config.similarity_threshold = similarity_threshold / 100
        config.test_all_sources = request.form.get('test_all_sources') == 'on'
        
        # 保存配置
        config.save_config()
        
        # 更新调度任务
        scheduler.remove_job('update_subscriptions')
        scheduler.add_interval_job(
            'update_subscriptions', 
            update_subscriptions, 
            hours=config.update_interval_hours
        )
        
        if config.enable_stream_test:
            scheduler.remove_job('test_streams')
            scheduler.add_interval_job(
                'test_streams', 
                test_streams, 
                hours=config.test_interval_hours
            )
        else:
            scheduler.remove_job('test_streams')
        
        return redirect(url_for('settings'))
    
    return render_template('settings.html', config=config)

# 启动应用
if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=80)