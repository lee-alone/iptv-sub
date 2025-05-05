# -*- coding: utf-8 -*-
"""
频道聚合模块

该模块负责聚合来自多个M3U源的频道，识别并合并重复频道。
"""

import os
import json
import logging
from difflib import SequenceMatcher

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class ChannelAggregator:
    """频道聚合器"""
    
    def __init__(self, data_dir='data'):
        """初始化聚合器
        
        Args:
            data_dir: 数据存储目录
        """
        self.data_dir = data_dir
        self.channels_file = os.path.join(data_dir, 'channels.json')
        self.channels = []
        
        # 确保数据目录存在
        if not os.path.exists(data_dir):
            os.makedirs(data_dir)
        
        # 加载已有频道数据
        self.load_channels()
    
    def load_channels(self):
        """从文件加载频道列表"""
        if os.path.exists(self.channels_file):
            try:
                with open(self.channels_file, 'r', encoding='utf-8') as f:
                    self.channels = json.load(f)
                logger.info(f"已加载 {len(self.channels)} 个频道")
            except Exception as e:
                logger.error(f"加载频道数据时出错: {str(e)}")
                self.channels = []
        else:
            logger.info("频道数据文件不存在，创建新的频道列表")
            self.channels = []
    
    def save_channels(self):
        """保存频道列表到文件"""
        try:
            with open(self.channels_file, 'w', encoding='utf-8') as f:
                json.dump(self.channels, f, ensure_ascii=False, indent=2)
            logger.info(f"已保存 {len(self.channels)} 个频道")
            return True
        except Exception as e:
            logger.error(f"保存频道数据时出错: {str(e)}")
            return False
    
    def aggregate_channels(self, new_channels, match_by='name', similarity_threshold=0.85):
        """聚合新的频道数据
        
        Args:
            new_channels: 新的频道列表
            match_by: 匹配方式，'name'、'tvg_id'或'both'
            similarity_threshold: 名称相似度阈值（0-1之间）
            
        Returns:
            tuple: (聚合后的频道数, 新增频道数, 更新频道数)
        """
        if not self.channels:
            # 如果当前没有频道，直接使用新频道列表
            self.channels = new_channels
            self.save_channels()
            return len(self.channels), len(self.channels), 0
        
        # 统计数据
        initial_count = len(self.channels)
        updated_count = 0
        
        # 遍历新频道
        for new_channel in new_channels:
            # 查找匹配的现有频道
            existing_channel = self._find_matching_channel(new_channel, match_by, similarity_threshold)
            
            if existing_channel:
                # 更新现有频道
                self._update_channel(existing_channel, new_channel)
                updated_count += 1
            else:
                # 添加新频道
                self.channels.append(new_channel)
        
        # 保存更改
        self.save_channels()
        
        # 计算新增频道数
        added_count = len(self.channels) - initial_count
        
        return len(self.channels), added_count, updated_count
    
    def _find_matching_channel(self, new_channel, match_by, similarity_threshold):
        """查找匹配的现有频道
        
        Args:
            new_channel: 新频道
            match_by: 匹配方式
            similarity_threshold: 名称相似度阈值
            
        Returns:
            dict: 匹配的频道，如果未找到则返回None
        """
        for channel in self.channels:
            # 按tvg_id匹配
            if match_by in ['tvg_id', 'both'] and new_channel['tvg_id'] and channel['tvg_id'] == new_channel['tvg_id']:
                return channel
            
            # 按名称匹配
            if match_by in ['name', 'both']:
                # 精确匹配
                if channel['name'].lower() == new_channel['name'].lower():
                    return channel
                
                # 模糊匹配
                similarity = self._name_similarity(channel['name'], new_channel['name'])
                if similarity >= similarity_threshold:
                    return channel
        
        return None
    
    def _update_channel(self, existing_channel, new_channel):
        """更新现有频道信息
        
        Args:
            existing_channel: 现有频道
            new_channel: 新频道
        """
        # 确保sources字段存在
        if 'sources' not in existing_channel:
            existing_channel['sources'] = []
            if 'url' in existing_channel and 'source' in existing_channel:
                # 将原始URL添加到sources
                existing_channel['sources'].append({
                    'url': existing_channel['url'],
                    'source': existing_channel['source']
                })
        
        # 添加新的源URL（如果不存在）
        source_exists = False
        for source in existing_channel['sources']:
            if source['url'] == new_channel['url']:
                source_exists = True
                break
        
        if not source_exists:
            existing_channel['sources'].append({
                'url': new_channel['url'],
                'source': new_channel['source']
            })
        
        # 更新其他元数据（如果新频道有更完整的信息）
        if not existing_channel['tvg_id'] and new_channel['tvg_id']:
            existing_channel['tvg_id'] = new_channel['tvg_id']
        
        if not existing_channel['tvg_logo'] and new_channel['tvg_logo']:
            existing_channel['tvg_logo'] = new_channel['tvg_logo']
        
        # 保留原始URL作为主URL
        # 可以根据需要实现URL优先级逻辑
    
    def _name_similarity(self, name1, name2):
        """计算两个频道名称的相似度
        
        Args:
            name1: 第一个名称
            name2: 第二个名称
            
        Returns:
            float: 相似度（0-1之间）
        """
        # 转换为小写并去除空格
        name1 = self._normalize_name(name1)
        name2 = self._normalize_name(name2)
        
        # 使用序列匹配器计算相似度
        return SequenceMatcher(None, name1, name2).ratio()
    
    def _normalize_name(self, name):
        """规范化频道名称
        
        Args:
            name: 原始名称
            
        Returns:
            str: 规范化后的名称
        """
        # 转换为小写
        name = name.lower()
        
        # 移除常见的频道后缀
        suffixes = ['hd', 'sd', 'fhd', '4k', 'uhd', 'h264', 'h265', 'hevc']
        for suffix in suffixes:
            if name.endswith(f" {suffix}"):
                name = name[:-len(suffix)-1]
        
        # 移除特殊字符
        name = ''.join(c for c in name if c.isalnum() or c.isspace())
        
        # 移除多余空格
        name = ' '.join(name.split())
        
        return name
    
    def get_all_channels(self):
        """获取所有频道
        
        Returns:
            list: 频道列表
        """
        return self.channels
    
    def get_channels_by_group(self, group_title):
        """获取指定分组的频道
        
        Args:
            group_title: 分组名称
            
        Returns:
            list: 频道列表
        """
        return [channel for channel in self.channels if channel['group_title'] == group_title]
    
    def get_channel_groups(self):
        """获取所有频道分组
        
        Returns:
            list: 分组名称列表
        """
        groups = set()
        for channel in self.channels:
            groups.add(channel['group_title'])
        return sorted(list(groups))
    
    def clear_channels(self):
        """清空频道列表"""
        self.channels = []
        self.save_channels()
        return True