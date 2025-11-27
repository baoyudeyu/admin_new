package utils

import (
	"sync"
	"time"
)

// SafeMap 并发安全的 map，带自动清理功能
type SafeMap struct {
	data      map[int64]time.Time
	mutex     sync.RWMutex
	maxAge    time.Duration // 条目最大存活时间
	cleanupCh chan struct{}
}

// NewSafeMap 创建并发安全的 map
func NewSafeMap(maxAge time.Duration) *SafeMap {
	sm := &SafeMap{
		data:      make(map[int64]time.Time),
		maxAge:    maxAge,
		cleanupCh: make(chan struct{}),
	}

	// 启动自动清理协程
	go sm.autoCleanup()

	return sm
}

// Set 设置值
func (sm *SafeMap) Set(key int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.data[key] = time.Now()
}

// Has 检查是否存在
func (sm *SafeMap) Has(key int64) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	_, exists := sm.data[key]
	return exists
}

// Delete 删除值
func (sm *SafeMap) Delete(key int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.data, key)
}

// Size 获取大小
func (sm *SafeMap) Size() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return len(sm.data)
}

// Clear 清空所有数据
func (sm *SafeMap) Clear() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.data = make(map[int64]time.Time)
}

// autoCleanup 自动清理过期条目
func (sm *SafeMap) autoCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanup()
		case <-sm.cleanupCh:
			return
		}
	}
}

// cleanup 清理过期条目
func (sm *SafeMap) cleanup() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	expired := 0

	for key, timestamp := range sm.data {
		if now.Sub(timestamp) > sm.maxAge {
			delete(sm.data, key)
			expired++
		}
	}

	if expired > 0 {
		// 日志可以在外部记录
	}
}

// Stop 停止自动清理
func (sm *SafeMap) Stop() {
	close(sm.cleanupCh)
}



