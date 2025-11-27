package models

import "time"

// UserCache 用户缓存表（用于存储用户名和ID的映射关系）
type UserCache struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"uniqueIndex:idx_user_username;not null" json:"user_id"`           // 用户ID
	Username  string    `gorm:"uniqueIndex:idx_user_username;type:varchar(255)" json:"username"` // 用户名（不带@）
	FirstName string    `gorm:"type:varchar(255)" json:"first_name"`                             // 名字
	LastName  string    `gorm:"type:varchar(255)" json:"last_name"`                              // 姓氏
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`                                // 最后更新时间
}

// TableName 指定表名
func (UserCache) TableName() string {
	return "user_cache"
}
