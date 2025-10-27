package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"
	"errors"

	"gorm.io/gorm"
)

// AdminService 管理员服务
type AdminService struct{}

// NewAdminService 创建管理员服务
func NewAdminService() *AdminService {
	return &AdminService{}
}

// IsGlobalAdmin 检查是否为全局管理员
func (s *AdminService) IsGlobalAdmin(userID int64) (bool, error) {
	var count int64
	err := database.DB.Model(&models.GlobalAdmin{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddGlobalAdmin 添加全局管理员
func (s *AdminService) AddGlobalAdmin(userID int64, username, fullName string, addedBy int64) error {
	admin := &models.GlobalAdmin{
		UserID:   userID,
		Username: username,
		FullName: fullName,
		AddedBy:  addedBy,
	}
	return database.DB.Create(admin).Error
}

// RemoveGlobalAdmin 移除全局管理员
func (s *AdminService) RemoveGlobalAdmin(userID int64) error {
	return database.DB.Where("user_id = ?", userID).
		Delete(&models.GlobalAdmin{}).Error
}

// GetGlobalAdmins 获取所有全局管理员
func (s *AdminService) GetGlobalAdmins() ([]models.GlobalAdmin, error) {
	var admins []models.GlobalAdmin
	err := database.DB.Find(&admins).Error
	return admins, err
}

// GetGlobalAdmin 获取指定全局管理员
func (s *AdminService) GetGlobalAdmin(userID int64) (*models.GlobalAdmin, error) {
	var admin models.GlobalAdmin
	err := database.DB.Where("user_id = ?", userID).First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

