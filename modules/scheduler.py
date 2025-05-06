# -*- coding: utf-8 -*-
"""
调度器模块

该模块负责管理周期性任务，如自动更新订阅源和测试流媒体。
"""

import logging
from datetime import datetime
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from apscheduler.triggers.interval import IntervalTrigger

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class UpdateScheduler:
    """更新调度器"""
    
    def __init__(self):
        """初始化调度器"""
        self.scheduler = BackgroundScheduler()
        self.jobs = {}
    
    def start(self):
        """启动调度器"""
        if not self.scheduler.running:
            self.scheduler.start()
            logger.info("调度器已启动")
    
    def shutdown(self):
        """关闭调度器"""
        if self.scheduler.running:
            self.scheduler.shutdown()
            logger.info("调度器已关闭")
    
    def add_interval_job(self, job_id, func, hours=24, minutes=0, seconds=0, args=None, kwargs=None, run_immediately=True):
        """添加间隔任务
        
        Args:
            job_id: 任务ID
            func: 要执行的函数
            hours: 间隔小时数
            minutes: 间隔分钟数
            seconds: 间隔秒数
            args: 函数参数
            kwargs: 函数关键字参数
            run_immediately: 是否立即执行一次，默认为True
            
        Returns:
            bool: 是否成功
        """
        # 移除同ID的现有任务
        self.remove_job(job_id)
        
        # 添加新任务
        try:
            # 设置下次运行时间
            next_run_time = datetime.now() if run_immediately else None
            
            job = self.scheduler.add_job(
                func=func,
                trigger=IntervalTrigger(hours=hours, minutes=minutes, seconds=seconds),
                id=job_id,
                args=args or (),
                kwargs=kwargs or {},
                next_run_time=next_run_time  # 根据参数决定是否立即执行
            )
            self.jobs[job_id] = job
            
            if run_immediately:
                logger.info(f"已添加间隔任务: {job_id}, 间隔: {hours}小时 {minutes}分钟 {seconds}秒, 立即执行一次")
            else:
                logger.info(f"已添加间隔任务: {job_id}, 间隔: {hours}小时 {minutes}分钟 {seconds}秒, 等待下次执行时间")
                
            return True
        except Exception as e:
            logger.error(f"添加间隔任务失败: {str(e)}")
            return False
    
    def add_cron_job(self, job_id, func, cron_expression, args=None, kwargs=None):
        """添加Cron任务
        
        Args:
            job_id: 任务ID
            func: 要执行的函数
            cron_expression: Cron表达式
            args: 函数参数
            kwargs: 函数关键字参数
            
        Returns:
            bool: 是否成功
        """
        # 移除同ID的现有任务
        self.remove_job(job_id)
        
        # 添加新任务
        try:
            job = self.scheduler.add_job(
                func=func,
                trigger=CronTrigger.from_crontab(cron_expression),
                id=job_id,
                args=args or (),
                kwargs=kwargs or {}
            )
            self.jobs[job_id] = job
            logger.info(f"已添加Cron任务: {job_id}, 表达式: {cron_expression}")
            return True
        except Exception as e:
            logger.error(f"添加Cron任务失败: {str(e)}")
            return False
    
    def remove_job(self, job_id):
        """移除任务
        
        Args:
            job_id: 任务ID
            
        Returns:
            bool: 是否成功
        """
        if job_id in self.jobs:
            try:
                self.scheduler.remove_job(job_id)
                del self.jobs[job_id]
                logger.info(f"已移除任务: {job_id}")
                return True
            except Exception as e:
                logger.error(f"移除任务失败: {str(e)}")
                return False
        return False
    
    def get_jobs(self):
        """获取所有任务
        
        Returns:
            list: 任务信息列表
        """
        job_list = []
        for job in self.scheduler.get_jobs():
            next_run = job.next_run_time.strftime("%Y-%m-%d %H:%M:%S") if job.next_run_time else "未计划"
            job_info = {
                'id': job.id,
                'next_run_time': next_run,
                'trigger': str(job.trigger)
            }
            job_list.append(job_info)
        return job_list
    
    def pause_job(self, job_id):
        """暂停任务
        
        Args:
            job_id: 任务ID
            
        Returns:
            bool: 是否成功
        """
        if job_id in self.jobs:
            try:
                self.scheduler.pause_job(job_id)
                logger.info(f"已暂停任务: {job_id}")
                return True
            except Exception as e:
                logger.error(f"暂停任务失败: {str(e)}")
                return False
        return False
    
    def resume_job(self, job_id):
        """恢复任务
        
        Args:
            job_id: 任务ID
            
        Returns:
            bool: 是否成功
        """
        if job_id in self.jobs:
            try:
                self.scheduler.resume_job(job_id)
                logger.info(f"已恢复任务: {job_id}")
                return True
            except Exception as e:
                logger.error(f"恢复任务失败: {str(e)}")
                return False
        return False