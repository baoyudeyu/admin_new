package models

import (
	"time"
)

// SystemConfig 系统配置表
type SystemConfig struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ConfigKey   string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"config_key"`
	ConfigValue string    `gorm:"type:text" json:"config_value"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_config"
}

// Config keys
const (
	ConfigKeyAdminEnabled          = "admin_enabled"
	ConfigKeyRateLimitPerGroup     = "rate_limit_per_group"
	ConfigKeyNotificationChannelID = "notification_channel_id"
	ConfigKeyAuthorID              = "author_id"
)
