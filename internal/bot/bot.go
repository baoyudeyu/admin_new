package bot

import (
	"admin-bot/internal/cache"
	"admin-bot/internal/config"
	"admin-bot/internal/scheduler"
	"admin-bot/internal/service"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Bot Telegram机器人
type Bot struct {
	api       *tgbotapi.BotAPI
	cfg       *config.Config
	handler   *Handler
	scheduler *scheduler.Scheduler
}

// NewBot 创建机器人实例
func NewBot(cfg *config.Config) (*Bot, error) {
	// 创建Bot API
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	if err != nil {
		return nil, err
	}

	bot.Debug = false
	logrus.WithFields(logrus.Fields{
		"用户名":   bot.Self.UserName,
		"机器人ID": bot.Self.ID,
	}).Info("🔐 机器人授权成功")

	// 输出隐私模式提示
	logrus.Warn("⚠️  如果机器人在公开群组中无法接收命令，请检查隐私模式设置")
	logrus.Warn("📝 使用 @BotFather 发送 /setprivacy 并选择 Disable")

	// 初始化授权缓存（30分钟 TTL，授权很少变更）
	logrus.Info("💾 正在初始化授权缓存...")
	cache.InitAuthCache(30 * time.Minute)

	// 创建服务
	banService := service.NewBanService()
	muteService := service.NewMuteService()
	groupService := service.NewGroupService()
	adminService := service.NewAdminService()
	logService := service.NewLogService()
	userCacheService := service.NewUserCacheService()
	notificationService := service.NewNotificationService(bot,
		cfg.Telegram.NotificationChannelID,
		cfg.Telegram.AuthorIDs)

	// 预加载授权群组到缓存
	logrus.Info("🔄 正在预加载授权群组...")
	err = preloadAuthCache(groupService, cfg.Telegram.NotificationChannelID)
	if err != nil {
		logrus.Warnf("⚠️  预加载授权缓存失败: %v（将在首次查询时加载）", err)
	} else {
		logrus.Info("✅ 授权缓存预加载完成")
	}

	// 创建权限检查器
	permissionChecker := NewPermissionChecker(cfg, adminService, groupService, bot)

	// 创建处理器
	handler := NewHandler(bot, cfg, permissionChecker,
		banService, muteService, groupService, adminService,
		logService, notificationService, userCacheService)

	// 创建调度器
	taskScheduler := scheduler.NewScheduler(banService, muteService,
		groupService, notificationService, bot,
		cfg.System.RateLimitPerGroup)

	return &Bot{
		api:       bot,
		cfg:       cfg,
		handler:   handler,
		scheduler: taskScheduler,
	}, nil
}

// preloadAuthCache 预加载授权缓存
func preloadAuthCache(groupService *service.GroupService, notificationChannelID int64) error {
	// 获取所有授权群组
	groups, err := groupService.GetAuthorizedGroups()
	if err != nil {
		return err
	}

	// 提取群组ID列表
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.GroupID)
	}

	// 加载到缓存
	authCache := cache.GetAuthCache()
	authCache.SetAuthorizedGroups(groupIDs)

	// 设置通知频道ID
	if notificationChannelID != 0 {
		authCache.SetNotificationChannel(notificationChannelID)
		logrus.WithField("频道ID", notificationChannelID).Info("✅ 通知频道已加载到缓存")
	}

	logrus.WithField("授权群组数", len(groupIDs)).Info("✅ 授权群组已加载到缓存")
	return nil
}

// Start 启动机器人
func (b *Bot) Start() error {
	// 启动调度器
	logrus.Info("⏰ 正在启动定时任务...")
	err := b.scheduler.Start(b.cfg.Scheduler.CheckExpireInterval)
	if err != nil {
		return err
	}
	logrus.WithField("检查间隔", b.cfg.Scheduler.CheckExpireInterval).Info("✅ 定时任务已启动")

	// 配置更新 - 使用 -1 来只获取新消息，忽略历史消息
	u := tgbotapi.NewUpdate(-1)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	logrus.Info("📡 开始监听 Telegram 更新...")

	// 处理更新
	for update := range updates {
		go b.handleUpdate(update)
	}

	return nil
}

// Stop 停止机器人
func (b *Bot) Stop() {
	b.scheduler.Stop()
	b.api.StopReceivingUpdates()
	logrus.Info("🛑 机器人已停止")
}

// handleUpdate 处理更新
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// 处理消息
	if update.Message != nil {
		// 调试日志：记录所有消息
		logrus.WithFields(logrus.Fields{
			"消息ID":  update.Message.MessageID,
			"消息文本":  update.Message.Text,
			"是否为命令": update.Message.IsCommand(),
			"聊天类型":  update.Message.Chat.Type,
			"聊天ID":  update.Message.Chat.ID,
			"聊天标题":  update.Message.Chat.Title,
			"聊天用户名": update.Message.Chat.UserName,
		}).Debug("🔍 收到消息")

		// 自动缓存发言用户信息
		if update.Message.From != nil && !update.Message.From.IsBot {
			b.handler.CacheUserInfo(update.Message.From)
		}

		// 检查新成员
		if len(update.Message.NewChatMembers) > 0 {
			// 检查是否有机器人自己被添加
			b.handler.CheckBotAddedToGroup(update.Message, b.api.Self.ID)
			// 检查新成员是否在黑名单
			b.handler.CheckNewMember(update.Message)
			return
		}

		// 处理命令
		if update.Message.IsCommand() {
			b.handler.HandleMessage(update.Message)
		} else {
			// 处理文本消息（对话模式）
			b.handler.HandleTextMessage(update.Message)
		}

		// 检查是否为未授权群组（在处理完命令后再检查，避免阻止命令执行）
		if update.Message.Chat.IsGroup() || update.Message.Chat.IsSuperGroup() {
			b.handler.CheckUnauthorizedGroup(update.Message)
		}
	}

	// 处理回调查询
	if update.CallbackQuery != nil {
		b.handler.HandleCallback(update.CallbackQuery)
	}
}

// GetAPI 获取Bot API
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}
