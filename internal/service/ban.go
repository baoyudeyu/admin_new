package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"
	"admin-bot/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
)

// BanService 拉黑服务
type BanService struct{}

// NewBanService 创建拉黑服务
func NewBanService() *BanService {
	return &BanService{}
}

// BanUser 拉黑用户
func (s *BanService) BanUser(userID int64, username, fullName string, groupID int64, groupName string,
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

	ban := &models.Blacklist{
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

	err := database.DB.Create(ban).Error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"用户ID": userID,
			"群组ID": groupID,
			"错误信息": err.Error(),
		}).Error("❌ 保存拉黑记录失败")
	}
	return err
}

// UnbanUser 解除拉黑
func (s *BanService) UnbanUser(userID int64, reason string, unbanBy int64) error {
	now := time.Now()
	return database.DB.Model(&models.Blacklist{}).
		Where("user_id = ? AND status = 1", userID).
		Updates(map[string]interface{}{
			"status":       0,
			"unban_reason": reason,
			"unban_at":     now,
			"unban_by":     unbanBy,
		}).Error
}

// IsUserBanned 检查用户是否被拉黑
func (s *BanService) IsUserBanned(userID int64) (bool, *models.Blacklist, error) {
	var ban models.Blacklist
	err := database.DB.Where("user_id = ? AND status = 1", userID).
		Order("created_at DESC").
		First(&ban).Error

	if err != nil {
		if err.Error() == "record not found" {
			return false, nil, nil
		}
		return false, nil, err
	}

	// 检查是否过期
	if ban.IsExpired() {
		return false, &ban, nil
	}

	return true, &ban, nil
}

// GetActiveBans 获取所有生效中的拉黑记录
func (s *BanService) GetActiveBans() ([]models.Blacklist, error) {
	var bans []models.Blacklist
	err := database.DB.Where("status = 1").Find(&bans).Error
	return bans, err
}

// GetExpiredBans 获取已过期但状态仍为1的记录
func (s *BanService) GetExpiredBans() ([]models.Blacklist, error) {
	var bans []models.Blacklist
	now := time.Now()
	err := database.DB.Where("status = 1 AND expire_at IS NOT NULL AND expire_at <= ?", now).
		Find(&bans).Error
	return bans, err
}

// AutoUnban 自动解除拉黑
func (s *BanService) AutoUnban(banID int64) error {
	now := time.Now()
	return database.DB.Model(&models.Blacklist{}).
		Where("id = ?", banID).
		Updates(map[string]interface{}{
			"status":       0,
			"unban_reason": "到期自动解除",
			"unban_at":     now,
		}).Error
}

// GetUserBanHistory 获取用户拉黑历史
func (s *BanService) GetUserBanHistory(userID int64) ([]models.Blacklist, error) {
	var bans []models.Blacklist
	err := database.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bans).Error
	return bans, err
}
