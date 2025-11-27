package models

import (
	"time"
)

// Blacklist 拉黑记录表
type Blacklist struct {
	ID           int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64      `gorm:"index;not null" json:"user_id"`
	Username     string     `gorm:"type:varchar(255)" json:"username"`
	FullName     string     `gorm:"type:varchar(255)" json:"full_name"`
	GroupID      int64      `gorm:"not null" json:"group_id"`
	GroupName    string     `gorm:"type:varchar(255)" json:"group_name"`
	OperatorID   int64      `gorm:"not null" json:"operator_id"`
	OperatorName string     `gorm:"type:varchar(255)" json:"operator_name"`
	Reason       string     `gorm:"type:text" json:"reason"`
	Duration     *int       `json:"duration"` // 秒数，NULL 表示永久
	ExpireAt     *time.Time `gorm:"index" json:"expire_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	Status       int8       `gorm:"default:1;index" json:"status"` // 1=生效中，0=已解除
	UnbanReason  string     `gorm:"type:text" json:"unban_reason"`
	UnbanAt      *time.Time `json:"unban_at"`
	UnbanBy      *int64     `json:"unban_by"`
}

// TableName 指定表名
func (Blacklist) TableName() string {
	return "blacklist"
}

// IsActive 是否生效中
func (b *Blacklist) IsActive() bool {
	if b.Status != 1 {
		return false
	}
	if b.ExpireAt != nil && time.Now().After(*b.ExpireAt) {
		return false
	}
	return true
}

// IsExpired 是否已过期
func (b *Blacklist) IsExpired() bool {
	if b.ExpireAt == nil {
		return false
	}
	return time.Now().After(*b.ExpireAt)
}
