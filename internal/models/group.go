package models

import (
	"time"
)

// AuthorizedGroup 授权群组表
type AuthorizedGroup struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupID   int64     `gorm:"uniqueIndex;not null" json:"group_id"`
	GroupName string    `gorm:"type:varchar(255)" json:"group_name"`
	Username  string    `gorm:"type:varchar(255)" json:"username"` // 公开群组的用户名
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (AuthorizedGroup) TableName() string {
	return "authorized_groups"
}
