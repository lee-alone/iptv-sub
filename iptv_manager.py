import requests
import m3u8
import hashlib
import json
import logging
from typing import Dict, List, Set
from flask import Flask, request, jsonify, render_template, send_from_directory
import yaml
from pathlib import Path
import os
from datetime import datetime

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Flask应用配置
app = Flask(__name__, 
    template_folder=os.path.abspath('templates'),
    static_folder=os.path.abspath('static')
)
app.config['JSON_AS_ASCII'] = False  # 支持中文

# IPTV管理器实例
iptv_manager = None

class Channel:
    def __init__(self, url: str, name: str = None, tvg_name: str = None, 
                 tvg_logo: str = None, group_title: str = None):
        self.url = url.strip()
        self.name = name or "Unknown"
        self.tvg_name = tvg_name
        self.tvg_logo = tvg_logo
        self.group_title = group_title or "未分类"
        self.last_check = None
        self.resolution = "Unknown"
        self.segments_count = 0
        self.duration = 0

    def __eq__(self, other):
        if not isinstance(other, Channel):
            return False
        return self.url == other.url or (
            self.tvg_name == other.tvg_name and 
            self.name == other.name and 
            self.group_title == other.group_title
        )

    def __hash__(self):
        # 使用URL和频道信息组合生成哈希值，用于去重
        return hash((self.url, self.tvg_name, self.name, self.group_title))

    def to_m3u8_entry(self) -> str:
        """生成M3U8文件中的条目"""
        extinf = '#EXTINF:-1'
        if self.tvg_name:
            extinf += f' tvg-name="{self.tvg_name}"'
        if self.tvg_logo:
            extinf += f' tvg-logo="{self.tvg_logo}"'
        if self.group_title:
            extinf += f' group-title="{self.group_title}"'
        extinf += f',{self.name}\n'
        return f'{extinf}{self.url}\n'

class IPTVManager:
    def __init__(self):
        self.channels: Set[Channel] = set()  # 使用集合存储Channel对象
        self.config_file = Path("config.json")
        self.last_check_time = {}  # 新增：记录每个URL的最后检查时间
        self.load_config()

    def load_config(self):
        """从配置文件加载频道信息"""
        if self.config_file.exists():
            try:
                with open(self.config_file, 'r', encoding='utf-8') as f:
                    config = json.load(f)
                    if config and 'channels' in config:
                        for channel_data in config['channels']:
                            channel = Channel(
                                url=channel_data['url'],
                                name=channel_data['name'],
                                tvg_name=channel_data.get('tvg_name'),
                                tvg_logo=channel_data.get('tvg_logo'),
                                group_title=channel_data.get('group_title')
                            )
                            channel.last_check = channel_data.get('last_check')
                            channel.resolution = channel_data.get('resolution', 'Unknown')
                            channel.segments_count = channel_data.get('segments_count', 0)
                            channel.duration = channel_data.get('duration', 0)
                            self.channels.add(channel)
                logger.info(f"已从配置文件加载 {len(self.channels)} 个频道")
            except Exception as e:
                logger.error(f"加载配置文件失败: {str(e)}")

    def save_config(self):
        """保存配置到JSON文件"""
        try:
            config = {
                'channels': [
                    {
                        'url': ch.url,
                        'name': ch.name,
                        'tvg_name': ch.tvg_name,
                        'tvg_logo': ch.tvg_logo,
                        'group_title': ch.group_title,
                        'last_check': ch.last_check,
                        'resolution': ch.resolution,
                        'segments_count': ch.segments_count,
                        'duration': ch.duration
                    } for ch in self.channels
                ]
            }
            with open(self.config_file, 'w', encoding='utf-8') as f:
                json.dump(config, f, ensure_ascii=False, indent=2)
            logger.info("配置已保存")
        except Exception as e:
            logger.error(f"保存配置文件失败: {str(e)}")

    def generate_fingerprint(self, url: str) -> str:
        """生成M3U8内容的指纹（改进版）"""
        try:
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            
            # 仅使用URL本身生成基础指纹
            base_fingerprint = hashlib.md5(url.encode()).hexdigest()
            
            # 尝试解析M3U8内容
            try:
                playlist = m3u8.loads(response.text)
                if playlist.segments:
                    # 如果能够解析segments，使用第一个segment的信息加强指纹
                    first_segment = playlist.segments[0].uri
                    return hashlib.md5(f"{base_fingerprint}:{first_segment}".encode()).hexdigest()
            except:
                pass
            
            # 如果无法解析M3U8或没有segments，只返回基础指纹
            return base_fingerprint
            
        except Exception as e:
            logger.error(f"生成指纹失败 {url}: {str(e)}")
            return ""

    def parse_m3u_content(self, content: str) -> List[Channel]:
        """解析M3U内容，返回频道列表"""
        channels = []
        lines = []
        
        # 预处理：处理换行问题
        for line in content.splitlines():
            line = line.strip()
            if not line:
                continue
            if line.startswith('#EXTINF:'):
                # 如果是新的EXTINF行，直接添加
                lines.append(line)
            elif line.startswith('#'):
                # 其他注释行直接添加
                lines.append(line)
            elif lines and lines[-1].startswith('#EXTINF:'):
                # 如果上一行是EXTINF，且当前行不是以#开头，
                # 检查是否是被分割的EXTINF属性
                if '="' in line and not '"' in line.split('="')[1]:
                    # 这是被分割的属性行，追加到上一行
                    lines[-1] = lines[-1] + line
                else:
                    # 这是URL行
                    lines.append(line)
            else:
                # 可能是被分割的URL，尝试和前一行合并
                if lines and not lines[-1].startswith('#'):
                    if any(lines[-1].endswith(ext) for ext in ['.m3u8', '.m3u']):
                        # 上一行已经是完整URL，这是新的一行
                        lines.append(line)
                    else:
                        # 合并URL
                        lines[-1] = lines[-1] + line
                else:
                    lines.append(line)
        
        # 解析处理后的行
        current_channel = None
        current_extinf = ''
        
        for line in lines:
            if line.startswith('#EXTM3U'):
                continue
            
            if line.startswith('#EXTINF:'):
                # 保存完整的EXTINF行，以便解析
                current_extinf = line
                try:
                    # 提取属性
                    tvg_name = self._extract_attribute(line, 'tvg-id') or self._extract_attribute(line, 'tvg-name')
                    tvg_logo = self._extract_attribute(line, 'tvg-logo')
                    group_title = self._extract_attribute(line, 'group-title')
                    
                    # 提取频道名称（处理特殊情况）
                    name_part = line.split(',')[-1].strip()
                    if name_part and not name_part.startswith('tvg-'):
                        name = name_part
                    else:
                        # 如果没有找到名称，尝试使用tvg-name
                        name = tvg_name or "Unknown"
                    
                    current_channel = {
                        'name': name,
                        'tvg_name': tvg_name,
                        'tvg_logo': tvg_logo,
                        'group_title': group_title
                    }
                except Exception as e:
                    logger.error(f"解析EXTINF行失败: {str(e)}, 行内容: {line}")
                    current_channel = None
                    
            elif not line.startswith('#') and current_channel is not None:
                # 处理URL行
                url = line.strip()
                if url.lower().endswith('.m3u8'):
                    try:
                        # 确保URL是完整的
                        if not url.startswith(('http://', 'https://')):
                            if url.startswith('//'):
                                url = 'http:' + url
                            else:
                                logger.warning(f"跳过无效URL: {url}")
                                continue
                        
                        channel = Channel(
                            url=url,
                            name=current_channel['name'],
                            tvg_name=current_channel['tvg_name'],
                            tvg_logo=current_channel['tvg_logo'],
                            group_title=current_channel['group_title']
                        )
                        channels.append(channel)
                        logger.info(f"成功解析频道: {channel.name} ({url})")
                    except Exception as e:
                        logger.error(f"创建频道对象失败: {str(e)}, URL: {url}")
                current_channel = None
        
        logger.info(f"解析完成，共找到 {len(channels)} 个频道")
        return channels

    def _extract_attribute(self, line: str, attr: str) -> str:
        """从EXTINF行提取属性值"""
        try:
            import re
            pattern = f'{attr}="([^"]*)"'
            match = re.search(pattern, line)
            return match.group(1) if match else None
        except:
            return None

    def add_m3u_content(self, url: str) -> Dict[str, any]:
        """添加M3U内容，支持单个地址或完整的M3U列表"""
        try:
            logger.info(f"开始加载M3U内容: {url}")
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            content = response.text
            
            # 检查是否是M3U列表文件
            if content.strip().upper().startswith('#EXTM3U'):
                # 解析M3U列表
                new_channels = self.parse_m3u_content(content)
                results = {
                    'success': True,
                    'total': len(new_channels),
                    'added': 0,
                    'skipped': 0,
                    'failed': 0,
                    'details': []
                }
                
                skipped_reasons = {
                    'invalid_url': 0,
                    'unreachable': 0,
                    'duplicate': 0,
                    'parse_error': 0
                }
                
                for channel in new_channels:
                    try:
                        if not channel.url.startswith(('http://', 'https://')):
                            skipped_reasons['invalid_url'] += 1
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '无效的URL格式'
                            })
                            continue
                            
                        if not self.check_url_availability(channel.url):
                            skipped_reasons['unreachable'] += 1
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '地址无法访问'
                            })
                            continue
                            
                        if channel in self.channels:
                            skipped_reasons['duplicate'] += 1
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '频道已存在'
                            })
                            continue
                            
                        # 尝试加载频道信息
                        channel_info = self.get_channel_info(channel.url)
                        if channel_info.get('error'):
                            skipped_reasons['parse_error'] += 1
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': f'解析失败: {channel_info["error"]}'
                            })
                            continue
                            
                        # 更新频道信息
                        channel.resolution = channel_info.get('resolution', 'Unknown')
                        channel.segments_count = channel_info.get('segments_count', 0)
                        channel.duration = channel_info.get('duration', 0)
                        channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                        
                        # 添加到频道列表
                        self.channels.add(channel)
                        results['added'] += 1
                        results['details'].append({
                            'url': channel.url,
                            'name': channel.name,
                            'status': 'added',
                            'info': {
                                'resolution': channel.resolution,
                                'duration': channel.duration
                            }
                        })
                        
                    except Exception as e:
                        results['failed'] += 1
                        results['details'].append({
                            'url': channel.url,
                            'name': channel.name,
                            'status': 'error',
                            'error': str(e)
                        })
                
                # 添加汇总信息
                results['summary'] = {
                    'invalid_urls': skipped_reasons['invalid_url'],
                    'unreachable': skipped_reasons['unreachable'],
                    'duplicates': skipped_reasons['duplicate'],
                    'parse_errors': skipped_reasons['parse_error']
                }
                
                self.save_config()
                logger.info(f"导入完成: 总数={results['total']}, 添加={results['added']}, "
                          f"跳过={results['skipped']}, 失败={results['failed']}")
                return results
            else:
                # 单个M3U8地址
                channel = Channel(url=url)
                success = self.add_channel(channel)
                self.save_config()
                return {
                    'success': success,
                    'total': 1,
                    'added': 1 if success else 0,
                    'skipped': 0 if success else 1,
                    'failed': 0,
                    'details': [{
                        'url': url,
                        'status': 'added' if success else 'skipped'
                    }]
                }
                
        except Exception as e:
            logger.error(f"添加M3U内容失败: {str(e)}")
            return {
                'success': False,
                'error': str(e),
                'total': 1,
                'added': 0,
                'skipped': 0,
                'failed': 1,
                'details': [{
                    'url': url,
                    'status': 'error',
                    'error': str(e)
                }]
            }

    def add_channel(self, channel: Channel) -> bool:
        """添加新的频道"""
        try:
            # 检查URL是否可访问
            if not self.check_url_availability(channel.url):
                logger.info(f"频道地址无法访问: {channel.url}")
                return False

            # 更新频道信息
            channel_info = self.get_channel_info(channel.url)
            channel.resolution = channel_info['resolution']
            channel.segments_count = channel_info['segments_count']
            channel.duration = channel_info['duration']
            channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

            # 检查是否已存在
            if channel in self.channels:
                logger.info(f"频道已存在: {channel.name} ({channel.url})")
                return False

            # 添加新频道
            self.channels.add(channel)
            logger.info(f"成功添加新频道: {channel.name} ({channel.url})")
            return True

        except Exception as e:
            logger.error(f"添加频道失败: {str(e)}")
            return False

    def generate_m3u8_playlist(self) -> str:
        """生成新的M3U8播放列表"""
        playlist = "#EXTM3U\n"
        # 按分组排序频道
        sorted_channels = sorted(self.channels, key=lambda x: (x.group_title or "", x.name or ""))
        for channel in sorted_channels:
            playlist += channel.to_m3u8_entry()
        return playlist

    def get_channel_info(self, url: str) -> Dict:
        """获取频道信息，包含错误处理"""
        try:
            response = requests.get(url, timeout=10, allow_redirects=True)
            response.raise_for_status()
            content = response.text

            # 预处理内容，移除可能的BOM标记
            if content.startswith('\ufeff'):
                content = content[1:]
                
            try:
                playlist = m3u8.loads(content)
            except Exception as parse_error:
                return {
                    'name': 'Unknown',
                    'segments_count': 0,
                    'duration': 0,
                    'resolution': 'Unknown',
                    'last_check': 'Error',
                    'error': f'解析M3U8失败: {str(parse_error)}'
                }

            # 提取频道名称（尝试多种方式）
            channel_name = "Unknown"
            try:
                # 1. 从EXTINF行提取
                for line in content.split('\n'):
                    if line.startswith('#EXTINF:'):
                        # 尝试提取频道名称
                        if ',' in line:
                            # 标准格式：#EXTINF:-1,频道名
                            channel_name = line.split(',', 1)[1].strip()
                        elif 'tvg-name="' in line:
                            # 替代格式：从tvg-name属性提取
                            import re
                            match = re.search('tvg-name="([^"]+)"', line)
                            if match:
                                channel_name = match.group(1)
                        break
            except Exception as e:
                logger.warning(f"提取频道名称失败: {str(e)}")

            # 获取分辨率信息
            resolution = self._get_resolution(playlist)
            if resolution == 'Unknown' and playlist.segments:
                # 尝试从第一个分片获取更多信息
                try:
                    seg_url = playlist.segments[0].uri
                    if not seg_url.startswith(('http://', 'https://')):
                        # 处理相对路径
                        from urllib.parse import urljoin
                        seg_url = urljoin(url, seg_url)
                    seg_response = requests.head(seg_url, timeout=5)
                    content_type = seg_response.headers.get('content-type', '')
                    if content_type:
                        resolution = f"Format: {content_type}"
                except Exception as e:
                    logger.warning(f"获取分片信息失败: {str(e)}")

            return {
                'name': channel_name,
                'segments_count': len(playlist.segments),
                'duration': sum(float(seg.duration) for seg in playlist.segments),
                'resolution': resolution,
                'last_check': datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            }
            
        except requests.RequestException as e:
            error_msg = str(e)
            if 'timeout' in error_msg.lower():
                error_msg = '连接超时'
            elif 'connection' in error_msg.lower():
                error_msg = '连接失败'
            return {
                'name': 'Unknown',
                'segments_count': 0,
                'duration': 0,
                'resolution': 'Unknown',
                'last_check': 'Error',
                'error': f'访问失败: {error_msg}'
            }
        except Exception as e:
            return {
                'name': 'Unknown',
                'segments_count': 0,
                'duration': 0,
                'resolution': 'Unknown',
                'last_check': 'Error',
                'error': f'处理失败: {str(e)}'
            }

    def check_url_availability(self, url: str) -> bool:
        """增强版的URL可用性检查"""
        try:
            # 规范化URL
            url = self._normalize_url(url)
            
            # 使用HEAD请求快速检查资源是否存在
            try:
                head_response = requests.head(url, timeout=5, allow_redirects=True)
                if head_response.status_code == 200:
                    # 快速检查成功
                    self.last_check_time[url] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                    return True
            except:
                pass  # 如果HEAD请求失败，继续尝试GET请求
            
            # 使用GET请求获取内容
            response = requests.get(url, timeout=10, stream=True)
            response.raise_for_status()
            
            # 只读取前8KB来验证内容
            content_start = response.raw.read(8192).decode('utf-8', errors='ignore')
            response.close()
            
            # 验证内容是否符合M3U格式
            if not self._is_m3u_content(content_start):
                logger.warning(f"URL内容不是有效的M3U格式: {url}")
                return False
                
            self.last_check_time[url] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            return True
            
        except requests.RequestException as e:
            logger.error(f"检查URL失败 {url}: {str(e)}")
            return False
        except Exception as e:
            logger.error(f"验证URL时发生错误 {url}: {str(e)}")
            return False

    def validate_m3u_content(self, content: str) -> Dict[str, any]:
        """验证M3U内容的有效性"""
        try:
            if not content.strip():
                return {'valid': False, 'error': '内容为空'}
                
            if not self._is_m3u_content(content):
                return {'valid': False, 'error': '不是有效的M3U格式'}
            
            # 尝试解析内容
            try:
                playlist = m3u8.loads(content)
                if not playlist.segments and not playlist.playlists:
                    return {'valid': False, 'error': '播放列表没有有效的媒体内容'}
            except Exception as e:
                return {'valid': False, 'error': f'解析M3U内容失败: {str(e)}'}
            
            # 提取所有URL
            urls = self._extract_m3u_urls(content)
            if not urls:
                return {'valid': False, 'error': '未找到有效的M3U8 URL'}
            
            return {
                'valid': True,
                'urls_count': len(urls),
                'duration': sum(float(seg.duration) for seg in playlist.segments) if playlist.segments else 0,
                'has_variants': bool(playlist.playlists)
            }
            
        except Exception as e:
            return {'valid': False, 'error': f'验证内容时发生错误: {str(e)}'}

    def batch_check_channels(self, auto_repair: bool = False) -> Dict[str, any]:
        """批量检查所有频道，可选择自动修复问题"""
        results = {
            'total': len(self.channels),
            'checked': 0,
            'available': 0,
            'unavailable': 0,
            'repaired': 0,
            'failed': 0,
            'details': []
        }

        channels_to_remove = set()
        channels_to_update = {}

        for channel in self.channels:
            try:
                results['checked'] += 1
                logger.info(f"检查频道 [{results['checked']}/{results['total']}]: {channel.name}")

                # 检查原始URL
                if self.check_url_availability(channel.url):
                    results['available'] += 1
                    continue

                if not auto_repair:
                    results['unavailable'] += 1
                    results['details'].append({
                        'url': channel.url,
                        'name': channel.name,
                        'status': 'unavailable'
                    })
                    continue

                # 尝试自动修复
                fixed = False
                
                # 1. 尝试处理重定向
                try:
                    response = requests.head(channel.url, allow_redirects=True, timeout=5)
                    if response.url != channel.url:
                        new_url = response.url
                        if self.check_url_availability(new_url):
                            channels_to_update[channel] = new_url
                            fixed = True
                except:
                    pass

                # 2. 尝试HTTPS/HTTP切换
                if not fixed:
                    alt_url = None
                    if channel.url.startswith('https://'):
                        alt_url = 'http://' + channel.url[8:]
                    elif channel.url.startswith('http://'):
                        alt_url = 'https://' + channel.url[7:]

                    if alt_url and self.check_url_availability(alt_url):
                        channels_to_update[channel] = alt_url
                        fixed = True

                if fixed:
                    results['repaired'] += 1
                    results['details'].append({
                        'url': channel.url,
                        'name': channel.name,
                        'status': 'repaired',
                        'new_url': channels_to_update[channel]
                    })
                else:
                    results['failed'] += 1
                    results['details'].append({
                        'url': channel.url,
                        'name': channel.name,
                        'status': 'failed'
                    })
                    channels_to_remove.add(channel)

            except Exception as e:
                results['failed'] += 1
                results['details'].append({
                    'url': channel.url,
                    'name': channel.name,
                    'status': 'error',
                    'error': str(e)
                })

        # 应用修复
        if auto_repair:
            # 更新URL
            for channel, new_url in channels_to_update.items():
                channel.url = new_url
                self.last_check_time[new_url] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

            # 移除无效频道
            for channel in channels_to_remove:
                self.channels.remove(channel)

            # 保存更改
            self.save_config()

        return results

    def cleanup_invalid_urls(self, auto_repair: bool = True) -> Dict[str, any]:
        """增强版的无效URL清理功能"""
        results = self.batch_check_channels(auto_repair=auto_repair)
        
        # 添加汇总信息
        if results['repaired'] > 0:
            logger.info(f"成功修复 {results['repaired']} 个频道")
        if results['failed'] > 0:
            logger.warning(f"无法修复 {results['failed']} 个频道")
            
        return results

@app.route('/add', methods=['POST'])
def add_url():
    url = request.json.get('url')
    if not url:
        return jsonify({'error': '未提供URL'}), 400
    
    success = iptv_manager.add_m3u8_url(url)
    return jsonify({'success': success})

# Web路由
@app.route('/')
def index():
    """主页"""
    return render_template('index.html')

# API路由
@app.route('/api/urls', methods=['GET'])
def get_urls():
    """获取所有M3U8地址"""
    return jsonify({'urls': iptv_manager.m3u8_urls})

@app.route('/api/add', methods=['POST'])
def api_add_url():
    """添加新的M3U8地址或M3U列表"""
    url = request.json.get('url')
    if not url:
        return jsonify({'error': '未提供URL', 'success': False}), 400
    
    results = iptv_manager.add_m3u_content(url)
    return jsonify(results)

@app.route('/api/cleanup', methods=['POST'])
def api_cleanup():
    """清理无效地址的增强版API"""
    try:
        auto_repair = request.json.get('auto_repair', True)  # 默认启用自动修复
        results = iptv_manager.cleanup_invalid_urls(auto_repair=auto_repair)
        return jsonify(results)
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@app.route('/api/check', methods=['POST'])
def check_channels():
    """检查所有频道状态"""
    try:
        auto_repair = request.json.get('auto_repair', False)  # 默认不自动修复
        results = iptv_manager.batch_check_channels(auto_repair=auto_repair)
        return jsonify(results)
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500

@app.route('/playlist', methods=['GET'])
def get_playlist():
    """获取M3U8播放列表"""
    content = iptv_manager.generate_m3u8_playlist()
    response = app.response_class(
        response=content,
        status=200,
        mimetype='application/vnd.apple.mpegurl'
    )
    response.headers["Content-Disposition"] = "attachment; filename=playlist.m3u8"
    return response

@app.route('/playlist.m3u8')
def playlist_file():
    """获取M3U8播放列表文件"""
    return get_playlist()

@app.route('/api/health', methods=['GET'])
def health_check():
    """健康检查接口"""
    return jsonify({'status': 'ok', 'count': len(iptv_manager.m3u8_urls)})

@app.route('/api/channel-info')
def get_channel_info():
    """获取频道详细信息"""
    url = request.args.get('url')
    if not url:
        return jsonify({'error': '未提供URL'}), 400
    
    info = iptv_manager.get_channel_info(url)
    return jsonify(info)

@app.route('/api/channels', methods=['GET'])
def get_channels():
    """获取所有频道信息"""
    channels_info = [{
        'url': ch.url,
        'name': ch.name,
        'tvg_name': ch.tvg_name,
        'tvg_logo': ch.tvg_logo,
        'group_title': ch.group_title,
        'resolution': ch.resolution,
        'last_check': ch.last_check
    } for ch in iptv_manager.channels]
    
    return jsonify({
        'total': len(channels_info),
        'channels': channels_info,
        'groups': list(set(ch['group_title'] for ch in channels_info if ch['group_title']))
    })

@app.route('/api/groups', methods=['GET'])
def get_groups():
    """获取所有频道分组"""
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

@app.route('/api/channel/<path:url>', methods=['DELETE'])
def delete_channel(url):
    """删除指定频道"""
    try:
        channel_to_remove = None
        for channel in iptv_manager.channels:
            if channel.url == url:
                channel_to_remove = channel
                break
        
        if channel_to_remove:
            iptv_manager.channels.remove(channel_to_remove)
            iptv_manager.save_config()
            return jsonify({'success': True})
        return jsonify({'success': False, 'error': '频道不存在'}), 404
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/playlist/group/<group>', methods=['GET'])
def get_group_playlist(group):
    """获取指定分组的播放列表"""
    playlist = "#EXTM3U\n"
    for channel in sorted(iptv_manager.channels, key=lambda x: x.name or ""):
        if channel.group_title == group:
            playlist += channel.to_m3u8_entry()
    
    response = app.response_class(
        response=playlist,
        status=200,
        mimetype='application/vnd.apple.mpegurl'
    )
    response.headers["Content-Disposition"] = f"attachment; filename={group}.m3u8"
    return response

if __name__ == '__main__':
    # 创建必要的目录
    os.makedirs('templates', exist_ok=True)
    os.makedirs('static', exist_ok=True)
    
    # 初始化IPTV管理器
    iptv_manager = IPTVManager()
    
    # 启动Web服务
    print("正在启动Web服务器...")
    print("请在浏览器中访问 http://localhost:5000")
    app.run(host='0.0.0.0', port=5000, debug=True)
