package models

import (
	"time"
)

// GlobalAdmin 全局管理员表
type GlobalAdmin struct {
	ID       int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	Username string    `gorm:"type:varchar(255)" json:"username"`
	FullName string    `gorm:"type:varchar(255)" json:"full_name"`
	AddedAt  time.Time `gorm:"autoCreateTime" json:"added_at"`
	AddedBy  int64     `json:"added_by"`
}

// TableName 指定表名
func (GlobalAdmin) TableName() string {
	return "global_admins"
}
