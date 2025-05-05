# -*- coding: utf-8 -*-
"""
配置模块

该模块负责管理应用配置。
"""

import os
import json
import logging

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class Config:
    """应用配置类"""
    
    def __init__(self, config_file='data/config.json'):
        """初始化配置
        
        Args:
            config_file: 配置文件路径
        """
        # 基础配置
        self.data_dir = 'data'
        self.config_file = config_file
        
        # 请求配置
        self.request_timeout = 30  # 请求超时时间（秒）
        self.stream_test_timeout = 5  # 流测试超时时间（秒）
        self.max_test_workers = 10  # 最大并发测试数
        
        # 更新配置
        self.update_interval_hours = 24  # 更新间隔（小时）
        self.enable_stream_test = True  # 是否启用流测试
        self.test_interval_hours = 24  # 测试间隔（小时）
        self.test_all_sources = False  # 是否测试所有源URL
        
        # 聚合配置
        self.match_by = 'name'  # 匹配方式：'name', 'tvg_id', 'both'
        self.similarity_threshold = 0.85  # 名称相似度阈值
        
        # 确保数据目录存在
        if not os.path.exists(self.data_dir):
            os.makedirs(self.data_dir)
        
        # 加载配置
        self.load_config()
    
    def load_config(self):
        """从文件加载配置"""
        if os.path.exists(self.config_file):
            try:
                with open(self.config_file, 'r', encoding='utf-8') as f:
                    config_data = json.load(f)
                
                # 更新配置属性
                for key, value in config_data.items():
                    if hasattr(self, key):
                        setattr(self, key, value)
                
                logger.info("配置已加载")
            except Exception as e:
                logger.error(f"加载配置时出错: {str(e)}")
        else:
            logger.info("配置文件不存在，使用默认配置")
            self.save_config()  # 保存默认配置
    
    def save_config(self):
        """保存配置到文件"""
        try:
            # 创建配置字典
            config_data = {
                'request_timeout': self.request_timeout,
                'stream_test_timeout': self.stream_test_timeout,
                'max_test_workers': self.max_test_workers,
                'update_interval_hours': self.update_interval_hours,
                'enable_stream_test': self.enable_stream_test,
                'test_interval_hours': self.test_interval_hours,
                'test_all_sources': self.test_all_sources,
                'match_by': self.match_by,
                'similarity_threshold': self.similarity_threshold
            }
            
            with open(self.config_file, 'w', encoding='utf-8') as f:
                json.dump(config_data, f, ensure_ascii=False, indent=2)
            
            logger.info("配置已保存")
            return True
        except Exception as e:
            logger.error(f"保存配置时出错: {str(e)}")
            return False
    
    def get_config_dict(self):
        """获取配置字典
        
        Returns:
            dict: 配置字典
        """
        return {
            'request_timeout': self.request_timeout,
            'stream_test_timeout': self.stream_test_timeout,
            'max_test_workers': self.max_test_workers,
            'update_interval_hours': self.update_interval_hours,
            'enable_stream_test': self.enable_stream_test,
            'test_interval_hours': self.test_interval_hours,
            'test_all_sources': self.test_all_sources,
            'match_by': self.match_by,
            'similarity_threshold': self.similarity_threshold
        }