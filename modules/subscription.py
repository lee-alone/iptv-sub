# -*- coding: utf-8 -*-
"""
订阅源管理模块

该模块负责管理IPTV M3U订阅源，包括添加、显示、删除和编辑功能。
"""

import os
import json
import requests
from datetime import datetime
import logging

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class SubscriptionManager:
    """订阅源管理类"""
    
    def __init__(self, data_dir='data'):
        """初始化订阅源管理器
        
        Args:
            data_dir: 数据存储目录
        """
        self.data_dir = data_dir
        self.subscriptions_file = os.path.join(data_dir, 'subscriptions.json')
        self.subscriptions = []
        
        # 确保数据目录存在
        if not os.path.exists(data_dir):
            os.makedirs(data_dir)
            
        # 加载订阅源
        self.load_subscriptions()
    
    def load_subscriptions(self):
        """从文件加载订阅源列表"""
        if os.path.exists(self.subscriptions_file):
            try:
                with open(self.subscriptions_file, 'r', encoding='utf-8') as f:
                    self.subscriptions = json.load(f)
                logger.info(f"已加载 {len(self.subscriptions)} 个订阅源")
            except Exception as e:
                logger.error(f"加载订阅源时出错: {str(e)}")
                self.subscriptions = []
        else:
            logger.info("订阅源文件不存在，创建新的订阅列表")
            self.subscriptions = []
            self.save_subscriptions()
    
    def save_subscriptions(self):
        """保存订阅源列表到文件"""
        try:
            with open(self.subscriptions_file, 'w', encoding='utf-8') as f:
                json.dump(self.subscriptions, f, ensure_ascii=False, indent=2)
            logger.info(f"已保存 {len(self.subscriptions)} 个订阅源")
            return True
        except Exception as e:
            logger.error(f"保存订阅源时出错: {str(e)}")
            return False
    
    def validate_url(self, url):
        """验证URL格式和可访问性
        
        Args:
            url: 要验证的URL
            
        Returns:
            tuple: (是否有效, 错误消息)
        """        # 检查URL格式
        if not url.startswith(('http://', 'https://')):
            return False, "URL必须以http://或https://开头"
        
        # 尝试访问URL
        try:
            response = requests.head(url, timeout=10)
            if response.status_code != 200:
                return False, f"URL返回状态码: {response.status_code}"
        except requests.exceptions.RequestException as e:
            return False, f"无法访问URL: {str(e)}"
        
        return True, ""
    
    def add_subscription(self, url, name=""):
        """添加新的订阅源
        
        Args:
            url: 订阅源URL
            name: 订阅源名称（可选）
            
        Returns:
            tuple: (是否成功, 消息)
        """
        # 检查URL是否已存在
        for sub in self.subscriptions:
            if sub['url'] == url:
                return False, "该URL已存在于订阅列表中"
        
        # 验证URL
        is_valid, error_msg = self.validate_url(url)
        if not is_valid:
            return False, error_msg
        
        # 创建新的订阅记录
        subscription = {
            'url': url,
            'name': name if name else f"订阅源 {len(self.subscriptions) + 1}",
            'added_at': datetime.now().isoformat(),
            'last_updated': None,
            'channel_count': 0,
            'status': 'active'
        }
        
        # 添加到列表并保存
        self.subscriptions.append(subscription)
        if self.save_subscriptions():
            return True, "订阅源添加成功"
        else:
            # 如果保存失败，回滚操作
            self.subscriptions.pop()
            return False, "保存订阅源失败"
    
    def remove_subscription(self, url):
        """删除订阅源
        
        Args:
            url: 要删除的订阅源URL
            
        Returns:
            tuple: (是否成功, 消息)
        """
        initial_count = len(self.subscriptions)
        self.subscriptions = [sub for sub in self.subscriptions if sub['url'] != url]
        
        if len(self.subscriptions) < initial_count:
            if self.save_subscriptions():
                return True, "订阅源删除成功"
            else:
                # 保存失败，恢复原始数据
                self.load_subscriptions()
                return False, "保存更改失败"
        else:
            return False, "未找到指定的订阅源"
    
    def update_subscription(self, old_url, new_url, new_name=None):
        """更新订阅源信息
        
        Args:
            old_url: 原订阅源URL
            new_url: 新订阅源URL
            new_name: 新订阅源名称（可选）
            
        Returns:
            tuple: (是否成功, 消息)
        """
        # 查找订阅源
        found = False
        for i, sub in enumerate(self.subscriptions):
            if sub['url'] == old_url:
                found = True
                break
        
        if not found:
            return False, "未找到指定的订阅源"
        
        # 如果URL发生变化，检查新URL是否已存在
        if old_url != new_url:
            for sub in self.subscriptions:
                if sub['url'] == new_url:
                    return False, "新URL已存在于订阅列表中"
            
            # 验证新URL
            is_valid, error_msg = self.validate_url(new_url)
            if not is_valid:
                return False, error_msg
        
        # 更新订阅源信息
        self.subscriptions[i]['url'] = new_url
        if new_name is not None:
            self.subscriptions[i]['name'] = new_name
        
        # 保存更改
        if self.save_subscriptions():
            return True, "订阅源更新成功"
        else:
            # 保存失败，恢复原始数据
            self.load_subscriptions()
            return False, "保存更改失败"
    
    def get_all_subscriptions(self):
        """获取所有订阅源
        
        Returns:
            list: 订阅源列表
        """
        return self.subscriptions
    
    def get_subscription(self, url):
        """获取指定URL的订阅源信息
        
        Args:
            url: 订阅源URL
            
        Returns:
            dict: 订阅源信息，如果未找到则返回None
        """
        for sub in self.subscriptions:
            if sub['url'] == url:
                return sub
        return None
    
    def update_subscription_status(self, url, status, channel_count=None):
        """更新订阅源状态
        
        Args:
            url: 订阅源URL
            status: 新状态（'active', 'invalid', 'failed'等）
            channel_count: 频道数量（可选）
            
        Returns:
            bool: 是否成功
        """
        for i, sub in enumerate(self.subscriptions):
            if sub['url'] == url:
                self.subscriptions[i]['status'] = status
                self.subscriptions[i]['last_updated'] = datetime.now().isoformat()
                if channel_count is not None:
                    self.subscriptions[i]['channel_count'] = channel_count
                return self.save_subscriptions()
        return False