package utils

import (
	"sync"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	limiters map[int64]*groupLimiter
	mu       sync.RWMutex
	maxRate  int // 每秒最大操作数
}

type groupLimiter struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(maxRate int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[int64]*groupLimiter),
		maxRate:  maxRate,
	}
}

// Allow 检查是否允许操作
func (r *RateLimiter) Allow(groupID int64) bool {
	r.mu.Lock()
	limiter, exists := r.limiters[groupID]
	if !exists {
		limiter = &groupLimiter{
			tokens:    r.maxRate,
			lastReset: time.Now(),
		}
		r.limiters[groupID] = limiter
	}
	r.mu.Unlock()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// 检查是否需要重置
	now := time.Now()
	if now.Sub(limiter.lastReset) >= time.Second {
		limiter.tokens = r.maxRate
		limiter.lastReset = now
	}

	// 检查是否有可用令牌
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// Wait 等待直到可以执行操作
func (r *RateLimiter) Wait(groupID int64) {
	for !r.Allow(groupID) {
		time.Sleep(100 * time.Millisecond)
	}
}

// Reset 重置指定群组的限制器
func (r *RateLimiter) Reset(groupID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.limiters, groupID)
}

// CleanupOldLimiters 清理旧的限制器（定期调用）
func (r *RateLimiter) CleanupOldLimiters() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for groupID, limiter := range r.limiters {
		limiter.mu.Lock()
		if now.Sub(limiter.lastReset) > 5*time.Minute {
			delete(r.limiters, groupID)
		}
		limiter.mu.Unlock()
	}
}

