import os
import logging
from app.api.routes import app
from app.utils.config import logger

def setup_directories():
    """创建必要的目录"""
    directories = ['templates', 'static']
    for directory in directories:
        os.makedirs(directory, exist_ok=True)
        logger.info(f"确保目录存在: {directory}")

def main():
    """主程序入口"""
    # 设置日志级别
    logging.getLogger('werkzeug').setLevel(logging.WARNING)
    
    # 创建必要的目录
    setup_directories()
    
    # 启动Web服务器
    print("正在启动IPTV管理系统...")
    print("请在浏览器中访问 http://localhost:5000")
    
    # 运行Flask应用
    app.run(
        host='0.0.0.0',
        port=5000,
        debug=True
    )

if __name__ == '__main__':
    main()
