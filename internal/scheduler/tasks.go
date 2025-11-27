package scheduler

import (
	"admin-bot/internal/cache"
	"admin-bot/internal/database"
	"admin-bot/internal/service"
	"admin-bot/internal/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron                *cron.Cron
	banService          *service.BanService
	muteService         *service.MuteService
	groupService        *service.GroupService
	notificationService *service.NotificationService
	bot                 *tgbotapi.BotAPI
	rateLimiter         *utils.RateLimiter
}

// NewScheduler 创建调度器
func NewScheduler(banService *service.BanService,
	muteService *service.MuteService,
	groupService *service.GroupService,
	notificationService *service.NotificationService,
	bot *tgbotapi.BotAPI,
	rateLimitPerGroup int) *Scheduler {

	return &Scheduler{
		cron:                cron.New(),
		banService:          banService,
		muteService:         muteService,
		groupService:        groupService,
		notificationService: notificationService,
		bot:                 bot,
		rateLimiter:         utils.NewRateLimiter(rateLimitPerGroup),
	}
}

// Start 启动调度器
func (s *Scheduler) Start(checkExpireInterval string) error {
	// 添加检查过期记录的任务
	_, err := s.cron.AddFunc(checkExpireInterval, s.checkExpiredRecords)
	if err != nil {
		return err
	}

	// 添加清理限流器的任务（每5分钟）
	_, err = s.cron.AddFunc("*/5 * * * *", s.cleanupLimiters)
	if err != nil {
		return err
	}

	// 添加数据库健康检查任务（每5分钟）
	_, err = s.cron.AddFunc("*/5 * * * *", s.checkDatabaseHealth)
	if err != nil {
		return err
	}

	s.cron.Start()
	logrus.WithField("tasks", len(s.cron.Entries())).Debug("Scheduler tasks registered")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	logrus.Info("⏹️  定时任务已停止")
}

// checkExpiredRecords 检查过期记录
func (s *Scheduler) checkExpiredRecords() {
	logrus.Debug("🔍 正在检查过期记录...")

	// 检查过期的拉黑记录
	s.checkExpiredBans()

	// 检查过期的禁言记录
	s.checkExpiredMutes()
}

// checkExpiredBans 检查过期的拉黑记录
func (s *Scheduler) checkExpiredBans() {
	expiredBans, err := s.banService.GetExpiredBans()
	if err != nil {
		logrus.Errorf("Failed to get expired bans: %v", err)
		return
	}

	if len(expiredBans) == 0 {
		return
	}

	logrus.WithField("数量", len(expiredBans)).Info("🔓 发现过期的拉黑记录")

	// 获取所有授权群组
	authorizedGroups, err := s.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		return
	}

	for _, ban := range expiredBans {
		// 更新数据库状态
		err := s.banService.AutoUnban(ban.ID)
		if err != nil {
			logrus.Errorf("Failed to auto unban user %d: %v", ban.UserID, err)
			continue
		}

		// 在所有授权群组中解除拉黑
		for _, group := range authorizedGroups {
			s.rateLimiter.Wait(group.GroupID)

			unbanConfig := tgbotapi.UnbanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: ban.UserID,
				},
			}

			_, err = s.bot.Request(unbanConfig)
			if err != nil {
				logrus.Errorf("Failed to unban user %d in group %d: %v", ban.UserID, group.GroupID, err)
			}
		}

		// 发送自动解除通知（系统自动操作，operatorID 使用 0，groupUsername为空因为是定时任务）
		s.notificationService.SendUnbanNotification(ban.GroupID, ban.GroupName, "",
			ban.FullName, ban.UserID, "到期自动解除", "系统", 0)

		logrus.WithFields(logrus.Fields{
			"用户ID": ban.UserID,
			"用户名":  ban.FullName,
		}).Info("✅ 已自动解除拉黑")
	}
}

// checkExpiredMutes 检查过期的禁言记录
func (s *Scheduler) checkExpiredMutes() {
	expiredMutes, err := s.muteService.GetExpiredMutes()
	if err != nil {
		logrus.Errorf("Failed to get expired mutes: %v", err)
		return
	}

	if len(expiredMutes) == 0 {
		return
	}

	logrus.WithField("数量", len(expiredMutes)).Info("🔊 发现过期的禁言记录")

	// 获取所有授权群组
	authorizedGroups, err := s.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		return
	}

	for _, mute := range expiredMutes {
		// 更新数据库状态
		err := s.muteService.AutoUnmute(mute.ID)
		if err != nil {
			logrus.Errorf("Failed to auto unmute user %d: %v", mute.UserID, err)
			continue
		}

		// 在所有授权群组中解除禁言
		for _, group := range authorizedGroups {
			s.rateLimiter.Wait(group.GroupID)

			restrictConfig := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: mute.UserID,
				},
				Permissions: &tgbotapi.ChatPermissions{
					CanSendMessages:       true,
					CanSendMediaMessages:  true,
					CanSendPolls:          true,
					CanSendOtherMessages:  true,
					CanAddWebPagePreviews: true,
					CanChangeInfo:         false,
					CanInviteUsers:        false,
					CanPinMessages:        false,
				},
			}

			_, err = s.bot.Request(restrictConfig)
			if err != nil {
				logrus.Errorf("Failed to unmute user %d in group %d: %v", mute.UserID, group.GroupID, err)
			}
		}

		// 发送自动解除通知（系统自动操作，operatorID 使用 0，groupUsername为空因为是定时任务）
		s.notificationService.SendUnmuteNotification(mute.GroupID, mute.GroupName, "",
			mute.FullName, mute.UserID, "到期自动解除", "系统", 0)

		logrus.WithFields(logrus.Fields{
			"用户ID": mute.UserID,
			"用户名":  mute.FullName,
		}).Info("✅ 已自动解除禁言")
	}
}

// cleanupLimiters 清理限流器
func (s *Scheduler) cleanupLimiters() {
	s.rateLimiter.CleanupOldLimiters()
	logrus.Debug("🧹 已清理旧的限流器")
}

// checkDatabaseHealth 检查数据库连接健康状态（增强版：自动重连 + 缓存刷新）
func (s *Scheduler) checkDatabaseHealth() {
	logrus.Debug("🏥 正在检查数据库连接健康状态...")
	
	// 1. 尝试 ping 数据库（带重试）
	err := database.PingDBWithRetry(3)
	if err != nil {
		logrus.Errorf("❌ 数据库健康检查失败: %v", err)
		logrus.Warn("⚠️  数据库连接异常，授权检查将使用缓存宽容策略")
		
		// 标记缓存为过期，下次查询时会触发刷新
		authCache := cache.GetAuthCache()
		cacheStatus := authCache.GetCacheStatus()
		logrus.WithFields(logrus.Fields{
			"缓存状态": cacheStatus,
		}).Info("💾 当前授权缓存状态")
		
		return
	}

	// 2. 数据库连接正常，获取连接池统计信息
	stats := database.GetDBStats()
	logrus.WithField("连接池状态", stats).Debug("✅ 数据库连接正常")

	// 3. 定期刷新授权缓存（每次健康检查时）
	go func() {
		groupService := service.NewGroupService()
		groupService.RefreshAuthCache()
	}()
}
