import logging
from typing import Dict, List, Set, Optional
from datetime import datetime
import requests
import m3u8
import asyncio
import time
import re

from app.models.channel import Channel
from app.utils.config import ConfigManager
from app.utils.m3u_utils import M3UParser, URLChecker

logger = logging.getLogger(__name__)

class IPTVManager:
    def __init__(self):
        self.channels: Set[Channel] = set()
        self.config = ConfigManager()
        self.last_check_time = {}
        self.load_channels()

    def load_channels(self) -> None:
        """从配置加载频道信息"""
        channels_data = self.config.get('channels', [])
        for channel_data in channels_data:
            try:
                channel = Channel.from_dict(channel_data)
                self.channels.add(channel)
            except Exception as e:
                logger.error(f"加载频道失败: {str(e)}")
        logger.info(f"已加载 {len(self.channels)} 个频道")

    def save_channels(self) -> None:
        """保存频道信息到配置"""
        try:
            channels_data = [ch.to_dict() for ch in self.channels]
            self.config.set('channels', channels_data)
            logger.info(f"已保存 {len(self.channels)} 个频道")
        except Exception as e:
            logger.error(f"保存频道失败: {str(e)}")

    def get_channel_info(self, url: str) -> Dict:
        """获取频道信息"""
        try:
            # 使用session进行请求
            session = M3UParser.setup_requests_session()
            response = session.get(url, timeout=15)
            response.raise_for_status()
            content = response.text

            # 确保内容是UTF-8编码
            if response.encoding is None:
                content = response.content.decode('utf-8', errors='ignore')

            # 解析M3U8内容
            playlist = m3u8.loads(content)
            
            # 提取频道信息
            channel_info = {
                'name': 'Unknown',
                'segments_count': len(playlist.segments) if playlist.segments else 0,
                'duration': sum(float(seg.duration) for seg in playlist.segments) if playlist.segments else 0,
                'resolution': self._get_resolution(playlist),
                'last_check': datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            }

            # 尝试从EXTINF获取频道名称
            for line in content.split('\n'):
                if line.startswith('#EXTINF:'):
                    try:
                        channel_info['name'] = line.split(',', 1)[1].strip()
                        break
                    except:
                        pass

            return channel_info

        except requests.exceptions.Timeout:
            return {
                'name': 'Unknown',
                'segments_count': 0,
                'duration': 0,
                'resolution': 'Unknown',
                'last_check': 'Error',
                'error': '请求超时'
            }
        except requests.exceptions.SSLError:
            return {
                'name': 'Unknown',
                'segments_count': 0,
                'duration': 0,
                'resolution': 'Unknown',
                'last_check': 'Error',
                'error': 'SSL证书验证失败'
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

    def _get_resolution(self, playlist: m3u8.M3U8) -> str:
        """获取播放列表的分辨率"""
        if playlist.playlists:
            # 获取最高分辨率
            max_res = max(
                (p.stream_info.resolution for p in playlist.playlists 
                 if p.stream_info and p.stream_info.resolution),
                default=None
            )
            if max_res:
                return f"{max_res[0]}x{max_res[1]}"
        return 'Unknown'

    def add_channel(self, channel: Channel, skip_check: bool = True) -> bool:
        """添加新频道（跳过检查）"""
        try:
            # 检查是否已存在
            if channel in self.channels:
                logger.info(f"频道已存在: {channel.name}")
                return False

            # 添加频道
            self.channels.add(channel)
            channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            logger.info(f"成功添加频道: {channel.name}")
            return True

        except Exception as e:
            logger.error(f"添加频道失败: {str(e)}")
            return False

    def add_m3u_content(self, url: str) -> Dict[str, any]:
        """添加M3U内容（跳过检查）"""
        try:
            logger.info(f"开始加载M3U内容: {url}")
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            content = response.text

            if M3UParser.is_m3u_content(content):
                # 解析M3U列表
                new_channels = self._parse_m3u_content(content)
                results = {
                    'success': True,
                    'total': len(new_channels),
                    'added': 0,
                    'skipped': 0,
                    'details': []
                }

                # 处理每个频道
                for channel in new_channels:
                    try:
                        if self.add_channel(channel, skip_check=True):
                            results['added'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'added',
                                'group': channel.group_title
                            })
                        else:
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '频道已存在',
                                'group': channel.group_title
                            })
                    except Exception as e:
                        logger.error(f"处理频道失败: {str(e)}")

                # 保存配置
                self.save_channels()
                return results

            else:
                # 单个M3U8地址的处理
                channel = Channel(url=url)
                success = self.add_channel(channel, skip_check=True)
                return {
                    'success': success,
                    'total': 1,
                    'added': 1 if success else 0,
                    'skipped': 0 if success else 1,
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
                'total': 0,
                'added': 0,
                'skipped': 0
            }

    def _parse_m3u_content(self, content: str) -> List[Channel]:
        """解析M3U内容（优化分组和去重）"""
        channels = []
        current_channel = None
        
        for line in content.splitlines():
            line = line.strip()
            if not line:
                continue

            if line.startswith('#EXTINF:'):
                try:
                    # 提取标识信息
                    tvg_name = (M3UParser.extract_attribute(line, 'tvg-id') or 
                              M3UParser.extract_attribute(line, 'tvg-name'))
                    tvg_logo = M3UParser.extract_attribute(line, 'tvg-logo')
                    group_title = M3UParser.extract_attribute(line, 'group-title')
                    
                    # 提取和处理频道名称
                    name = line.split(',')[-1].strip()
                    
                    # 清理频道名称中的特殊字符
                    name = self._clean_channel_name(name)
                    
                    # 标准化分组名称
                    group_title = self._normalize_group_title(group_title)
                    
                    current_channel = {
                        'name': name or tvg_name or "Unknown",
                        'tvg_name': tvg_name,
                        'tvg_logo': tvg_logo,
                        'group_title': group_title
                    }
                except Exception as e:
                    logger.error(f"解析EXTINF失败: {str(e)}")
                    current_channel = None

            elif not line.startswith('#') and current_channel:
                url = line.strip()
                if url.lower().endswith('.m3u8'):
                    try:
                        channel = Channel(
                            url=url,
                            name=current_channel['name'],
                            tvg_name=current_channel['tvg_name'],
                            tvg_logo=current_channel['tvg_logo'],
                            group_title=current_channel['group_title']
                        )
                        channels.append(channel)
                    except Exception as e:
                        logger.error(f"创建频道失败: {str(e)}")
                current_channel = None

        return self._remove_duplicates(channels)

    def _clean_channel_name(self, name: str) -> str:
        """清理和标准化频道名称"""
        # 移除常见的后缀
        suffixes = ['HD', 'SD', '高清', '标清', '超清', '蓝光', '4K']
        name = name.strip()
        for suffix in suffixes:
            if name.upper().endswith(suffix.upper()):
                name = name[:-len(suffix)].strip()
        
        # 移除特殊字符
        name = re.sub(r'[\_\-\+\s]+', ' ', name)
        return name.strip()

    def _normalize_group_title(self, group_title: str) -> str:
        """标准化分组名称"""
        if not group_title:
            return "未分类"
            
        # 常见分组名称映射
        group_mapping = {
            r'(央视|CCTV)': '央视频道',
            r'(卫视|卫视频道)': '卫视频道',
            r'(港|澳|台|港澳台|港台)': '港澳台频道',
            r'(少儿|少女|动画|卡通)': '少儿频道',
            r'(新闻|资讯)': '新闻频道',
            r'(体育|运动)': '体育频道',
            r'(影视|电影|剧场)': '影视频道',
            r'(纪录|记录|探索)': '纪录频道'
        }
        
        group_title = group_title.strip()
        for pattern, replacement in group_mapping.items():
            if re.search(pattern, group_title, re.IGNORECASE):
                return replacement
                
        return group_title

    def _remove_duplicates(self, channels: List[Channel]) -> List[Channel]:
        """移除重复频道，保留最佳URL"""
        # 按名称和分组分类频道
        channel_groups = {}
        for channel in channels:
            key = (channel.name, channel.group_title)
            if key not in channel_groups:
                channel_groups[key] = []
            channel_groups[key].append(channel)
        
        # 从每组中选择最佳URL
        unique_channels = []
        for channels_list in channel_groups.values():
            # 优先选择带有tvg_name的频道
            best_channel = max(channels_list, key=lambda x: (
                bool(x.tvg_name),  # 首先考虑是否有tvg_name
                bool(x.tvg_logo),  # 其次考虑是否有logo
                len(x.url)         # 最后考虑URL长度（偏好较短的URL）
            ))
            unique_channels.append(best_channel)
        
        return unique_channels

    def add_channel(self, channel: Channel, skip_check: bool = True) -> bool:
        """添加新频道（跳过检查）"""
        try:
            # 检查是否已存在
            if channel in self.channels:
                logger.info(f"频道已存在: {channel.name}")
                return False

            # 添加频道
            self.channels.add(channel)
            channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            logger.info(f"成功添加频道: {channel.name}")
            return True

        except Exception as e:
            logger.error(f"添加频道失败: {str(e)}")
            return False

    def add_m3u_content(self, url: str) -> Dict[str, any]:
        """添加M3U内容（跳过检查）"""
        try:
            logger.info(f"开始加载M3U内容: {url}")
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            content = response.text

            if M3UParser.is_m3u_content(content):
                # 解析M3U列表
                new_channels = self._parse_m3u_content(content)
                results = {
                    'success': True,
                    'total': len(new_channels),
                    'added': 0,
                    'skipped': 0,
                    'details': []
                }

                # 处理每个频道
                for channel in new_channels:
                    try:
                        if self.add_channel(channel, skip_check=True):
                            results['added'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'added',
                                'group': channel.group_title
                            })
                        else:
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '频道已存在',
                                'group': channel.group_title
                            })
                    except Exception as e:
                        logger.error(f"处理频道失败: {str(e)}")

                # 保存配置
                self.save_channels()
                return results

            else:
                # 单个M3U8地址的处理
                channel = Channel(url=url)
                success = self.add_channel(channel, skip_check=True)
                return {
                    'success': success,
                    'total': 1,
                    'added': 1 if success else 0,
                    'skipped': 0 if success else 1,
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
                'total': 0,
                'added': 0,
                'skipped': 0
            }

    def _parse_m3u_content(self, content: str) -> List[Channel]:
        """解析M3U内容（优化分组和去重）"""
        channels = []
        current_channel = None
        
        for line in content.splitlines():
            line = line.strip()
            if not line:
                continue

            if line.startswith('#EXTINF:'):
                try:
                    # 提取标识信息
                    tvg_name = (M3UParser.extract_attribute(line, 'tvg-id') or 
                              M3UParser.extract_attribute(line, 'tvg-name'))
                    tvg_logo = M3UParser.extract_attribute(line, 'tvg-logo')
                    group_title = M3UParser.extract_attribute(line, 'group-title')
                    
                    # 提取和处理频道名称
                    name = line.split(',')[-1].strip()
                    
                    # 清理频道名称中的特殊字符
                    name = self._clean_channel_name(name)
                    
                    # 标准化分组名称
                    group_title = self._normalize_group_title(group_title)
                    
                    current_channel = {
                        'name': name or tvg_name or "Unknown",
                        'tvg_name': tvg_name,
                        'tvg_logo': tvg_logo,
                        'group_title': group_title
                    }
                except Exception as e:
                    logger.error(f"解析EXTINF失败: {str(e)}")
                    current_channel = None

            elif not line.startswith('#') and current_channel:
                url = line.strip()
                if url.lower().endswith('.m3u8'):
                    try:
                        channel = Channel(
                            url=url,
                            name=current_channel['name'],
                            tvg_name=current_channel['tvg_name'],
                            tvg_logo=current_channel['tvg_logo'],
                            group_title=current_channel['group_title']
                        )
                        channels.append(channel)
                    except Exception as e:
                        logger.error(f"创建频道失败: {str(e)}")
                current_channel = None

        return self._remove_duplicates(channels)

    def _clean_channel_name(self, name: str) -> str:
        """清理和标准化频道名称"""
        # 移除常见的后缀
        suffixes = ['HD', 'SD', '高清', '标清', '超清', '蓝光', '4K']
        name = name.strip()
        for suffix in suffixes:
            if name.upper().endswith(suffix.upper()):
                name = name[:-len(suffix)].strip()
        
        # 移除特殊字符
        name = re.sub(r'[\_\-\+\s]+', ' ', name)
        return name.strip()

    def _normalize_group_title(self, group_title: str) -> str:
        """标准化分组名称"""
        if not group_title:
            return "未分类"
            
        # 常见分组名称映射
        group_mapping = {
            r'(央视|CCTV)': '央视频道',
            r'(卫视|卫视频道)': '卫视频道',
            r'(港|澳|台|港澳台|港台)': '港澳台频道',
            r'(少儿|少女|动画|卡通)': '少儿频道',
            r'(新闻|资讯)': '新闻频道',
            r'(体育|运动)': '体育频道',
            r'(影视|电影|剧场)': '影视频道',
            r'(纪录|记录|探索)': '纪录频道'
        }
        
        group_title = group_title.strip()
        for pattern, replacement in group_mapping.items():
            if re.search(pattern, group_title, re.IGNORECASE):
                return replacement
                
        return group_title

    def _remove_duplicates(self, channels: List[Channel]) -> List[Channel]:
        """移除重复频道，保留最佳URL"""
        # 按名称和分组分类频道
        channel_groups = {}
        for channel in channels:
            key = (channel.name, channel.group_title)
            if key not in channel_groups:
                channel_groups[key] = []
            channel_groups[key].append(channel)
        
        # 从每组中选择最佳URL
        unique_channels = []
        for channels_list in channel_groups.values():
            # 优先选择带有tvg_name的频道
            best_channel = max(channels_list, key=lambda x: (
                bool(x.tvg_name),  # 首先考虑是否有tvg_name
                bool(x.tvg_logo),  # 其次考虑是否有logo
                len(x.url)         # 最后考虑URL长度（偏好较短的URL）
            ))
            unique_channels.append(best_channel)
        
        return unique_channels

    def add_channel(self, channel: Channel, skip_check: bool = True) -> bool:
        """添加新频道（跳过检查）"""
        try:
            # 检查是否已存在
            if channel in self.channels:
                logger.info(f"频道已存在: {channel.name}")
                return False

            # 添加频道
            self.channels.add(channel)
            channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            logger.info(f"成功添加频道: {channel.name}")
            return True

        except Exception as e:
            logger.error(f"添加频道失败: {str(e)}")
            return False

    def add_m3u_content(self, url: str) -> Dict[str, any]:
        """添加M3U内容（跳过检查）"""
        try:
            logger.info(f"开始加载M3U内容: {url}")
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            content = response.text

            if M3UParser.is_m3u_content(content):
                # 解析M3U列表
                new_channels = self._parse_m3u_content(content)
                results = {
                    'success': True,
                    'total': len(new_channels),
                    'added': 0,
                    'skipped': 0,
                    'details': []
                }

                # 处理每个频道
                for channel in new_channels:
                    try:
                        if self.add_channel(channel, skip_check=True):
                            results['added'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'added',
                                'group': channel.group_title
                            })
                        else:
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '频道已存在',
                                'group': channel.group_title
                            })
                    except Exception as e:
                        logger.error(f"处理频道失败: {str(e)}")

                # 保存配置
                self.save_channels()
                return results

            else:
                # 单个M3U8地址的处理
                channel = Channel(url=url)
                success = self.add_channel(channel, skip_check=True)
                return {
                    'success': success,
                    'total': 1,
                    'added': 1 if success else 0,
                    'skipped': 0 if success else 1,
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
                'total': 0,
                'added': 0,
                'skipped': 0
            }

    def _parse_m3u_content(self, content: str) -> List[Channel]:
        """解析M3U内容（优化分组和去重）"""
        channels = []
        current_channel = None
        
        for line in content.splitlines():
            line = line.strip()
            if not line:
                continue

            if line.startswith('#EXTINF:'):
                try:
                    # 提取标识信息
                    tvg_name = (M3UParser.extract_attribute(line, 'tvg-id') or 
                              M3UParser.extract_attribute(line, 'tvg-name'))
                    tvg_logo = M3UParser.extract_attribute(line, 'tvg-logo')
                    group_title = M3UParser.extract_attribute(line, 'group-title')
                    
                    # 提取和处理频道名称
                    name = line.split(',')[-1].strip()
                    
                    # 清理频道名称中的特殊字符
                    name = self._clean_channel_name(name)
                    
                    # 标准化分组名称
                    group_title = self._normalize_group_title(group_title)
                    
                    current_channel = {
                        'name': name or tvg_name or "Unknown",
                        'tvg_name': tvg_name,
                        'tvg_logo': tvg_logo,
                        'group_title': group_title
                    }
                except Exception as e:
                    logger.error(f"解析EXTINF失败: {str(e)}")
                    current_channel = None

            elif not line.startswith('#') and current_channel:
                url = line.strip()
                if url.lower().endswith('.m3u8'):
                    try:
                        channel = Channel(
                            url=url,
                            name=current_channel['name'],
                            tvg_name=current_channel['tvg_name'],
                            tvg_logo=current_channel['tvg_logo'],
                            group_title=current_channel['group_title']
                        )
                        channels.append(channel)
                    except Exception as e:
                        logger.error(f"创建频道失败: {str(e)}")
                current_channel = None

        return self._remove_duplicates(channels)

    def _clean_channel_name(self, name: str) -> str:
        """清理和标准化频道名称"""
        # 移除常见的后缀
        suffixes = ['HD', 'SD', '高清', '标清', '超清', '蓝光', '4K']
        name = name.strip()
        for suffix in suffixes:
            if name.upper().endswith(suffix.upper()):
                name = name[:-len(suffix)].strip()
        
        # 移除特殊字符
        name = re.sub(r'[\_\-\+\s]+', ' ', name)
        return name.strip()

    def _normalize_group_title(self, group_title: str) -> str:
        """标准化分组名称"""
        if not group_title:
            return "未分类"
            
        # 常见分组名称映射
        group_mapping = {
            r'(央视|CCTV)': '央视频道',
            r'(卫视|卫视频道)': '卫视频道',
            r'(港|澳|台|港澳台|港台)': '港澳台频道',
            r'(少儿|少女|动画|卡通)': '少儿频道',
            r'(新闻|资讯)': '新闻频道',
            r'(体育|运动)': '体育频道',
            r'(影视|电影|剧场)': '影视频道',
            r'(纪录|记录|探索)': '纪录频道'
        }
        
        group_title = group_title.strip()
        for pattern, replacement in group_mapping.items():
            if re.search(pattern, group_title, re.IGNORECASE):
                return replacement
                
        return group_title

    def _remove_duplicates(self, channels: List[Channel]) -> List[Channel]:
        """移除重复频道，保留最佳URL"""
        # 按名称和分组分类频道
        channel_groups = {}
        for channel in channels:
            key = (channel.name, channel.group_title)
            if key not in channel_groups:
                channel_groups[key] = []
            channel_groups[key].append(channel)
        
        # 从每组中选择最佳URL
        unique_channels = []
        for channels_list in channel_groups.values():
            # 优先选择带有tvg_name的频道
            best_channel = max(channels_list, key=lambda x: (
                bool(x.tvg_name),  # 首先考虑是否有tvg_name
                bool(x.tvg_logo),  # 其次考虑是否有logo
                len(x.url)         # 最后考虑URL长度（偏好较短的URL）
            ))
            unique_channels.append(best_channel)
        
        return unique_channels

    def add_channel(self, channel: Channel, skip_check: bool = True) -> bool:
        """添加新频道（跳过检查）"""
        try:
            # 检查是否已存在
            if channel in self.channels:
                logger.info(f"频道已存在: {channel.name}")
                return False

            # 添加频道
            self.channels.add(channel)
            channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
            logger.info(f"成功添加频道: {channel.name}")
            return True

        except Exception as e:
            logger.error(f"添加频道失败: {str(e)}")
            return False

    def add_m3u_content(self, url: str) -> Dict[str, any]:
        """添加M3U内容（跳过检查）"""
        try:
            logger.info(f"开始加载M3U内容: {url}")
            response = requests.get(url, timeout=10)
            response.raise_for_status()
            content = response.text

            if M3UParser.is_m3u_content(content):
                # 解析M3U列表
                new_channels = self._parse_m3u_content(content)
                results = {
                    'success': True,
                    'total': len(new_channels),
                    'added': 0,
                    'skipped': 0,
                    'details': []
                }

                # 处理每个频道
                for channel in new_channels:
                    try:
                        if self.add_channel(channel, skip_check=True):
                            results['added'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'added',
                                'group': channel.group_title
                            })
                        else:
                            results['skipped'] += 1
                            results['details'].append({
                                'url': channel.url,
                                'name': channel.name,
                                'status': 'skipped',
                                'reason': '频道已存在',
                                'group': channel.group_title
                            })
                    except Exception as e:
                        logger.error(f"处理频道失败: {str(e)}")

                # 保存配置
                self.save_channels()
                return results

            else:
                # 单个M3U8地址的处理
                channel = Channel(url=url)
                success = self.add_channel(channel, skip_check=True)
                return {
                    'success': success,
                    'total': 1,
                    'added': 1 if success else 0,
                    'skipped': 0 if success else 1,
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
                'total': 0,
                'added': 0,
                'skipped': 0
            }

    def _parse_m3u_content(self, content: str) -> List[Channel]:
        """解析M3U内容（优化分组和去重）"""
        channels = []
        current_channel = None
        
        for line in content.splitlines():
            line = line.strip()
            if not line:
                continue

            if line.startswith('#EXTINF:'):
                try:
                    # 提取标识信息
                    tvg_name = (M3UParser.extract_attribute(line, 'tvg-id') or 
                              M3UParser.extract_attribute(line, 'tvg-name'))
                    tvg_logo = M3UParser.extract_attribute(line, 'tvg-logo')
                    group_title = M3UParser.extract_attribute(line, 'group-title')
                    
                    # 提取和处理频道名称
                    name = line.split(',')[-1].strip()
                    
                    # 清理频道名称中的特殊字符
                    name = self._clean_channel_name(name)
                    
                    # 标准化分组名称
                    group_title = self._normalize_group_title(group_title)
                    
                    current_channel = {
                        'name': name or tvg_name or "Unknown",
                        'tvg_name': tvg_name,
                        'tvg_logo': tvg_logo,
                        'group_title': group_title
                    }
                except Exception as e:
                    logger.error(f"解析EXTINF失败: {str(e)}")
                    current_channel = None

            elif not line.startswith('#') and current_channel:
                url = line.strip()
                if url.lower().endswith('.m3u8'):
                    try:
                        channel = Channel(
                            url=url,
                            name=current_channel['name'],
                            tvg_name=current_channel['tvg_name'],
                            tvg_logo=current_channel['tvg_logo'],
                            group_title=current_channel['group_title']
                        )
                        channels.append(channel)
                    except Exception as e:
                        logger.error(f"创建频道失败: {str(e)}")
                current_channel = None

        return self._remove_duplicates(channels)

    def _clean_channel_name(self, name: str) -> str:
        """清理和标准化频道名称"""
        # 移除常见的后缀
        suffixes = ['HD', 'SD', '高清', '标清', '超清', '蓝光', '4K']
        name = name.strip()
        for suffix in suffixes:
            if name.upper().endswith(suffix.upper()):
                name = name[:-len(suffix)].strip()
        
        # 移除特殊字符
        name = re.sub(r'[\_\-\+\s]+', ' ', name)
        return name.strip()

    def _normalize_group_title(self, group_title: str) -> str:
        """标准化分组名称"""
        if not group_title:
            return "未分类"
            
        # 常见分组名称映射
        group_mapping = {
            r'(央视|CCTV)': '央视频道',
            r'(卫视|卫视频道)': '卫视频道',
            r'(港|澳|台|港澳台|港台)': '港澳台频道',
            r'(少儿|少女|动画|卡通)': '少儿频道',
            r'(新闻|资讯)': '新闻频道',
            r'(体育|运动)': '体育频道',
            r'(影视|电影|剧场)': '影视频道',
            r'(纪录|记录|探索)': '纪录频道'
        }
        
        group_title = group_title.strip()
        for pattern, replacement in group_mapping.items():
            if re.search(pattern, group_title, re.IGNORECASE):
                return replacement
                
        return group_title

    def _remove_duplicates(self, channels: List[Channel]) -> List[Channel]:
        """移除重复频道，保留最佳URL"""
        # 按名称和分组分类频道
        channel_groups = {}
        for channel in channels:
            key = (channel.name, channel.group_title)
            if key not in channel_groups:
                channel_groups[key] = []
            channel_groups[key].append(channel)
        
        # 从每组中选择最佳URL
        unique_channels = []
        for channels_list in channel_groups.values():
            # 优先选择带有tvg_name的频道
            best_channel = max(channels_list, key=lambda x: (
                bool(x.tvg_name),  # 首先考虑是否有tvg_name
                bool(x.tvg_logo),  # 其次考虑是否有logo
                len(x.url)         # 最后考虑URL长度（偏好较短的URL）
            ))
            unique_channels.append(best_channel)
        
        return unique_channels

    def _standardize_channel_name(self, name: str) -> str:
        """标准化频道名称，处理常见的变体"""
        if not name:
            return "Unknown"
            
        # 移除所有空白字符
        name = ''.join(name.split())
        
        # 统一大小写
        name = name.upper()
        
        # 常见频道名称映射
        name_mapping = {
            r'CCTV(\d+)': r'CCTV-\1',  # CCTV1 -> CCTV-1
            r'CCTV-(\d+)HD': r'CCTV-\1',  # CCTV-1HD -> CCTV-1
            r'CCTV(\d+)高清': r'CCTV-\1',  # CCTV1高清 -> CCTV-1
            r'([东南西北]方)卫视': r'\1卫视',  # 东方卫视
            r'([^卫视]+)(卫视)?(高清|HD)': r'\1卫视',  # 湖南高清 -> 湖南卫视
            r'(.*?)(频道|台|电视台)': r'\1',  # 移除"频道"、"台"等后缀
            r'凤凰(中文|资讯|卫视)': r'凤凰卫视',  # 统一凤凰卫视相关频道
            r'翡翠台': r'TVB翡翠台',  # 统一TVB相关频道名称
            r'无线新闻': r'TVB新闻台',
            r'CHANNELV': r'CHANNELV音乐台',
        }
        
        # 应用映射规则
        for pattern, replacement in name_mapping.items():
            name = re.sub(pattern, replacement, name)
            
        return name.strip()

    def _remove_duplicates(self, channels: List[Channel]) -> List[Channel]:
        """增强的去重逻辑，使用标准化名称"""
        # 按标准化名称和分组对频道进行分类
        channel_groups = {}
        for channel in channels:
            std_name = self._standardize_channel_name(channel.name)
            key = (std_name, channel.group_title)
            if key not in channel_groups:
                channel_groups[key] = []
            channel_groups[key].append(channel)
        
        # 从每组中选择最佳频道
        unique_channels = []
        for channels_list in channel_groups.values():
            if len(channels_list) > 1:
                # 记录发现的重复频道
                logger.info(f"发现重复频道: {channels_list[0].name}, 共{len(channels_list)}个变体")
                
            # 选择最佳频道（优先选择信息最完整的）
            best_channel = max(channels_list, key=lambda x: (
                bool(x.tvg_name),  # 首先考虑是否有tvg_name
                bool(x.tvg_logo),  # 其次考虑是否有logo
                len(x.name),       # 频道名称长度
                -len(x.url)        # URL越短越好
            ))
            unique_channels.append(best_channel)
        
        return unique_channels

    def generate_playlist(self) -> str:
        """生成M3U8播放列表"""
        playlist = "#EXTM3U\n"
        sorted_channels = sorted(
            self.channels,
            key=lambda x: (x.group_title or "", x.name or "")
        )
        for channel in sorted_channels:
            playlist += channel.to_m3u8_entry()
        return playlist

    def cleanup_channels(self, auto_repair: bool = True) -> Dict[str, any]:
        """清理和修复频道"""
        results = {
            'total': len(self.channels),
            'checked': 0,
            'available': 0,
            'repaired': 0,
            'removed': 0,
            'details': []
        }

        channels_to_remove = set()
        channels_to_update = {}

        for channel in self.channels:
            results['checked'] += 1
            logger.info(f"检查频道 [{results['checked']}/{results['total']}]: {channel.name}")

            if URLChecker.check_availability(channel.url):
                results['available'] += 1
                continue

            if auto_repair:
                fixed = False
                
                # 1. 检查重定向
                new_url = URLChecker.get_redirected_url(channel.url)
                if new_url != channel.url and URLChecker.check_availability(new_url):
                    channels_to_update[channel] = new_url
                    fixed = True
                
                # 2. 尝试HTTP/HTTPS切换
                if not fixed:
                    if channel.url.startswith('https://'):
                        alt_url = 'http://' + channel.url[8:]
                    elif channel.url.startswith('http://'):
                        alt_url = 'https://' + channel.url[7:]
                    else:
                        alt_url = None

                    if alt_url and URLChecker.check_availability(alt_url):
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
                    channels_to_remove.add(channel)
                    results['removed'] += 1
                    results['details'].append({
                        'url': channel.url,
                        'name': channel.name,
                        'status': 'removed'
                    })

        # 应用更改
        if auto_repair:
            # 更新URL
            for channel, new_url in channels_to_update.items():
                channel.url = new_url
                channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

            # 移除无效频道
            for channel in channels_to_remove:
                self.channels.remove(channel)

            # 保存更改
            self.save_channels()

        return results

    async def _check_batch_async(self, channels: List[Channel], batch_size: int = 10) -> Dict[str, any]:
        """异步检查一批频道"""
        start_time = time.time()
        total = len(channels)
        results = {
            'total': total,
            'available': 0,
            'unavailable': 0,
            'details': [],
            'performance': {
                'startTime': start_time,
                'concurrency': batch_size,
                'responseTimes': []
            }
        }

        # 分批处理频道
        for i in range(0, total, batch_size):
            batch = channels[i:i + batch_size]
            urls = [ch.url for ch in batch]
            
            # 并发检查这一批频道
            check_start = time.time()
            batch_results = await URLChecker.check_urls_concurrent(urls)
            check_end = time.time()
            
            # 更新结果
            url_results = {r['url']: r for r in batch_results}
            for channel in batch:
                result = url_results.get(channel.url, {'available': False, 'message': '检查超时'})
                if result['available']:
                    results['available'] += 1
                    channel.last_check = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
                else:
                    results['unavailable'] += 1
                
                # 记录响应时间
                try:
                    response_time = float(result['message'].replace('秒', ''))
                    results['performance']['responseTimes'].append(response_time)
                except:
                    pass
                
                results['details'].append({
                    'name': channel.name,
                    'url': channel.url,
                    'status': '正常' if result['available'] else '异常',
                    'message': result['message']
                })

        # 计算性能统计
        end_time = time.time()
        total_time = end_time - start_time
        response_times = results['performance']['responseTimes']
        
        results['performance'].update({
            'endTime': end_time,
            'totalTime': total_time,
            'avgResponseTime': sum(response_times) / len(response_times) if response_times else 0,
            'minResponseTime': min(response_times) if response_times else 0,
            'maxResponseTime': max(response_times) if response_times else 0
        })

        return results

    def batch_check_channels_concurrent(self, batch_size: int = 10) -> Dict[str, any]:
        """并发检查所有频道（包含性能统计）"""
        channels = list(self.channels)
        
        try:
            # 创建事件循环
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
            # 执行异步检查
            results = loop.run_until_complete(
                self._check_batch_async(channels, batch_size)
            )
            
            # 关闭事件循环
            loop.close()
            
            # 保存更新的频道状态
            self.save_channels()
            
            return results
            
        except Exception as e:
            logger.error(f"并发检查失败: {str(e)}")
            return {
                'error': str(e),
                'total': len(channels),
                'available': 0,
                'unavailable': 0,
                'details': []
            }
