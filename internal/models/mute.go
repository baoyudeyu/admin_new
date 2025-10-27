package models

import (
	"time"
)

// MuteList 禁言记录表
type MuteList struct {
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
	UnmuteReason string     `gorm:"type:text" json:"unmute_reason"`
	UnmuteAt     *time.Time `json:"unmute_at"`
	UnmuteBy     *int64     `json:"unmute_by"`
}

// TableName 指定表名
func (MuteList) TableName() string {
	return "mute_list"
}

// IsActive 是否生效中
func (m *MuteList) IsActive() bool {
	if m.Status != 1 {
		return false
	}
	if m.ExpireAt != nil && time.Now().After(*m.ExpireAt) {
		return false
	}
	return true
}

// IsExpired 是否已过期
func (m *MuteList) IsExpired() bool {
	if m.ExpireAt == nil {
		return false
	}
	return time.Now().After(*m.ExpireAt)
}
