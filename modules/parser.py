# -*- coding: utf-8 -*-
"""
M3U解析模块

该模块负责抓取和解析M3U文件，提取频道信息。
"""

import re
import requests
import logging
from urllib.parse import urlparse

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class M3UParser:
    """M3U文件解析器"""
    
    def __init__(self, timeout=30):
        """初始化解析器
        
        Args:
            timeout: 请求超时时间（秒）
        """
        self.timeout = timeout
    
    def fetch_m3u(self, url):
        """从URL获取M3U文件内容
        
        Args:
            url: M3U文件的URL
            
        Returns:
            tuple: (是否成功, 内容或错误消息)
        """
        try:
            response = requests.get(url, timeout=self.timeout)
            if response.status_code != 200:
                return False, f"HTTP错误: {response.status_code}"
            
            # 尝试检测编码
            content = response.content
            try:
                # 尝试UTF-8解码
                content = content.decode('utf-8')
            except UnicodeDecodeError:
                try:
                    # 尝试GBK解码（常见于中文环境）
                    content = content.decode('gbk')
                except UnicodeDecodeError:
                    # 回退到ISO-8859-1（Latin-1）
                    content = content.decode('iso-8859-1')
            
            # 验证是否为有效的M3U文件
            if not content.strip().startswith('#EXTM3U'):
                return False, "无效的M3U文件格式"
            
            return True, content
        except requests.exceptions.Timeout:
            return False, "请求超时"
        except requests.exceptions.ConnectionError:
            return False, "连接错误"
        except Exception as e:
            return False, f"获取M3U文件时出错: {str(e)}"
    
    def parse_m3u(self, content, source_url=None):
        """解析M3U文件内容（增强：支持多地址分割并只保留合法流地址）
        Args:
            content: M3U文件内容
            source_url: 源URL（用于相对路径解析）
        Returns:
            list: 解析后的频道列表
        """
        channels = []
        lines = content.splitlines()
        import re
        i = 0
        while i < len(lines):
            line = lines[i].strip()
            # 跳过空行和注释
            if not line or (line.startswith('#') and not line.startswith('#EXTINF:')):
                i += 1
                continue
            # 解析频道信息行
            if line.startswith('#EXTINF:'):
                # 确保下一行存在且不是以#开头
                if i + 1 < len(lines) and not lines[i + 1].strip().startswith('#'):
                    channel_info = self._parse_extinf(line)
                    channel_url_line = lines[i + 1].strip()
                    # 以 # ; 空格分割，过滤只保留合法流地址
                    parts = re.split(r'[;#\s]+', channel_url_line)
                    valid_urls = [p for p in parts if p.strip() and p.strip().lower().startswith(('http://', 'https://', 'rtmp://'))]
                    # 处理相对URL（只对主url处理）
                    if valid_urls and source_url and not (valid_urls[0].startswith('http://') or valid_urls[0].startswith('https://') or valid_urls[0].startswith('rtmp://')):
                        valid_urls[0] = self._resolve_relative_url(source_url, valid_urls[0])
                    if valid_urls:
                        channel = {
                            'name': channel_info.get('name', '未命名频道'),
                            'url': valid_urls[0],
                            'tvg_id': channel_info.get('tvg-id', ''),
                            'tvg_name': channel_info.get('tvg-name', ''),
                            'tvg_logo': channel_info.get('tvg-logo', ''),
                            'group_title': channel_info.get('group-title', '未分类'),
                            'source': source_url
                        }
                        if len(valid_urls) > 1:
                            channel['sources'] = [{'url': u} for u in valid_urls[1:]]
                        channels.append(channel)
                    # 跳过URL行
                    i += 2
                    continue
            i += 1
        logger.info(f"从M3U文件解析到 {len(channels)} 个频道")
        return channels
    
    def _parse_extinf(self, extinf_line):
        """解析#EXTINF行
        
        Args:
            extinf_line: #EXTINF行内容
            
        Returns:
            dict: 解析后的频道信息
        """
        result = {}
        
        # 提取频道名称
        name_match = re.search(r',(.*?)$', extinf_line)
        if name_match:
            result['name'] = name_match.group(1).strip()
        
        # 提取tvg-id
        tvg_id_match = re.search(r'tvg-id="([^"]*?)"', extinf_line)
        if tvg_id_match:
            result['tvg-id'] = tvg_id_match.group(1)
        
        # 提取tvg-name
        tvg_name_match = re.search(r'tvg-name="([^"]*?)"', extinf_line)
        if tvg_name_match:
            result['tvg-name'] = tvg_name_match.group(1)
        
        # 提取tvg-logo
        tvg_logo_match = re.search(r'tvg-logo="([^"]*?)"', extinf_line)
        if tvg_logo_match:
            result['tvg-logo'] = tvg_logo_match.group(1)
        
        # 提取group-title
        group_title_match = re.search(r'group-title="([^"]*?)"', extinf_line)
        if group_title_match:
            result['group-title'] = group_title_match.group(1)
        
        return result
    
    def _resolve_relative_url(self, base_url, relative_url):
        """解析相对URL
        
        Args:
            base_url: 基础URL
            relative_url: 相对URL
            
        Returns:
            str: 解析后的完整URL
        """
        parsed_base = urlparse(base_url)
        base_path = '/'.join(parsed_base.path.split('/')[:-1]) + '/'
        
        if relative_url.startswith('/'):
            # 绝对路径
            return f"{parsed_base.scheme}://{parsed_base.netloc}{relative_url}"
        else:
            # 相对路径
            return f"{parsed_base.scheme}://{parsed_base.netloc}{base_path}{relative_url}"