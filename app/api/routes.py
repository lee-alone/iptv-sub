from flask import Flask, request, jsonify, render_template, Response
from app.models.iptv_manager import IPTVManager
import os

# 创建Flask应用
app = Flask(__name__, 
    template_folder=os.path.abspath('templates'),
    static_folder=os.path.abspath('static')
)
app.config['JSON_AS_ASCII'] = False

# 创建IPTV管理器实例
iptv_manager = IPTVManager()

@app.route('/')
def index():
    """主页"""
    return render_template('index.html')

@app.route('/api/channels', methods=['GET'])
def get_channels():
    """获取所有频道信息"""
    channels = [{
        'url': ch.url,
        'name': ch.name,
        'tvg_name': ch.tvg_name,
        'tvg_logo': ch.tvg_logo,
        'group_title': ch.group_title,
        'resolution': ch.resolution,
        'last_check': ch.last_check
    } for ch in iptv_manager.channels]
    
    groups = list(set(ch['group_title'] for ch in channels if ch['group_title']))
    return jsonify({
        'total': len(channels),
        'channels': channels,
        'groups': groups
    })

@app.route('/api/groups', methods=['GET'])
def get_groups():
    """获取频道分组信息"""
    groups = {}
    for channel in iptv_manager.channels:
        group = channel.group_title or "未分类"
        if group not in groups:
            groups[group] = []
        groups[group].append({
            'name': channel.name,
            'url': channel.url,
            'tvg_logo': channel.tvg_logo
        })
    return jsonify(groups)

@app.route('/api/add', methods=['POST'])
def add_content():
    """添加M3U内容"""
    url = request.json.get('url')
    if not url:
        return jsonify({'error': '未提供URL', 'success': False}), 400
    
    results = iptv_manager.add_m3u_content(url)
    return jsonify(results)

@app.route('/api/cleanup', methods=['POST'])
def cleanup_channels():
    """清理无效频道"""
    auto_repair = request.json.get('auto_repair', True)
    results = iptv_manager.cleanup_channels(auto_repair=auto_repair)
    return jsonify(results)

@app.route('/api/channel/<path:url>', methods=['DELETE'])
def delete_channel(url):
    """删除指定频道"""
    for channel in iptv_manager.channels:
        if channel.url == url:
            iptv_manager.channels.remove(channel)
            iptv_manager.save_channels()
            return jsonify({'success': True})
    return jsonify({'success': False, 'error': '频道不存在'}), 404

@app.route('/playlist')
@app.route('/playlist.m3u8')
def get_playlist():
    """获取完整播放列表"""
    content = iptv_manager.generate_playlist()
    response = Response(
        content,
        mimetype='application/vnd.apple.mpegurl',
        headers={'Content-Disposition': 'attachment; filename=playlist.m3u8'}
    )
    return response

@app.route('/playlist/group/<group>')
def get_group_playlist(group):
    """获取分组播放列表"""
    playlist = "#EXTM3U\n"
    for channel in sorted(iptv_manager.channels, key=lambda x: x.name or ""):
        if channel.group_title == group:
            playlist += channel.to_m3u8_entry()
    
    response = Response(
        playlist,
        mimetype='application/vnd.apple.mpegurl',
        headers={'Content-Disposition': f'attachment; filename={group}.m3u8'}
    )
    return response

@app.route('/api/channel-info')
def get_channel_info():
    """获取频道详细信息"""
    url = request.args.get('url')
    if not url:
        return jsonify({'error': '未提供URL'}), 400
    
    info = iptv_manager.get_channel_info(url)
    return jsonify(info)

@app.route('/api/health')
def health_check():
    """健康检查接口"""
    return jsonify({
        'status': 'ok',
        'total_channels': len(iptv_manager.channels)
    })

@app.route('/api/check/concurrent', methods=['POST'])
def check_concurrent():
    """并发检查所有频道"""
    try:
        # 获取批量大小参数，默认为10
        batch_size = request.json.get('batch_size', 10)
        # 确保批量大小在合理范围内
        batch_size = max(1, min(batch_size, 20))
        
        results = iptv_manager.batch_check_channels_concurrent(batch_size=batch_size)
        return jsonify(results)
    except Exception as e:
        return jsonify({
            'error': str(e),
            'success': False
        }), 500
