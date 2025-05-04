import logging
import requests
import m3u8
import socket
from urllib.parse import urlparse, urljoin, quote, unquote
from typing import List, Dict, Tuple
import re
import asyncio
import aiohttp
import time
from concurrent.futures import ThreadPoolExecutor
from aiohttp import ClientTimeout
from aiohttp.client_exceptions import (
    ClientConnectorError, 
    ServerDisconnectedError,
    ClientOSError
)

logger = logging.getLogger(__name__)

class M3UParser:
    @staticmethod
    def normalize_url(url: str, base_url: str = None) -> str:
        """规范化URL地址，支持IPv6"""
        url = url.strip()
        
        # 解码URL，以处理可能的重复编码
        url = unquote(url)
        
        # 处理IPv6地址
        if '[' in url and ']' in url:
            # 保持IPv6地址的方括号
            ipv6_pattern = r'\[([0-9a-fA-F:]+)\]'
            url = re.sub(ipv6_pattern, lambda m: '[' + m.group(1) + ']', url)
        
        # 处理相对路径
        if base_url and not url.startswith(('http://', 'https://')):
            url = urljoin(base_url, url)
        
        # 解析URL
        parsed = urlparse(url)
        
        # 分别处理路径和查询参数
        path = quote(parsed.path) if not '[' in parsed.path else parsed.path
        query = parsed.query
        if query:
            # 保持某些特殊字符不被编码
            query = quote(query, safe='=&:')
        
        # 重建URL
        netloc = parsed.netloc if '[' in parsed.netloc else quote(parsed.netloc)
        if query:
            return f"{parsed.scheme}://{netloc}{path}?{query}"
        return f"{parsed.scheme}://{netloc}{path}"

    @staticmethod
    def setup_requests_session() -> requests.Session:
        """设置请求会话，支持IPv6"""
        session = requests.Session()
        # 配置连接池
        adapter = requests.adapters.HTTPAdapter(
            pool_connections=20,
            pool_maxsize=20,
            max_retries=2,
            pool_block=False
        )
        session.mount('http://', adapter)
        session.mount('https://', adapter)
        return session

    @staticmethod
    def is_ipv6_url(url: str) -> bool:
        """检查是否是IPv6地址的URL"""
        return '[' in url and ']' in url

    @staticmethod
    def is_m3u_content(content: str) -> bool:
        """检查内容是否是M3U格式"""
        # 移除BOM标记
        if content.startswith('\ufeff'):
            content = content[1:]
            
        # 检查常见的M3U标记
        content = content.lstrip()
        return (content.startswith('#EXTM3U') or
                '#EXTINF:' in content[:1000] or
                content.lower().endswith('.m3u8'))

    @staticmethod
    def extract_m3u_urls(content: str) -> List[str]:
        """从文本内容中提取所有可能的M3U8 URL"""
        urls = set()
        for line in content.splitlines():
            line = line.strip()
            if line and not line.startswith('#'):
                if line.lower().endswith('.m3u8'):
                    urls.add(line)
                elif 'application/x-mpegurl' in line.lower():
                    try:
                        url_match = re.search(r'https?://[^\s<>"\']+?\.m3u8', line)
                        if url_match:
                            urls.add(url_match.group(0))
                    except:
                        pass
        return list(urls)

    @staticmethod
    def extract_attribute(line: str, attr: str) -> str:
        """从EXTINF行提取属性值"""
        try:
            pattern = f'{attr}="([^"]*)"'
            match = re.search(pattern, line)
            return match.group(1) if match else None
        except:
            return None

class URLChecker:
    _session = requests.Session()
    _timeout = 5  # 默认5秒超时
    _headers = {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36',
        'Accept': '*/*'
    }

    @classmethod
    async def async_check_url(cls, url: str, session: aiohttp.ClientSession) -> Tuple[str, bool, str]:
        """异步检查URL可用性"""
        start_time = time.time()
        timeout = ClientTimeout(total=cls._timeout)
        
        try:
            async with session.get(url, timeout=timeout, headers=cls._headers, ssl=False) as response:
                if response.status == 200:
                    try:
                        # 只读取头部内容进行验证
                        content = await response.content.read(1024)
                        is_valid = content.startswith(b'#EXTM3U') or b'#EXTINF' in content
                        elapsed = time.time() - start_time
                        return url, is_valid, f"{elapsed:.2f}秒"
                    except asyncio.TimeoutError:
                        return url, False, "读取超时"
                    except Exception as e:
                        return url, False, f"读取错误: {str(e)}"
                return url, False, f"HTTP {response.status}"
                
        except asyncio.TimeoutError:
            return url, False, "连接超时"
        except ClientConnectorError:
            return url, False, "无法连接"
        except ServerDisconnectedError:
            return url, False, "服务器断开连接"
        except ClientOSError as e:
            return url, False, f"网络错误: {str(e)}"
        except Exception as e:
            return url, False, str(e)

    @classmethod
    async def check_urls_concurrent(cls, urls: List[str]) -> List[Dict]:
        """并发检查多个URL"""
        # 创建TCP连接器并配置参数
        connector = aiohttp.TCPConnector(
            limit=20,  # 最大并发连接数
            force_close=True,  # 每次请求后强制关闭连接
            enable_cleanup_closed=True,  # 自动清理关闭的连接
            ssl=False  # 禁用SSL验证以提高性能
        )
        
        # 创建客户端会话
        timeout = ClientTimeout(total=cls._timeout)
        async with aiohttp.ClientSession(
            connector=connector,
            timeout=timeout,
            headers=cls._headers
        ) as session:
            tasks = []
            for url in urls:
                task = asyncio.ensure_future(cls.async_check_url(url, session))
                tasks.append(task)
            
            results = []
            try:
                for future in asyncio.as_completed(tasks, timeout=cls._timeout * 2):
                    try:
                        url, is_valid, message = await future
                        results.append({
                            'url': url,
                            'available': is_valid,
                            'message': message
                        })
                    except asyncio.TimeoutError:
                        logger.error(f"任务执行超时")
                        continue
                    except Exception as e:
                        logger.error(f"任务执行错误: {str(e)}")
                        continue
            except asyncio.TimeoutError:
                logger.error("整体并发检查超时")
            
            # 取消所有未完成的任务
            for task in tasks:
                if not task.done():
                    task.cancel()
            
            return results

    @classmethod
    def check_availability(cls, url: str, timeout: int = 5) -> bool:
        """同步检查单个URL可用性"""
        try:
            # 对于单个URL的检查，使用同步方式
            response = cls._session.get(
                url,
                timeout=timeout,
                verify=False,
                stream=True,
                headers={'User-Agent': 'Mozilla/5.0'}
            )
            response.raise_for_status()
            
            # 读取部分内容验证
            content_start = response.raw.read(1024).decode('utf-8', errors='ignore')
            response.close()
            
            return M3UParser.is_m3u_content(content_start)
            
        except Exception as e:
            logger.debug(f"URL检查失败 {url}: {str(e)}")
            return False

    @classmethod
    def run_concurrent_check(cls, urls: List[str]) -> List[Dict]:
        """运行并发检查（同步包装异步）"""
        async def run():
            return await cls.check_urls_concurrent(urls)
            
        try:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            results = loop.run_until_complete(cls.check_urls_concurrent(urls))
            loop.close()
            return results
        except Exception as e:
            logger.error(f"并发检查失败: {str(e)}")
            return []
