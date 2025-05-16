# -*- coding: utf-8 -*-
"""
流媒体测试模块

该模块负责测试频道URL的可用性。
"""

import requests
import logging
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class StreamTester:
    """流媒体测试器"""
    
    def __init__(self, timeout=5, max_workers=10):
        """初始化测试器
        
        Args:
            timeout: 请求超时时间（秒）
            max_workers: 最大并发测试数
        """
        self.timeout = timeout
        self.max_workers = max_workers
    def test_stream(self, url):
        """测试单个流URL的可用性（支持m3u8分片检测和rtmp协议）
        Args:
            url: 流媒体URL
        Returns:
            tuple: (是否可用, 响应时间或错误消息)
        """
        import re
        from urllib.parse import urljoin
        start_time = time.time()
        try:
            headers = {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
            }
            # RTMP 协议检测（仅做格式校验和端口可达性简单检测）
            if url.lower().startswith('rtmp'):
                # 只检测格式和端口可达性，不做完整握手
                import socket
                from urllib.parse import urlparse
                parsed = urlparse(url)
                host = parsed.hostname
                port = parsed.port or 1935
                try:
                    sock = socket.create_connection((host, port), timeout=self.timeout)
                    sock.close()
                    elapsed = time.time() - start_time
                    return True, elapsed
                except Exception as e:
                    return False, f"RTMP端口不可达: {e}"
            # 对于特殊格式的URL（如斗鱼等），先尝试GET请求
            if 'douyu' in url or 'huya' in url or 'bilibili' in url:
                response = requests.get(url, timeout=self.timeout, headers=headers, stream=True)
                for chunk in response.iter_content(chunk_size=1024):
                    if chunk:
                        break
                elapsed = time.time() - start_time
                if response.status_code == 200:
                    return True, elapsed
                else:
                    return False, f"HTTP错误: {response.status_code}"
            # 对于M3U8文件，下载并测试分片
            elif url.endswith('.m3u8'):
                m3u8_resp = requests.get(url, timeout=self.timeout, headers=headers)
                if m3u8_resp.status_code != 200:
                    return False, f"M3U8下载失败: {m3u8_resp.status_code}"
                m3u8_text = m3u8_resp.text
                # 提取前3个ts分片
                ts_urls = []
                for line in m3u8_text.splitlines():
                    line = line.strip()
                    if not line or line.startswith('#'):
                        continue
                    # 只取.ts结尾或无扩展名的分片
                    if line.endswith('.ts') or re.match(r"^[^#?]+\.ts([?#].*)?$", line) or ('.ts' in line):
                        ts_urls.append(urljoin(url, line))
                    # 兼容部分无扩展名的分片
                    elif re.match(r"^[^#]+$", line) and (not line.startswith('http')):
                        ts_urls.append(urljoin(url, line))
                    if len(ts_urls) >= 3:
                        break
                if not ts_urls:
                    return False, "未找到TS分片"
                # 依次HEAD分片，只要有一个可用即判为可用
                for ts_url in ts_urls:
                    try:
                        ts_resp = requests.head(ts_url, timeout=self.timeout, headers=headers, allow_redirects=True)
                        if ts_resp.status_code == 200:
                            elapsed = time.time() - start_time
                            return True, elapsed
                    except Exception as e:
                        continue
                return False, "TS分片不可访问"
            # 对于其他类型的流，使用HEAD请求
            else:
                response = requests.head(url, timeout=self.timeout, headers=headers)
                elapsed = time.time() - start_time
                if response.status_code == 200:
                    return True, elapsed
                else:
                    return False, f"HTTP错误: {response.status_code}"
        except requests.exceptions.Timeout:
            return False, "请求超时"
        except requests.exceptions.ConnectionError:
            return False, "连接错误"
        except Exception as e:
            return False, str(e)
    
    def batch_test(self, channels, test_all_sources=False):
        """批量测试频道
        
        Args:
            channels: 频道列表
            test_all_sources: 是否测试所有源URL
            
        Returns:
            list: 更新后的频道列表
        """
        logger.info(f"开始批量测试 {len(channels)} 个频道")

        # 创建测试任务列表
        test_tasks = []

        for i, channel in enumerate(channels):
            # 初始化测试状态
            if 'test_results' not in channel or not isinstance(channel['test_results'], dict):
                channel['test_results'] = {
                    'status': 'untested',
                    'last_tested': None,
                    'working_url': None,
                    'response_time': None
                }
            # 确保主URL存在
            if 'url' in channel and channel['url']:
                test_tasks.append((i, channel['url'], 'main'))
            # 测试所有源URL
            if test_all_sources and 'sources' in channel and isinstance(channel['sources'], list):
                for j, source in enumerate(channel['sources']):
                    if isinstance(source, dict) and 'url' in source and source['url']:
                        test_tasks.append((i, source['url'], f"source_{j}"))

        # 使用sys.modules动态查找test_progress变量，避免循环导入问题
        import sys
        test_progress = None
        if 'app' in sys.modules and hasattr(sys.modules['app'], 'test_progress'):
            test_progress = sys.modules['app'].test_progress
        else:
            for module_name, module in sys.modules.items():
                if hasattr(module, 'test_progress'):
                    test_progress = module.test_progress
                    logger.info(f"从模块 {module_name} 获取到test_progress变量")
                    break
        if test_progress is None:
            logger.warning("无法获取测试进度变量，将跳过进度更新")

        completed_count = 0
        online_count = 0
        offline_count = 0

        # 支持 Ctrl+C 中断
        try:
            with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
                future_to_task = {executor.submit(self.test_stream, url): (i, url, url_type)
                                 for i, url, url_type in test_tasks}
                for future in as_completed(future_to_task):
                    i, url, url_type = future_to_task[future]
                    try:
                        is_working, result = future.result()
                        self._update_test_result(channels[i], url, url_type, is_working, result)
                    except Exception as e:
                        logger.exception(f"Error testing {url}: {e}")

                    completed_count += 1
                    # 更新在线/离线计数
                    if channels[i]['test_results']['status'] == 'online':
                        online_count += 1
                    elif channels[i]['test_results']['status'] == 'offline':
                        offline_count += 1

                    # 更新全局测试进度（如果存在）
                    if test_progress is not None:
                        test_progress['completed'] = completed_count
                        test_progress['online'] = online_count
                        test_progress['offline'] = offline_count
                        test_progress['total'] = len(channels)

                    if completed_count % 10 == 0:
                        logger.info(f"测试进度: {completed_count}/{len(channels)} (在线: {online_count}, 离线: {offline_count})")
                logger.info(f"所有频道测试完成! 总计: {len(channels)}, 在线: {online_count}, 离线: {offline_count}")
        except KeyboardInterrupt:
            logger.warning("检测任务被用户中断 (Ctrl+C)")
            # 标记所有未完成的频道为未测试
            for i, channel in enumerate(channels):
                if channel['test_results'].get('status', 'untested') == 'untested':
                    channel['test_results']['status'] = 'untested'
            if test_progress is not None:
                test_progress['is_testing'] = False
            raise
        except Exception as e:
            logger.exception(f"测试任务异常: {str(e)}")

        # 确保所有频道都至少有 test_results 字段
        for channel in channels:
            if 'test_results' not in channel or not isinstance(channel['test_results'], dict):
                channel['test_results'] = {
                    'status': 'untested',
                    'last_tested': None,
                    'working_url': None,
                    'response_time': None
                }
        logger.info("批量测试完成")
        return channels
    
    def _update_test_result(self, channel, url, url_type, is_working, result):
        """更新频道的测试结果
        
        Args:
            channel: 频道数据
            url: 测试的URL
            url_type: URL类型（'main'或'source_X'）
            is_working: 是否可用
            result: 测试结果（响应时间或错误消息）
        """
        # 更新测试时间
        channel['test_results']['last_tested'] = time.strftime("%Y-%m-%d %H:%M:%S")

        # 如果是可用的URL
        if is_working:
            # 更新状态为直播中
            channel['test_results']['status'] = 'online'
            channel['test_results']['error'] = None
            # 记录工作的URL和响应时间
            channel['test_results']['working_url'] = url
            channel['test_results']['response_time'] = round(result, 3) if isinstance(result, (int, float)) else None
            # 如果是源URL且主URL不可用，将其设为主URL
            if url_type.startswith('source_') and channel.get('url') != url:
                channel['url'] = url
        # 如果是主URL且不可用
        elif url_type == 'main':
            # 暂时将状态设为离线，后续源URL测试可能会更新
            channel['test_results']['status'] = 'offline'
            channel['test_results']['error'] = result
            channel['test_results']['working_url'] = None
            channel['test_results']['response_time'] = None
        # 如果是源URL且不可用，且主URL也不可用，保持状态为离线
        elif url_type.startswith('source_'):
            # 只有当主URL也不可用时才设置为离线
            if channel['test_results'].get('status') != 'online':
                channel['test_results']['status'] = 'offline'
                channel['test_results']['error'] = result
                channel['test_results']['working_url'] = None
                channel['test_results']['response_time'] = None