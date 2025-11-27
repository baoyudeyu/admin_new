package models

import (
	"time"
)

// OperationLog 操作日志表
type OperationLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OperationType  string    `gorm:"type:varchar(50);index;not null" json:"operation_type"` // ban/unban/mute/unmute/kick
	TargetUserID   int64     `gorm:"index;not null" json:"target_user_id"`
	TargetUsername string    `gorm:"type:varchar(255)" json:"target_username"`
	GroupID        int64     `gorm:"not null" json:"group_id"`
	GroupName      string    `gorm:"type:varchar(255)" json:"group_name"`
	OperatorID     int64     `gorm:"not null" json:"operator_id"`
	OperatorName   string    `gorm:"type:varchar(255)" json:"operator_name"`
	Reason         string    `gorm:"type:text" json:"reason"`
	Duration       *int      `json:"duration"`                 // 秒数
	Success        int8      `gorm:"default:1" json:"success"` // 1=成功，0=失败
	ErrorMsg       string    `gorm:"type:text" json:"error_msg"`
	CreatedAt      time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// TableName 指定表名
func (OperationLog) TableName() string {
	return "operation_logs"
}

// Operation types
const (
	OpTypeBan    = "ban"
	OpTypeUnban  = "unban"
	OpTypeMute   = "mute"
	OpTypeUnmute = "unmute"
	OpTypeKick   = "kick"
)
