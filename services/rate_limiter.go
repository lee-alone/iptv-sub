package services

import (
	"net/url"
	"sync"
	"time"

	"iptv-aggregator/utils"
)

// DomainLimiter 基于域名的分桶限流器
type DomainLimiter struct {
	semaphores map[string]chan struct{}
	mu         sync.Mutex
	maxPerHost int // 每个主机的最大并发数
	logger     *utils.Logger
}

// NewDomainLimiter 创建域名限流器
func NewDomainLimiter(maxPerHost int) *DomainLimiter {
	return &DomainLimiter{
		semaphores: make(map[string]chan struct{}),
		maxPerHost: maxPerHost,
		logger:     utils.NewLogger(),
	}
}

// Acquire 获取对指定 URL 的访问权限，返回释放函数
func (dl *DomainLimiter) Acquire(urlStr string) func() {
	u, err := url.Parse(urlStr)
	if err != nil {
		// 如果 URL 解析失败，使用整个 URL 作为 key
		return dl.acquireForHost(urlStr)
	}

	host := u.Hostname()
	if host == "" {
		host = urlStr
	}

	return dl.acquireForHost(host)
}

// acquireForHost 获取对指定主机的访问权限
func (dl *DomainLimiter) acquireForHost(host string) func() {
	dl.mu.Lock()
	if dl.semaphores[host] == nil {
		dl.semaphores[host] = make(chan struct{}, dl.maxPerHost)
		dl.logger.Debug("Created semaphore for host: %s (max: %d)", host, dl.maxPerHost)
	}
	sem := dl.semaphores[host]
	dl.mu.Unlock()

	// 获取信号量（阻塞直到有可用位置）
	sem <- struct{}{}

	// 返回释放函数
	return func() {
		<-sem
	}
}

// GetStats 获取限流器统计信息
func (dl *DomainLimiter) GetStats() map[string]int {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	stats := make(map[string]int)
	for host, sem := range dl.semaphores {
		// 计算当前使用的信号量数
		used := dl.maxPerHost - len(sem)
		stats[host] = used
	}
	return stats
}

// AdaptiveLimiter 自适应限流器 - 根据错误率动态调整并发
type AdaptiveLimiter struct {
	maxWorkers      int
	currentWorkers  int
	recentErrors    []time.Time // 记录错误时间戳，而不是计数
	lastAdjustTime  time.Time
	adjustInterval  time.Duration
	errorThreshold  int // 触发降级的错误阈值
	recoveryTimeout time.Duration
	mu              sync.Mutex
	logger          *utils.Logger
}

// NewAdaptiveLimiter 创建自适应限流器
func NewAdaptiveLimiter(maxWorkers int) *AdaptiveLimiter {
	return &AdaptiveLimiter{
		maxWorkers:      maxWorkers,
		currentWorkers:  maxWorkers,
		recentErrors:    make([]time.Time, 0),
		lastAdjustTime:  time.Now(),
		adjustInterval:  30 * time.Second, // 每 30 秒检查一次
		errorThreshold:  5,                // 5 个错误触发降级
		recoveryTimeout: 2 * time.Minute,  // 2 分钟后尝试恢复
		logger:          utils.NewLogger(),
	}
}

// GetEffectiveWorkers 获取当前有效的并发数
func (al *AdaptiveLimiter) GetEffectiveWorkers() int {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 检查是否需要调整
	if time.Since(al.lastAdjustTime) > al.adjustInterval {
		al.adjustWorkers()
	}

	return al.currentWorkers
}

// RecordError 记录一个错误，使用时间窗口避免频繁降级
func (al *AdaptiveLimiter) RecordError() {
	al.mu.Lock()
	defer al.mu.Unlock()

	now := time.Now()
	al.recentErrors = append(al.recentErrors, now)

	// 只保留最近 30 秒的错误
	cutoff := now.Add(-30 * time.Second)
	valid := al.recentErrors[:0]
	for _, t := range al.recentErrors {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	al.recentErrors = valid

	al.logger.Warn("Error recorded. Recent errors in 30s: %d", len(al.recentErrors))

	// 如果错误过多，立即降级
	if len(al.recentErrors) >= al.errorThreshold {
		al.downgrade()
	}
}

// RecordSuccess 记录一个成功
func (al *AdaptiveLimiter) RecordSuccess() {
	al.mu.Lock()
	defer al.mu.Unlock()

	// 成功时逐步恢复错误计数
	if len(al.recentErrors) > 0 {
		al.recentErrors = al.recentErrors[1:]
	}
}

// adjustWorkers 调整并发数
func (al *AdaptiveLimiter) adjustWorkers() {
	// 如果错误少，尝试恢复
	if len(al.recentErrors) < 2 && al.currentWorkers < al.maxWorkers {
		al.currentWorkers = min(al.currentWorkers+1, al.maxWorkers)
		al.logger.Info("Recovering workers: %d -> %d", al.currentWorkers-1, al.currentWorkers)
	}

	// 重置错误计数
	al.recentErrors = al.recentErrors[:0]
	al.lastAdjustTime = time.Now()
}

// downgrade 降级并发数
func (al *AdaptiveLimiter) downgrade() {
	newWorkers := max(1, al.currentWorkers/2)
	if newWorkers != al.currentWorkers {
		al.logger.Warn("Downgrading workers due to high error rate: %d -> %d", al.currentWorkers, newWorkers)
		al.currentWorkers = newWorkers
		al.lastAdjustTime = time.Now()
	}
}

// HostCooldown 主机冷却管理器 - 避免对同一主机的过度请求
type HostCooldown struct {
	lastRequestTime map[string]time.Time
	cooldownTime    time.Duration
	mu              sync.Mutex
	logger          *utils.Logger
}

// NewHostCooldown 创建主机冷却管理器
func NewHostCooldown(cooldownTime time.Duration) *HostCooldown {
	return &HostCooldown{
		lastRequestTime: make(map[string]time.Time),
		cooldownTime:    cooldownTime,
		logger:          utils.NewLogger(),
	}
}

// Wait 等待直到可以请求指定主机
func (hc *HostCooldown) Wait(urlStr string) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return
	}

	host := u.Hostname()
	if host == "" {
		return
	}

	hc.mu.Lock()
	lastTime, exists := hc.lastRequestTime[host]
	hc.mu.Unlock()

	if exists {
		elapsed := time.Since(lastTime)
		if elapsed < hc.cooldownTime {
			waitTime := hc.cooldownTime - elapsed
			hc.logger.Debug("Cooling down host %s for %v", host, waitTime)
			time.Sleep(waitTime)
		}
	}
}

// Record 记录对指定主机的请求时间
func (hc *HostCooldown) Record(urlStr string) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return
	}

	host := u.Hostname()
	if host == "" {
		return
	}

	hc.mu.Lock()
	hc.lastRequestTime[host] = time.Now()
	hc.mu.Unlock()
}

// GetStats 获取冷却统计信息
func (hc *HostCooldown) GetStats() map[string]time.Time {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	stats := make(map[string]time.Time)
	for host, lastTime := range hc.lastRequestTime {
		stats[host] = lastTime
	}
	return stats
}
