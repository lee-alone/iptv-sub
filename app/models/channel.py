from typing import Dict
from datetime import datetime

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
        # 只比较名称和分组，不检查URL
        return (self.name == other.name and 
                self.group_title == other.group_title)

    def __hash__(self):
        # 使用名称和分组生成哈希值
        return hash((self.name, self.group_title))

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

    def to_dict(self) -> Dict:
        """将频道转换为字典格式"""
        return {
            'url': self.url,
            'name': self.name,
            'tvg_name': self.tvg_name,
            'tvg_logo': self.tvg_logo,
            'group_title': self.group_title,
            'last_check': self.last_check,
            'resolution': self.resolution,
            'segments_count': self.segments_count,
            'duration': self.duration
        }

    @classmethod
    def from_dict(cls, data: Dict) -> 'Channel':
        """从字典创建频道对象"""
        channel = cls(
            url=data['url'],
            name=data['name'],
            tvg_name=data.get('tvg_name'),
            tvg_logo=data.get('tvg_logo'),
            group_title=data.get('group_title')
        )
        channel.last_check = data.get('last_check')
        channel.resolution = data.get('resolution', 'Unknown')
        channel.segments_count = data.get('segments_count', 0)
        channel.duration = data.get('duration', 0)
        return channel
