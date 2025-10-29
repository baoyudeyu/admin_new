package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UserCacheService 用户缓存服务
type UserCacheService struct{}

// NewUserCacheService 创建用户缓存服务
func NewUserCacheService() *UserCacheService {
	return &UserCacheService{}
}

// SaveOrUpdateUser 保存或更新用户信息
func (s *UserCacheService) SaveOrUpdateUser(userID int64, username, firstName, lastName string) error {
	db := database.GetDB()

	// 如果没有用户名，不保存
	if username == "" {
		return nil
	}

	userCache := models.UserCache{
		UserID:    userID,
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
	}

	// 使用 GORM 的 Upsert 功能
	err := db.Where("user_id = ?", userID).
		Assign(models.UserCache{
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		}).
		FirstOrCreate(&userCache).Error

	if err != nil {
		logrus.Errorf("保存用户缓存失败: %v", err)
		return err
	}

	logrus.Debugf("✓ 用户缓存已更新: @%s (ID: %d)", username, userID)
	return nil
}

// GetUserIDByUsername 通过用户名获取用户ID
func (s *UserCacheService) GetUserIDByUsername(username string) (int64, error) {
	db := database.GetDB()

	var userCache models.UserCache
	err := db.Where("username = ?", username).First(&userCache).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logrus.Debugf("缓存中未找到用户: @%s", username)
			return 0, err
		}
		logrus.Errorf("查询用户缓存失败: %v", err)
		return 0, err
	}

	logrus.Debugf("✓ 从缓存获取用户ID: @%s -> %d", username, userCache.UserID)
	return userCache.UserID, nil
}

// GetUserByID 通过用户ID获取用户信息
func (s *UserCacheService) GetUserByID(userID int64) (*models.UserCache, error) {
	db := database.GetDB()

	var userCache models.UserCache
	err := db.Where("user_id = ?", userID).First(&userCache).Error

	if err != nil {
		return nil, err
	}

	return &userCache, nil
}
