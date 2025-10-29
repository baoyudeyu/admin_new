package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"
	"admin-bot/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
)

// MuteService 禁言服务
type MuteService struct{}

// NewMuteService 创建禁言服务
func NewMuteService() *MuteService {
	return &MuteService{}
}

// MuteUser 禁言用户
func (s *MuteService) MuteUser(userID int64, username, fullName string, groupID int64, groupName string,
	operatorID int64, operatorName string, reason string, duration int) error {

	// 安全处理字符串，防止编码问题
	username = utils.SafeUsername(username)
	fullName = utils.SafeFullName(fullName)
	groupName = utils.SafeGroupName(groupName)
	operatorName = utils.SafeFullName(operatorName)
	reason = utils.SafeReason(reason)

	expireAt := utils.CalculateExpireTime(duration)
	var durationPtr *int
	if duration > 0 {
		durationPtr = &duration
	}

	mute := &models.MuteList{
		UserID:       userID,
		Username:     username,
		FullName:     fullName,
		GroupID:      groupID,
		GroupName:    groupName,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Reason:       reason,
		Duration:     durationPtr,
		ExpireAt:     expireAt,
		Status:       1,
	}

	err := database.DB.Create(mute).Error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"用户ID": userID,
			"群组ID": groupID,
			"错误信息": err.Error(),
		}).Error("❌ 保存禁言记录失败")
	}
	return err
}

// UnmuteUser 解除禁言
func (s *MuteService) UnmuteUser(userID int64, reason string, unmuteBy int64) error {
	now := time.Now()
	return database.DB.Model(&models.MuteList{}).
		Where("user_id = ? AND status = 1", userID).
		Updates(map[string]interface{}{
			"status":        0,
			"unmute_reason": reason,
			"unmute_at":     now,
			"unmute_by":     unmuteBy,
		}).Error
}

// IsUserMuted 检查用户是否被禁言
func (s *MuteService) IsUserMuted(userID int64) (bool, *models.MuteList, error) {
	var mute models.MuteList
	err := database.DB.Where("user_id = ? AND status = 1", userID).
		Order("created_at DESC").
		First(&mute).Error

	if err != nil {
		if err.Error() == "record not found" {
			return false, nil, nil
		}
		return false, nil, err
	}

	// 检查是否过期
	if mute.IsExpired() {
		return false, &mute, nil
	}

	return true, &mute, nil
}

// GetActiveMutes 获取所有生效中的禁言记录
func (s *MuteService) GetActiveMutes() ([]models.MuteList, error) {
	var mutes []models.MuteList
	err := database.DB.Where("status = 1").Find(&mutes).Error
	return mutes, err
}

// GetExpiredMutes 获取已过期但状态仍为1的记录
func (s *MuteService) GetExpiredMutes() ([]models.MuteList, error) {
	var mutes []models.MuteList
	now := time.Now()
	err := database.DB.Where("status = 1 AND expire_at IS NOT NULL AND expire_at <= ?", now).
		Find(&mutes).Error
	return mutes, err
}

// AutoUnmute 自动解除禁言
func (s *MuteService) AutoUnmute(muteID int64) error {
	now := time.Now()
	return database.DB.Model(&models.MuteList{}).
		Where("id = ?", muteID).
		Updates(map[string]interface{}{
			"status":        0,
			"unmute_reason": "到期自动解除",
			"unmute_at":     now,
		}).Error
}

// GetUserMuteHistory 获取用户禁言历史
func (s *MuteService) GetUserMuteHistory(userID int64) ([]models.MuteList, error) {
	var mutes []models.MuteList
	err := database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&mutes).Error
	return mutes, err
}
