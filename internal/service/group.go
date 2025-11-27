package service

import (
	"admin-bot/internal/cache"
	"admin-bot/internal/database"
	"admin-bot/internal/models"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GroupService 群组服务
type GroupService struct{}

// NewGroupService 创建群组服务
func NewGroupService() *GroupService {
	return &GroupService{}
}

// IsAuthorized 检查群组是否已授权（带缓存和重试）
func (s *GroupService) IsAuthorized(groupID int64) (bool, error) {
	// 1. 先检查缓存
	authCache := cache.GetAuthCache()
	authorized, cached := authCache.IsGroupAuthorized(groupID)
	if cached {
		// 缓存命中，直接返回
		logrus.WithField("群组ID", groupID).Debug("✅ 从缓存读取授权状态")
		return authorized, nil
	}

	// 2. 缓存未命中或过期，查询数据库（带重试）
	logrus.WithField("群组ID", groupID).Debug("🔍 缓存未命中，查询数据库")

	var count int64
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		err = database.DB.Model(&models.AuthorizedGroup{}).
			Where("group_id = ?", groupID).
			Count(&count).Error

		if err == nil {
			// 查询成功
			result := count > 0

			// 如果缓存过期，触发后台刷新
			if !cached {
				go s.RefreshAuthCache()
			}

			return result, nil
		}

		// 查询失败，记录日志
		logrus.WithFields(logrus.Fields{
			"群组ID": groupID,
			"重试次数": i + 1,
			"错误":   err.Error(),
		}).Warn("⚠️ 数据库查询失败")

		// 最后一次重试前稍等
		if i < maxRetries-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}

	// 3. 所有重试都失败，返回错误
	logrus.WithFields(logrus.Fields{
		"群组ID": groupID,
		"重试次数": maxRetries,
		"最后错误": err.Error(),
	}).Error("❌ 数据库查询失败，已达最大重试次数")

	return false, err
}

// RefreshAuthCache 刷新授权缓存
func (s *GroupService) RefreshAuthCache() {
	groups, err := s.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("❌ 刷新授权缓存失败: %v", err)
		return
	}

	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.GroupID)
	}

	authCache := cache.GetAuthCache()
	authCache.SetAuthorizedGroups(groupIDs)

	logrus.WithField("群组数", len(groupIDs)).Info("♻️ 授权缓存已刷新")
}

// AddAuthorizedGroup 添加授权群组
func (s *GroupService) AddAuthorizedGroup(groupID int64, groupName string) error {
	// 先检查是否已存在
	existingGroup, err := s.GetAuthorizedGroup(groupID)
	if err != nil {
		return err
	}
	if existingGroup != nil {
		return errors.New("该群组已在授权列表中")
	}

	// 创建新的授权群组
	group := &models.AuthorizedGroup{
		GroupID:   groupID,
		GroupName: groupName,
	}
	err = database.DB.Create(group).Error
	if err != nil {
		return err
	}

	// 完全刷新缓存，确保数据库和缓存同步
	logrus.WithField("群组ID", groupID).Info("✅ 已添加授权群组，正在刷新缓存...")
	go s.RefreshAuthCache()

	return nil
}

// AddAuthorizedGroupWithUsername 添加授权群组（包含用户名）
func (s *GroupService) AddAuthorizedGroupWithUsername(groupID int64, groupName, username string) error {
	// 先检查是否已存在
	existingGroup, err := s.GetAuthorizedGroup(groupID)
	if err != nil {
		return err
	}
	if existingGroup != nil {
		return errors.New("该群组已在授权列表中")
	}

	// 创建新的授权群组
	group := &models.AuthorizedGroup{
		GroupID:   groupID,
		GroupName: groupName,
		Username:  username,
	}
	err = database.DB.Create(group).Error
	if err != nil {
		return err
	}

	// 完全刷新缓存，确保数据库和缓存同步
	logrus.WithField("群组ID", groupID).Info("✅ 已添加授权群组，正在刷新缓存...")
	go s.RefreshAuthCache()

	return nil
}

// RemoveAuthorizedGroup 移除授权群组
func (s *GroupService) RemoveAuthorizedGroup(groupID int64) error {
	err := database.DB.Where("group_id = ?", groupID).
		Delete(&models.AuthorizedGroup{}).Error
	if err != nil {
		return err
	}

	// 完全刷新缓存，确保数据库和缓存同步
	logrus.WithField("群组ID", groupID).Info("✅ 已删除授权群组，正在刷新缓存...")
	go s.RefreshAuthCache()

	return nil
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
