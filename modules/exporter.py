# -*- coding: utf-8 -*-
"""
导出功能模块

该模块负责将聚合后的频道列表导出为M3U或JSON格式。
"""

import os
import json
import logging
from datetime import datetime

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class ChannelExporter:
    """频道导出器"""
    
    def __init__(self, data_dir='data'):
        """初始化导出器
        
        Args:
            data_dir: 数据存储目录
        """
        self.data_dir = data_dir
        self.export_dir = os.path.join(data_dir, 'exports')
        
        # 确保导出目录存在
        if not os.path.exists(self.export_dir):
            os.makedirs(self.export_dir)
    
    def export_m3u(self, channels, filename=None, only_working=False):
        """导出为M3U格式
        
        Args:
            channels: 频道列表
            filename: 文件名（可选）
            only_working: 是否只导出工作的频道
            
        Returns:
            tuple: (是否成功, 文件路径或错误消息)
        """
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"iptv_export_{timestamp}.m3u"
        
        file_path = os.path.join(self.export_dir, filename)
        
        try:
            with open(file_path, 'w', encoding='utf-8') as f:
                # 写入M3U头
                f.write("#EXTM3U\n")
                
                # 写入频道信息
                exported_count = 0
                for channel in channels:
                    # 如果只导出工作的频道，检查状态
                    if only_working and ('test_results' in channel and 
                                         channel['test_results'].get('status') != 'online'):
                        continue
                    
                    # 获取URL（优先使用测试通过的URL）
                    url = channel.get('url', '')
                    if 'test_results' in channel and channel['test_results'].get('working_url'):
                        url = channel['test_results']['working_url']
                    
                    if not url:  # 跳过没有URL的频道
                        continue
                    
                    # 构建EXTINF行
                    extinf = f"#EXTINF:-1 "
                    if channel.get('tvg_id'):
                        extinf += f"tvg-id=\"{channel['tvg_id']}\" "
                    if channel.get('tvg_name'):
                        extinf += f"tvg-name=\"{channel['tvg_name']}\" "
                    elif channel.get('name'):
                        extinf += f"tvg-name=\"{channel['name']}\" "
                    if channel.get('tvg_logo'):
                        extinf += f"tvg-logo=\"{channel['tvg_logo']}\" "
                    if channel.get('group_title'):
                        extinf += f"group-title=\"{channel['group_title']}\" "
                    
                    # 添加频道名称
                    extinf += f",{channel['name']}\n"
                    
                    # 写入EXTINF行和URL
                    f.write(extinf)
                    f.write(f"{url}\n")
                    
                    exported_count += 1
            
            logger.info(f"已导出 {exported_count} 个频道到 {file_path}")
            return True, file_path
        except Exception as e:
            error_msg = f"导出M3U文件时出错: {str(e)}"
            logger.error(error_msg)
            return False, error_msg
    
    def export_json(self, channels, filename=None, only_working=False):
        """导出为JSON格式
        
        Args:
            channels: 频道列表
            filename: 文件名（可选）
            only_working: 是否只导出工作的频道
            
        Returns:
            tuple: (是否成功, 文件路径或错误消息)
        """
        if not filename:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"iptv_export_{timestamp}.json"
        
        file_path = os.path.join(self.export_dir, filename)
        
        try:
            # 过滤频道
            export_channels = channels
            if only_working:
                export_channels = [ch for ch in channels 
                                  if 'test_results' in ch and ch['test_results'].get('status') == 'online']
            
            # 导出数据
            export_data = {
                'exported_at': datetime.now().isoformat(),
                'total_channels': len(export_channels),
                'channels': export_channels
            }
            
            with open(file_path, 'w', encoding='utf-8') as f:
                json.dump(export_data, f, ensure_ascii=False, indent=2)
            
            logger.info(f"已导出 {len(export_channels)} 个频道到 {file_path}")
            return True, file_path
        except Exception as e:
            error_msg = f"导出JSON文件时出错: {str(e)}"
            logger.error(error_msg)
            return False, error_msg
    
    def get_export_list(self):
        """获取导出文件列表
        
        Returns:
            list: 导出文件信息列表
        """
        exports = []
        
        if os.path.exists(self.export_dir):
            for filename in os.listdir(self.export_dir):
                file_path = os.path.join(self.export_dir, filename)
                if os.path.isfile(file_path):
                    # 获取文件信息
                    file_info = {
                        'filename': filename,
                        'path': file_path,
                        'size': os.path.getsize(file_path),
                        'created_at': datetime.fromtimestamp(os.path.getctime(file_path)).isoformat()
                    }
                    exports.append(file_info)
        
        # 按创建时间排序
        exports.sort(key=lambda x: x['created_at'], reverse=True)
        
        return exports
    
    def delete_export(self, filename):
        """删除导出文件
        
        Args:
            filename: 文件名
            
        Returns:
            bool: 是否成功
        """
        file_path = os.path.join(self.export_dir, filename)
        
        if os.path.exists(file_path) and os.path.isfile(file_path):
            try:
                os.remove(file_path)
                logger.info(f"已删除导出文件: {filename}")
                return True
            except Exception as e:
                logger.error(f"删除导出文件时出错: {str(e)}")
                return False
        else:
            logger.warning(f"导出文件不存在: {filename}")
            return False