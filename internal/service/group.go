package service

import (
	"admin-bot/internal/database"
	"admin-bot/internal/models"
	"errors"

	"gorm.io/gorm"
)

// GroupService 群组服务
type GroupService struct{}

// NewGroupService 创建群组服务
func NewGroupService() *GroupService {
	return &GroupService{}
}

// IsAuthorized 检查群组是否已授权
func (s *GroupService) IsAuthorized(groupID int64) (bool, error) {
	var count int64
	err := database.DB.Model(&models.AuthorizedGroup{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddAuthorizedGroup 添加授权群组
func (s *GroupService) AddAuthorizedGroup(groupID int64, groupName string) error {
	group := &models.AuthorizedGroup{
		GroupID:   groupID,
		GroupName: groupName,
	}
	return database.DB.Create(group).Error
}

// AddAuthorizedGroupWithUsername 添加授权群组（包含用户名）
func (s *GroupService) AddAuthorizedGroupWithUsername(groupID int64, groupName, username string) error {
	group := &models.AuthorizedGroup{
		GroupID:   groupID,
		GroupName: groupName,
		Username:  username,
	}
	return database.DB.Create(group).Error
}

// RemoveAuthorizedGroup 移除授权群组
func (s *GroupService) RemoveAuthorizedGroup(groupID int64) error {
	return database.DB.Where("group_id = ?", groupID).
		Delete(&models.AuthorizedGroup{}).Error
}

// GetAuthorizedGroups 获取所有授权群组
func (s *GroupService) GetAuthorizedGroups() ([]models.AuthorizedGroup, error) {
	var groups []models.AuthorizedGroup
	err := database.DB.Find(&groups).Error
	return groups, err
}

// GetAuthorizedGroup 获取指定授权群组
func (s *GroupService) GetAuthorizedGroup(groupID int64) (*models.AuthorizedGroup, error) {
	var group models.AuthorizedGroup
	err := database.DB.Where("group_id = ?", groupID).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// UpdateGroupName 更新群组名称
func (s *GroupService) UpdateGroupName(groupID int64, groupName string) error {
	return database.DB.Model(&models.AuthorizedGroup{}).
		Where("group_id = ?", groupID).
		Update("group_name", groupName).Error
}

// UpdateGroupInfo 更新群组信息（名称和用户名）
func (s *GroupService) UpdateGroupInfo(groupID int64, groupName, username string) error {
	updates := map[string]interface{}{
		"group_name": groupName,
		"username":   username,
	}
	return database.DB.Model(&models.AuthorizedGroup{}).
		Where("group_id = ?", groupID).
		Updates(updates).Error
}
