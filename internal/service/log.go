package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"
)

// LogService 日志服务
type LogService struct{}

// NewLogService 创建日志服务
func NewLogService() *LogService {
	return &LogService{}
}

// LogOperation 记录操作日志
func (s *LogService) LogOperation(opType string, targetUserID int64, targetUsername string,
	groupID int64, groupName string, operatorID int64, operatorName string,
	reason string, duration *int, success bool, errorMsg string) error {

	log := &models.OperationLog{
		OperationType:  opType,
		TargetUserID:   targetUserID,
		TargetUsername: targetUsername,
		GroupID:        groupID,
		GroupName:      groupName,
		OperatorID:     operatorID,
		OperatorName:   operatorName,
		Reason:         reason,
		Duration:       duration,
		Success:        boolToInt8(success),
		ErrorMsg:       errorMsg,
	}

	return database.DB.Create(log).Error
}

// GetUserLogs 获取用户相关的操作日志
func (s *LogService) GetUserLogs(userID int64, limit int) ([]models.OperationLog, error) {
	var logs []models.OperationLog
	query := database.DB.Where("target_user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetGroupLogs 获取群组相关的操作日志
func (s *LogService) GetGroupLogs(groupID int64, limit int) ([]models.OperationLog, error) {
	var logs []models.OperationLog
	query := database.DB.Where("group_id = ?", groupID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetFailedLogs 获取失败的操作日志
func (s *LogService) GetFailedLogs(limit int) ([]models.OperationLog, error) {
	var logs []models.OperationLog
	query := database.DB.Where("success = 0").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

func boolToInt8(b bool) int8 {
	if b {
		return 1
	}
	return 0
}

