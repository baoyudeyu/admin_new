package cache

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AuthCache 授权缓存
type AuthCache struct {
	authorizedGroups map[int64]bool // 群组ID -> 是否授权
	notificationChan int64          // 通知频道ID
	mutex            sync.RWMutex   // 读写锁
	lastUpdate       time.Time      // 最后更新时间
	ttl              time.Duration  // 缓存过期时间
}

var (
	globalAuthCache *AuthCache
	once            sync.Once
)

// InitAuthCache 初始化授权缓存
func InitAuthCache(ttl time.Duration) *AuthCache {
	once.Do(func() {
		globalAuthCache = &AuthCache{
			authorizedGroups: make(map[int64]bool),
			notificationChan: 0,
			lastUpdate:       time.Time{},
			ttl:              ttl,
		}
		logrus.WithField("TTL", ttl).Info("✅ 授权缓存已初始化")
	})
	return globalAuthCache
}

// GetAuthCache 获取全局授权缓存实例
func GetAuthCache() *AuthCache {
	if globalAuthCache == nil {
		// 默认 TTL 30分钟（授权很少变更，可以使用较长缓存）
		return InitAuthCache(30 * time.Minute)
	}
	return globalAuthCache
}

// SetAuthorizedGroups 设置授权群组列表（批量更新）
func (c *AuthCache) SetAuthorizedGroups(groupIDs []int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 清空旧缓存
	c.authorizedGroups = make(map[int64]bool, len(groupIDs))

	// 添加所有授权群组
	for _, groupID := range groupIDs {
		c.authorizedGroups[groupID] = true
	}

	c.lastUpdate = time.Now()

	logrus.WithFields(logrus.Fields{
		"授权群组数": len(groupIDs),
		"更新时间":  c.lastUpdate.Format("2006-01-02 15:04:05"),
	}).Info("✅ 授权缓存已更新")
}

// AddAuthorizedGroup 添加单个授权群组
func (c *AuthCache) AddAuthorizedGroup(groupID int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.authorizedGroups[groupID] = true
	c.lastUpdate = time.Now()

	logrus.WithField("群组ID", groupID).Debug("✅ 已添加授权群组到缓存")
}

// RemoveAuthorizedGroup 移除单个授权群组
func (c *AuthCache) RemoveAuthorizedGroup(groupID int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.authorizedGroups, groupID)
	c.lastUpdate = time.Now()

	logrus.WithField("群组ID", groupID).Debug("🗑️ 已从缓存移除授权群组")
}

// IsGroupAuthorized 检查群组是否已授权（从缓存读取）
func (c *AuthCache) IsGroupAuthorized(groupID int64) (authorized bool, cached bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// 检查缓存是否过期
	if time.Since(c.lastUpdate) > c.ttl {
		logrus.Debug("⚠️ 授权缓存已过期")
		return false, false
	}

	// 从缓存中查询
	authorized, exists := c.authorizedGroups[groupID]
	if !exists {
		// 缓存中不存在，返回未授权但标记为缓存有效
		return false, true
	}

	return authorized, true
}

// SetNotificationChannel 设置通知频道ID
func (c *AuthCache) SetNotificationChannel(channelID int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.notificationChan = channelID
	logrus.WithField("频道ID", channelID).Debug("✅ 已更新通知频道ID到缓存")
}

// GetNotificationChannel 获取通知频道ID
func (c *AuthCache) GetNotificationChannel() int64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.notificationChan
}

// IsNotificationChannel 检查是否为通知频道
func (c *AuthCache) IsNotificationChannel(chatID int64) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.notificationChan != 0 && c.notificationChan == chatID
}

// GetCacheStatus 获取缓存状态
func (c *AuthCache) GetCacheStatus() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"授权群组数": len(c.authorizedGroups),
		"最后更新":  c.lastUpdate.Format("2006-01-02 15:04:05"),
		"缓存年龄":  time.Since(c.lastUpdate).String(),
		"是否过期":  time.Since(c.lastUpdate) > c.ttl,
		"TTL":   c.ttl.String(),
	}
}

// InvalidateCache 使缓存失效（强制下次重新加载）
func (c *AuthCache) InvalidateCache() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.lastUpdate = time.Time{}
	logrus.Info("♻️ 授权缓存已失效，将在下次查询时重新加载")
}

// GetAuthorizedGroupCount 获取授权群组数量
func (c *AuthCache) GetAuthorizedGroupCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.authorizedGroups)
}
