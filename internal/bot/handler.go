package bot

import (
	"admin-bot/internal/config"
	"admin-bot/internal/models"
	"admin-bot/internal/service"
	"admin-bot/internal/utils"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Handler Bot命令处理器
type Handler struct {
	bot                  *tgbotapi.BotAPI
	cfg                  *config.Config
	permissionChecker    *PermissionChecker
	banService           *service.BanService
	muteService          *service.MuteService
	groupService         *service.GroupService
	adminService         *service.AdminService
	logService           *service.LogService
	notificationService  *service.NotificationService
	rateLimiter          *utils.RateLimiter
	notifiedUnauthorized map[int64]bool // 记录已通知的未授权群组
}

// NewHandler 创建处理器
func NewHandler(bot *tgbotapi.BotAPI, cfg *config.Config,
	permissionChecker *PermissionChecker,
	banService *service.BanService,
	muteService *service.MuteService,
	groupService *service.GroupService,
	adminService *service.AdminService,
	logService *service.LogService,
	notificationService *service.NotificationService) *Handler {

	return &Handler{
		bot:                  bot,
		cfg:                  cfg,
		permissionChecker:    permissionChecker,
		banService:           banService,
		muteService:          muteService,
		groupService:         groupService,
		adminService:         adminService,
		logService:           logService,
		notificationService:  notificationService,
		rateLimiter:          utils.NewRateLimiter(cfg.System.RateLimitPerGroup),
		notifiedUnauthorized: make(map[int64]bool),
	}
}

// HandleMessage 处理消息
func (h *Handler) HandleMessage(message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	// 忽略机器人自己的消息
	if message.From.IsBot {
		return
	}

	// 检查是否为命令
	if !message.IsCommand() {
		return
	}

	command := message.Command()

	// 调试：记录原始命令和处理后的命令
	logrus.WithFields(logrus.Fields{
		"原始文本":  message.Text,
		"提取的命令": command,
		"是否为命令": message.IsCommand(),
		"命令参数":  message.CommandArguments(),
	}).Debug("🔍 命令解析")

	// 获取用户和群组信息
	userName := message.From.FirstName
	if message.From.LastName != "" {
		userName += " " + message.From.LastName
	}
	chatTitle := message.Chat.Title
	if chatTitle == "" {
		chatTitle = "Private Chat"
	}

	logrus.WithFields(logrus.Fields{
		"命令":   command,
		"用户":   userName,
		"用户ID": message.From.ID,
		"群组":   chatTitle,
		"群组ID": message.Chat.ID,
	}).Info("📨 收到命令")

	// 处理不同的命令
	switch command {
	case "start":
		h.handleStart(message)
	case "help":
		h.handleHelp(message)
	case "t":
		h.handleKick(message)
	case "lh":
		h.handleBan(message)
	case "unlh":
		h.handleUnban(message)
	case "jy":
		h.handleMute(message)
	case "unjy":
		h.handleUnmute(message)
	case "config":
		h.handleConfig(message)
	default:
		logrus.Debugf("Unknown command: %s", command)
	}
}

// handleStart 处理 /start 命令
func (h *Handler) handleStart(message *tgbotapi.Message) {
	text := "👋 欢迎使用多群组群管机器人\n\n" +
		"使用 /help 查看可用命令"
	h.sendReply(message.Chat.ID, message.MessageID, text)
}

// handleHelp 处理 /help 命令
func (h *Handler) handleHelp(message *tgbotapi.Message) {
	text := "📖 *命令列表*\n\n" +
		"*基础命令：*\n" +
		"/t - 踢出群组\n" +
		"/lh \\[时间\\] \\[理由\\] - 拉黑用户\n" +
		"/unlh \\[理由\\] - 解除拉黑\n" +
		"/jy \\[时间\\] \\[理由\\] - 禁言用户\n" +
		"/unjy \\[理由\\] - 解除禁言\n\n" +
		"*使用方式：*\n" +
		"\\- 引用回复目标用户的消息\n" +
		"\\- 或在命令后指定 @username\n\n" +
		"*时间单位：*\n" +
		"s=秒，m=分钟，h=小时，d=天\n\n" +
		"*示例：*\n" +
		"`/jy @user 10m 违规`\n" +
		"`/lh 1d 刷屏`"

	h.sendReply(message.Chat.ID, message.MessageID, text)
}

// handleKick 处理踢出命令
func (h *Handler) handleKick(message *tgbotapi.Message) {
	// 检查权限
	hasPermission, reason := h.permissionChecker.CheckPermission(message)
	if !hasPermission {
		logrus.WithFields(logrus.Fields{
			"用户ID": message.From.ID,
			"群组ID": message.Chat.ID,
			"原因":   reason,
		}).Warn("⛔ 权限检查失败")
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 您没有权限执行此操作")
		return
	}

	logrus.WithFields(logrus.Fields{
		"用户ID": message.From.ID,
		"群组ID": message.Chat.ID,
		"权限类型": reason,
	}).Debug("✅ 权限检查通过")

	// 解析命令
	params, err := ParseCommand(message, h.bot)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 获取操作人信息
	_, operatorName := GetUserInfo(message.From)
	groupName := GetChatTitle(message.Chat)
	groupUsername := GetChatUsername(message.Chat)

	successCount := 0
	failedCount := 0

	// 批量处理
	for _, targetUserID := range params.TargetUsers {
		// 限流
		h.rateLimiter.Wait(message.Chat.ID)

		// 获取目标用户信息
		chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		})

		if err != nil {
			logrus.Errorf("Failed to get chat member: %v", err)
			h.notificationService.SendErrorNotification(groupName, "踢出", fmt.Sprintf("%d", targetUserID),
				targetUserID, err.Error(), operatorName)
			continue
		}

		targetUsername, targetName := GetUserInfo(chatMember.User)

		// 执行踢出
		kickConfig := tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		}

		_, err = h.bot.Request(kickConfig)
		if err != nil {
			logrus.Errorf("Failed to kick user: %v", err)
			h.logService.LogOperation(models.OpTypeKick, targetUserID, targetUsername,
				message.Chat.ID, groupName, message.From.ID, operatorName,
				"", nil, false, err.Error())
			h.notificationService.SendErrorNotification(groupName, "踢出", targetName,
				targetUserID, err.Error(), operatorName)
			failedCount++
			continue
		}

		// 解除封禁（允许用户再次加入）
		unbanConfig := tgbotapi.UnbanChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		}
		h.bot.Request(unbanConfig)

		// 记录日志
		h.logService.LogOperation(models.OpTypeKick, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			"", nil, true, "")

		// 发送通知
		h.notificationService.SendKickNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, operatorName, message.From.ID)

		successCount++
	}

	// 发送操作结果反馈
	if failedCount == 0 {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 踢出操作成功")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 踢出操作完成，部分失败")
	}
}

// handleBan 处理拉黑命令
func (h *Handler) handleBan(message *tgbotapi.Message) {
	// 检查权限
	hasPermission, reason := h.permissionChecker.CheckPermission(message)
	if !hasPermission {
		logrus.WithFields(logrus.Fields{
			"用户ID": message.From.ID,
			"群组ID": message.Chat.ID,
			"原因":   reason,
		}).Warn("⛔ 权限检查失败")
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 您没有权限执行此操作")
		return
	}

	logrus.WithFields(logrus.Fields{
		"用户ID": message.From.ID,
		"群组ID": message.Chat.ID,
		"权限类型": reason,
	}).Debug("✅ 权限检查通过")

	// 解析命令
	params, err := ParseCommand(message, h.bot)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 获取操作人信息
	_, operatorName := GetUserInfo(message.From)
	groupName := GetChatTitle(message.Chat)
	groupUsername := GetChatUsername(message.Chat)

	// 获取所有授权群组
	authorizedGroups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		authorizedGroups = []models.AuthorizedGroup{}
	}

	successCount := 0
	failedCount := 0

	// 批量处理
	for _, targetUserID := range params.TargetUsers {
		// 限流
		h.rateLimiter.Wait(message.Chat.ID)

		// 获取目标用户信息
		chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		})

		if err != nil {
			logrus.Errorf("Failed to get chat member: %v", err)
			h.notificationService.SendErrorNotification(groupName, "拉黑", fmt.Sprintf("%d", targetUserID),
				targetUserID, err.Error(), operatorName)
			failedCount++
			continue
		}

		targetUsername, targetName := GetUserInfo(chatMember.User)

		// 保存到数据库
		err = h.banService.BanUser(targetUserID, targetUsername, targetName,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, params.Duration)

		if err != nil {
			logrus.Errorf("Failed to save ban record: %v", err)
			failedCount++
			continue
		}

		// 在所有授权群组中执行拉黑
		groupSuccessCount := 0
		for _, group := range authorizedGroups {
			h.rateLimiter.Wait(group.GroupID)

			kickConfig := tgbotapi.KickChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: targetUserID,
				},
			}

			if params.Duration > 0 {
				kickConfig.UntilDate = int64(params.Duration)
			}

			_, err = h.bot.Request(kickConfig)
			if err != nil {
				logrus.Errorf("Failed to ban user in group %d: %v", group.GroupID, err)
			} else {
				groupSuccessCount++
			}
		}

		// 记录日志
		durationPtr := &params.Duration
		h.logService.LogOperation(models.OpTypeBan, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, durationPtr, true, "")

		// 发送通知
		h.notificationService.SendBanNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, params.Duration, params.Reason, operatorName, message.From.ID)

		logrus.WithFields(logrus.Fields{
			"用户ID": targetUserID,
			"成功数量": groupSuccessCount,
			"总群组数": len(authorizedGroups),
		}).Info("✅ 用户已在多个群组中被拉黑")

		successCount++
	}

	// 发送操作结果反馈
	if failedCount == 0 {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 拉黑操作成功")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 拉黑操作完成，部分失败")
	}
}

// handleUnban 处理解除拉黑命令
func (h *Handler) handleUnban(message *tgbotapi.Message) {
	// 检查权限
	hasPermission, _ := h.permissionChecker.CheckPermission(message)
	if !hasPermission {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 您没有权限执行此操作")
		return
	}

	// 解析命令
	params, err := ParseCommand(message, h.bot)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 获取操作人信息
	_, operatorName := GetUserInfo(message.From)
	groupName := GetChatTitle(message.Chat)
	groupUsername := GetChatUsername(message.Chat)

	// 获取所有授权群组
	authorizedGroups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		authorizedGroups = []models.AuthorizedGroup{}
	}

	successCount := 0
	failedCount := 0

	// 批量处理
	for _, targetUserID := range params.TargetUsers {
		// 限流
		h.rateLimiter.Wait(message.Chat.ID)

		// 获取目标用户信息
		chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		})

		var targetUsername, targetName string
		if err == nil {
			targetUsername, targetName = GetUserInfo(chatMember.User)
		} else {
			targetName = fmt.Sprintf("User_%d", targetUserID)
		}

		// 更新数据库
		err = h.banService.UnbanUser(targetUserID, params.Reason, message.From.ID)
		if err != nil {
			logrus.Errorf("Failed to update unban record: %v", err)
			failedCount++
			continue
		}

		// 在所有授权群组中解除拉黑
		for _, group := range authorizedGroups {
			h.rateLimiter.Wait(group.GroupID)

			unbanConfig := tgbotapi.UnbanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: targetUserID,
				},
			}

			_, err = h.bot.Request(unbanConfig)
			if err != nil {
				logrus.Errorf("Failed to unban user in group %d: %v", group.GroupID, err)
			}
		}

		// 记录日志
		h.logService.LogOperation(models.OpTypeUnban, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, nil, true, "")

		// 发送通知
		h.notificationService.SendUnbanNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, params.Reason, operatorName, message.From.ID)

		successCount++
	}

	// 发送操作结果反馈
	if failedCount == 0 {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 解除拉黑操作成功")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 解除拉黑操作完成，部分失败")
	}
}

// handleMute 处理禁言命令
func (h *Handler) handleMute(message *tgbotapi.Message) {
	// 检查权限
	hasPermission, reason := h.permissionChecker.CheckPermission(message)
	if !hasPermission {
		logrus.WithFields(logrus.Fields{
			"用户ID": message.From.ID,
			"群组ID": message.Chat.ID,
			"原因":   reason,
		}).Warn("⛔ 权限检查失败")
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 您没有权限执行此操作")
		return
	}

	logrus.WithFields(logrus.Fields{
		"用户ID": message.From.ID,
		"群组ID": message.Chat.ID,
		"权限类型": reason,
	}).Debug("✅ 权限检查通过")

	// 解析命令
	params, err := ParseCommand(message, h.bot)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 获取操作人信息
	_, operatorName := GetUserInfo(message.From)
	groupName := GetChatTitle(message.Chat)
	groupUsername := GetChatUsername(message.Chat)

	// 获取所有授权群组
	authorizedGroups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		authorizedGroups = []models.AuthorizedGroup{}
	}

	successCount := 0
	failedCount := 0

	// 批量处理
	for _, targetUserID := range params.TargetUsers {
		// 限流
		h.rateLimiter.Wait(message.Chat.ID)

		// 获取目标用户信息
		chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		})

		if err != nil {
			logrus.Errorf("Failed to get chat member: %v", err)
			h.notificationService.SendErrorNotification(groupName, "禁言", fmt.Sprintf("%d", targetUserID),
				targetUserID, err.Error(), operatorName)
			failedCount++
			continue
		}

		targetUsername, targetName := GetUserInfo(chatMember.User)

		// 保存到数据库
		err = h.muteService.MuteUser(targetUserID, targetUsername, targetName,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, params.Duration)

		if err != nil {
			logrus.Errorf("Failed to save mute record: %v", err)
			failedCount++
			continue
		}

		// 在所有授权群组中执行禁言
		for _, group := range authorizedGroups {
			h.rateLimiter.Wait(group.GroupID)

			restrictConfig := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: targetUserID,
				},
				Permissions: &tgbotapi.ChatPermissions{
					CanSendMessages: false,
				},
			}

			if params.Duration > 0 {
				restrictConfig.UntilDate = int64(params.Duration)
			}

			_, err = h.bot.Request(restrictConfig)
			if err != nil {
				logrus.Errorf("Failed to mute user in group %d: %v", group.GroupID, err)
			}
		}

		// 记录日志
		durationPtr := &params.Duration
		h.logService.LogOperation(models.OpTypeMute, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, durationPtr, true, "")

		// 发送通知
		h.notificationService.SendMuteNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, params.Duration, params.Reason, operatorName, message.From.ID)

		successCount++
	}

	// 发送操作结果反馈
	if failedCount == 0 {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 禁言操作成功")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 禁言操作完成，部分失败")
	}
}

// handleUnmute 处理解除禁言命令
func (h *Handler) handleUnmute(message *tgbotapi.Message) {
	// 检查权限
	hasPermission, _ := h.permissionChecker.CheckPermission(message)
	if !hasPermission {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 您没有权限执行此操作")
		return
	}

	// 解析命令
	params, err := ParseCommand(message, h.bot)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 获取操作人信息
	_, operatorName := GetUserInfo(message.From)
	groupName := GetChatTitle(message.Chat)
	groupUsername := GetChatUsername(message.Chat)

	// 获取所有授权群组
	authorizedGroups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		authorizedGroups = []models.AuthorizedGroup{}
	}

	successCount := 0
	failedCount := 0

	// 批量处理
	for _, targetUserID := range params.TargetUsers {
		// 限流
		h.rateLimiter.Wait(message.Chat.ID)

		// 获取目标用户信息
		chatMember, err := h.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: message.Chat.ID,
				UserID: targetUserID,
			},
		})

		var targetUsername, targetName string
		if err == nil {
			targetUsername, targetName = GetUserInfo(chatMember.User)
		} else {
			targetName = fmt.Sprintf("User_%d", targetUserID)
		}

		// 更新数据库
		err = h.muteService.UnmuteUser(targetUserID, params.Reason, message.From.ID)
		if err != nil {
			logrus.Errorf("Failed to update unmute record: %v", err)
			failedCount++
			continue
		}

		// 在所有授权群组中解除禁言
		for _, group := range authorizedGroups {
			h.rateLimiter.Wait(group.GroupID)

			restrictConfig := tgbotapi.RestrictChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: targetUserID,
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

			_, err = h.bot.Request(restrictConfig)
			if err != nil {
				logrus.Errorf("Failed to unmute user in group %d: %v", group.GroupID, err)
			}
		}

		// 记录日志
		h.logService.LogOperation(models.OpTypeUnmute, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			params.Reason, nil, true, "")

		// 发送通知
		h.notificationService.SendUnmuteNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, params.Reason, operatorName, message.From.ID)

		successCount++
	}

	// 发送操作结果反馈
	if failedCount == 0 {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 解除禁言操作成功")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 解除禁言操作完成，部分失败")
	}
}

// handleConfig 处理配置命令（仅作者）
func (h *Handler) handleConfig(message *tgbotapi.Message) {
	// 只允许作者在私聊中使用
	if message.From.ID != h.cfg.Telegram.AuthorID {
		return // 不回复非作者用户
	}

	if message.Chat.IsGroup() || message.Chat.IsSuperGroup() {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 此命令只能在私聊中使用")
		return
	}

	// 显示配置菜单
	h.showConfigMenu(message.Chat.ID)
}

// showConfigMenu 显示配置菜单
func (h *Handler) showConfigMenu(chatID int64) {
	text := "⚙️ *系统配置面板*\n\n请选择要执行的操作："

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ 增加授权群组", "config:add_group"),
			tgbotapi.NewInlineKeyboardButtonData("➖ 删除授权群组", "config:del_group"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👤 增加全局管理员", "config:add_admin"),
			tgbotapi.NewInlineKeyboardButtonData("🗑 删除全局管理员", "config:del_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 授权群组列表", "config:list_groups"),
			tgbotapi.NewInlineKeyboardButtonData("📋 管理员列表", "config:list_admins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 设置通知频道", "config:set_channel"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 更新管理员权限", "config:sync_admins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔒 关闭群管权限", "config:disable_admins"),
			tgbotapi.NewInlineKeyboardButtonData("🔓 开启群管权限", "config:enable_admins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ 关闭菜单", "config:close"),
		),
	)

	h.notificationService.SendMessageWithButtons(chatID, text, keyboard)
}

// sendReply 发送回复消息
func (h *Handler) sendReply(chatID int64, replyToMessageID int, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMessageID

	_, err := h.bot.Send(msg)
	if err != nil {
		logrus.Errorf("Failed to send reply: %v", err)
	}
}

// CheckNewMember 检查新成员
func (h *Handler) CheckNewMember(message *tgbotapi.Message) {
	if message.NewChatMembers == nil || len(message.NewChatMembers) == 0 {
		return
	}

	for _, newMember := range message.NewChatMembers {
		// 检查是否被拉黑
		banned, banRecord, err := h.banService.IsUserBanned(newMember.ID)
		if err != nil {
			logrus.Errorf("Failed to check ban status: %v", err)
			continue
		}

		if banned && banRecord != nil {
			// 立即踢出
			kickConfig := tgbotapi.KickChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: message.Chat.ID,
					UserID: newMember.ID,
				},
			}

			_, err = h.bot.Request(kickConfig)
			if err != nil {
				logrus.Errorf("Failed to kick banned user: %v", err)
			} else {
				_, fullName := GetUserInfo(&newMember)
				groupName := GetChatTitle(message.Chat)
				h.notificationService.SendTextMessage(message.Chat.ID,
					fmt.Sprintf("🚫 用户 %s 已被拉黑，自动踢出", utils.FormatUserMention(newMember.ID, fullName)))
				logrus.WithFields(logrus.Fields{
					"用户ID": newMember.ID,
					"群组":   groupName,
				}).Info("🚫 已拉黑用户尝试加入，已自动踢出")
			}
		}
	}
}

// CheckUnauthorizedGroup 检查未授权群组
func (h *Handler) CheckUnauthorizedGroup(message *tgbotapi.Message) {
	// 如果是群组消息
	if !message.Chat.IsGroup() && !message.Chat.IsSuperGroup() {
		return
	}

	// 检查群组是否已授权
	isAuthorized := h.permissionChecker.IsGroupAuthorized(message.Chat.ID)
	if !isAuthorized {
		// 检查是否已经通知过这个群组
		if h.notifiedUnauthorized[message.Chat.ID] {
			return // 已经通知过，不重复通知
		}

		// 标记为已通知
		h.notifiedUnauthorized[message.Chat.ID] = true

		// 发送通知 - 使用普通文本避免转义问题
		groupName := GetChatTitle(message.Chat)
		groupUsername := GetChatUsername(message.Chat)

		var text string
		if groupUsername != "" {
			// 公开群组，显示为超链接
			text = fmt.Sprintf("⚠️ *检测到未授权群组*\n\n*群组*：[%s](https://t.me/%s)\n*ID*：`%d`\n\n机器人将退出该群组",
				utils.EscapeMarkdown(groupName), groupUsername, message.Chat.ID)
		} else {
			// 私密群组
			text = fmt.Sprintf("⚠️ *检测到未授权群组*\n\n*群组*：%s\n*ID*：`%d`\n\n机器人将退出该群组",
				utils.EscapeMarkdown(groupName), message.Chat.ID)
		}

		h.notificationService.SendTextMessage(h.cfg.Telegram.AuthorID, text)

		// 退出群组
		leaveConfig := tgbotapi.LeaveChatConfig{
			ChatID: message.Chat.ID,
		}
		_, err := h.bot.Request(leaveConfig)
		if err != nil {
			logrus.Errorf("Failed to leave unauthorized group: %v", err)
		} else {
			// 退出成功后，从已通知列表中移除（因为已经退出了）
			delete(h.notifiedUnauthorized, message.Chat.ID)
		}

		logrus.WithFields(logrus.Fields{
			"群组名称": groupName,
			"群组ID": message.Chat.ID,
		}).Info("⚠️ 检测到未授权群组，已自动退出")
	}
}
