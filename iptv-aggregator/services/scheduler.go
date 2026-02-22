package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/iptv-aggregator/utils"
)

// Job 任务定义
type Job struct {
	ID       string
	Schedule string
	Fn       func() error
	ticker   *time.Ticker
	done     chan bool
}

// Scheduler 任务调度器
type Scheduler struct {
	jobs map[string]*Job
	mu   sync.RWMutex
	stop chan bool
}

// NewScheduler 创建新的调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		stop: make(chan bool),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		go s.runJob(job)
	}

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.ticker != nil {
			job.ticker.Stop()
		}
		select {
		case job.done <- true:
		default:
		}
	}

	return nil
}

// AddJob 添加任务
func (s *Scheduler) AddJob(id string, schedule string, fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; exists {
		return fmt.Errorf("job already exists: %s", id)
	}

	duration, err := s.parseDuration(schedule)
	if err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	job := &Job{
		ID:       id,
		Schedule: schedule,
		Fn:       fn,
		ticker:   time.NewTicker(duration),
		done:     make(chan bool),
	}

	s.jobs[id] = job

	// 如果调度器已启动，立即启动新任务
	go s.runJob(job)

	return nil
}

// RemoveJob 删除任务
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.ticker != nil {
		job.ticker.Stop()
	}
	select {
	case job.done <- true:
	default:
	}

	delete(s.jobs, id)

	return nil
}

// TriggerJob 触发任务
func (s *Scheduler) TriggerJob(id string) error {
	logger := utils.NewLogger()

	s.mu.RLock()
	job, exists := s.jobs[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	// 异步执行任务
	go func() {
		if err := job.Fn(); err != nil {
			logger.Error("Job %s failed: %v", job.ID, err)
		}
	}()

	return nil
}

// runJob 运行任务
func (s *Scheduler) runJob(job *Job) {
	logger := utils.NewLogger()

	// 立即执行一次
	if err := job.Fn(); err != nil {
		logger.Error("Job %s failed: %v", job.ID, err)
	}

	// 定期执行
	for {
		select {
		case <-job.ticker.C:
			if err := job.Fn(); err != nil {
				logger.Error("Job %s failed: %v", job.ID, err)
			}
		case <-job.done:
			return
		}
	}
}

// parseDuration 解析时间间隔
func (s *Scheduler) parseDuration(schedule string) (time.Duration, error) {
	// 支持简单的时间格式，如 "1h", "30m", "60s"
	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %s", schedule)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return duration, nil
}
