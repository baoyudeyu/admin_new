package bot

import (
	"admin-bot/internal/cache"
	"admin-bot/internal/config"
	"admin-bot/internal/models"
	"admin-bot/internal/service"
	"admin-bot/internal/utils"
	"fmt"
	"time"

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
	userCacheService     *service.UserCacheService
	rateLimiter          *utils.RateLimiter
	notifiedUnauthorized map[int64]bool      // 记录已通知的未授权群组
	notifiedMutex        *utils.SafeMap      // 并发安全的通知记录 map
}

// NewHandler 创建处理器
func NewHandler(bot *tgbotapi.BotAPI, cfg *config.Config,
	permissionChecker *PermissionChecker,
	banService *service.BanService,
	muteService *service.MuteService,
	groupService *service.GroupService,
	adminService *service.AdminService,
	logService *service.LogService,
	notificationService *service.NotificationService,
	userCacheService *service.UserCacheService) *Handler {

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
		userCacheService:     userCacheService,
		rateLimiter:          utils.NewRateLimiter(cfg.System.RateLimitPerGroup),
		notifiedUnauthorized: make(map[int64]bool),
		notifiedMutex:        utils.NewSafeMap(30 * time.Minute), // 30分钟后自动清理通知记录
	}
}

// CacheUserInfo 缓存用户信息
func (h *Handler) CacheUserInfo(user *tgbotapi.User) {
	if user.UserName != "" {
		h.userCacheService.SaveOrUpdateUser(user.ID, user.UserName, user.FirstName, user.LastName)
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
	case "cancel":
		h.handleCancel(message)
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
		"/unjy \\[理由\\] - 解除禁言\n" +
		"/cancel - 取消当前操作\n\n" +
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

// handleCancel 处理 /cancel 命令（取消当前对话操作）
func (h *Handler) handleCancel(message *tgbotapi.Message) {
	// 只在私聊中有效
	if !message.Chat.IsPrivate() {
		return
	}

	// 只允许作者取消对话
	if !h.cfg.Telegram.IsAuthor(message.From.ID) {
		return
	}

	// 检查是否有待处理的状态
	state := getUserState(message.From.ID)

	// 无论是否有状态，都尝试清除（确保彻底清理）
	clearUserState(message.From.ID)

	if state == nil {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 已清除所有对话状态")
	} else {
		h.sendReply(message.Chat.ID, message.MessageID, "✅ 已取消当前操作并清除所有对话状态")
		logrus.WithFields(logrus.Fields{
			"用户ID": message.From.ID,
			"状态":   state.State,
		}).Info("🚫 用户取消了对话操作")
	}
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
	params, err := ParseCommand(message, h.bot, h.userCacheService)
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

	// 处理所有目标用户
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

		// 在所有授权群组中执行踢出
		groupSuccessCount := 0
		for _, group := range authorizedGroups {
			h.rateLimiter.Wait(group.GroupID)

			// 执行踢出
			kickConfig := tgbotapi.KickChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: group.GroupID,
					UserID: targetUserID,
				},
			}

			_, err = h.bot.Request(kickConfig)
			if err != nil {
				logrus.Errorf("Failed to kick user in group %d: %v", group.GroupID, err)
			} else {
				// 解除封禁（允许用户再次加入）
				unbanConfig := tgbotapi.UnbanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{
						ChatID: group.GroupID,
						UserID: targetUserID,
					},
				}
				h.bot.Request(unbanConfig)
				groupSuccessCount++
			}
		}

		// 如果所有群组都失败，则标记为失败
		if groupSuccessCount == 0 {
			failedCount++
			h.notificationService.SendErrorNotification(groupName, "踢出", targetName,
				targetUserID, "所有群组踢出失败", operatorName)
			continue
		}

		// 记录日志
		h.logService.LogOperation(models.OpTypeKick, targetUserID, targetUsername,
			message.Chat.ID, groupName, message.From.ID, operatorName,
			"", nil, true, "")

		// 发送通知
		h.notificationService.SendKickNotification(message.Chat.ID, groupName, groupUsername,
			targetName, targetUserID, operatorName, message.From.ID)

		logrus.WithFields(logrus.Fields{
			"用户ID": targetUserID,
			"成功数量": groupSuccessCount,
			"总群组数": len(authorizedGroups),
		}).Info("✅ 用户已在多个群组中被踢出")

		successCount++
	}

	// 发送操作结果反馈
	if params.IsBatch {
		// 批量操作显示详细结果
		if failedCount == 0 {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 踢出操作成功（%d/%d）", successCount, len(params.TargetUsers)))
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("⚠️ 踢出操作完成，成功 %d，失败 %d", successCount, failedCount))
		}
	} else {
		// 单用户操作简单反馈
		if successCount > 0 {
			h.sendReply(message.Chat.ID, message.MessageID, "✅ 踢出操作成功")
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, "❌ 踢出操作失败")
		}
	}
}

// handleBan 处理拉黑命令（异步优化版本）
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
	params, err := ParseCommand(message, h.bot, h.userCacheService)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 立即发送"处理中"反馈，提升响应速度
	processingMsg := h.sendReplyAndGetMessage(message.Chat.ID, message.MessageID, "⏳ 正在处理拉黑操作...")

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

	// 异步处理所有用户
	go func() {
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

			// 并发执行多群组拉黑操作
			var banSuccess int
			var banFailed int
			tasks := make([]func(), 0, len(authorizedGroups))

			for _, group := range authorizedGroups {
				grp := group // 捕获变量
				tasks = append(tasks, func() {
					h.rateLimiter.Wait(grp.GroupID)

					kickConfig := tgbotapi.KickChatMemberConfig{
						ChatMemberConfig: tgbotapi.ChatMemberConfig{
							ChatID: grp.GroupID,
							UserID: targetUserID,
						},
					}

					if params.Duration > 0 {
						kickConfig.UntilDate = int64(params.Duration)
					}

					_, err := h.bot.Request(kickConfig)
					if err != nil {
						logrus.Errorf("Failed to ban user in group %d: %v", grp.GroupID, err)
						banFailed++
					} else {
						banSuccess++
					}
				})
			}

			// 并发执行所有群组的拉黑操作
			utils.ParallelExecuteWithLimit(tasks, 5)

			// 只有至少一个群组成功才算成功
			if banSuccess > 0 {
				successCount++

				// 异步保存到数据库并记录日志（不阻塞核心功能）
				go func(uid int64, uname, fname string) {
					// 保存到数据库
					err := h.banService.BanUser(uid, uname, fname,
						message.Chat.ID, groupName, message.From.ID, operatorName,
						params.Reason, params.Duration)
					if err != nil {
						// 数据库保存失败不影响用户反馈，但记录详细错误
						logrus.WithFields(logrus.Fields{
							"用户ID": uid,
							"用户名":  uname,
							"群组":   groupName,
							"错误":   err.Error(),
						}).Error("❌ 数据库保存失败（Telegram操作已成功）")
					}

					// 记录操作日志
					durationPtr := &params.Duration
					h.logService.LogOperation(models.OpTypeBan, uid, uname,
						message.Chat.ID, groupName, message.From.ID, operatorName,
						params.Reason, durationPtr, true, "")
				}(targetUserID, targetUsername, targetName)

				// 发送通知（已经是异步的）
				h.notificationService.SendBanNotification(message.Chat.ID, groupName, groupUsername,
					targetName, targetUserID, params.Duration, params.Reason, operatorName, message.From.ID)

				logrus.WithFields(logrus.Fields{
					"用户ID":  targetUserID,
					"用户名":   targetName,
					"成功群组数": banSuccess,
					"失败群组数": banFailed,
					"总群组数":  len(authorizedGroups),
				}).Info("✅ 拉黑操作完成")
			} else {
				failedCount++
			}
		}

		// 更新消息状态
		var resultText string
		if params.IsBatch {
			// 批量操作显示详细结果
			if failedCount == 0 {
				resultText = fmt.Sprintf("✅ 拉黑操作成功（%d/%d）", successCount, len(params.TargetUsers))
			} else {
				resultText = fmt.Sprintf("⚠️ 拉黑操作完成，成功 %d，失败 %d", successCount, failedCount)
			}
		} else {
			// 单用户操作简单反馈
			if successCount > 0 {
				resultText = "✅ 拉黑操作成功"
			} else {
				resultText = "❌ 拉黑操作失败"
			}
		}

		// 更新处理中的消息
		if processingMsg != nil {
			h.editMessage(message.Chat.ID, processingMsg.MessageID, resultText)
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, resultText)
		}
	}()
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
	params, err := ParseCommand(message, h.bot, h.userCacheService)
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

	// 处理所有目标用户
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
	if params.IsBatch {
		// 批量操作显示详细结果
		if failedCount == 0 {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 解除拉黑操作成功（%d/%d）", successCount, len(params.TargetUsers)))
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("⚠️ 解除拉黑操作完成，成功 %d，失败 %d", successCount, failedCount))
		}
	} else {
		// 单用户操作简单反馈
		if successCount > 0 {
			h.sendReply(message.Chat.ID, message.MessageID, "✅ 解除拉黑操作成功")
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, "❌ 解除拉黑操作失败")
		}
	}
}

// handleMute 处理禁言命令（异步优化版本）
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
	params, err := ParseCommand(message, h.bot, h.userCacheService)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// 立即发送"处理中"反馈，提升响应速度
	processingMsg := h.sendReplyAndGetMessage(message.Chat.ID, message.MessageID, "⏳ 正在处理禁言操作...")

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

	// 异步处理所有用户
	go func() {
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

			// 并发执行多群组禁言操作
			var muteSuccess int
			var muteFailed int
			tasks := make([]func(), 0, len(authorizedGroups))

			for _, group := range authorizedGroups {
				grp := group // 捕获变量
				tasks = append(tasks, func() {
					h.rateLimiter.Wait(grp.GroupID)

					restrictConfig := tgbotapi.RestrictChatMemberConfig{
						ChatMemberConfig: tgbotapi.ChatMemberConfig{
							ChatID: grp.GroupID,
							UserID: targetUserID,
						},
						Permissions: &tgbotapi.ChatPermissions{
							CanSendMessages: false,
						},
					}

					if params.Duration > 0 {
						restrictConfig.UntilDate = int64(params.Duration)
					}

					_, err := h.bot.Request(restrictConfig)
					if err != nil {
						logrus.Errorf("Failed to mute user in group %d: %v", grp.GroupID, err)
						muteFailed++
					} else {
						muteSuccess++
					}
				})
			}

			// 并发执行所有群组的禁言操作
			utils.ParallelExecuteWithLimit(tasks, 5)

			// 只有至少一个群组成功才算成功
			if muteSuccess > 0 {
				successCount++

				// 异步保存到数据库并记录日志（不阻塞核心功能）
				go func(uid int64, uname, fname string) {
					// 保存到数据库
					err := h.muteService.MuteUser(uid, uname, fname,
						message.Chat.ID, groupName, message.From.ID, operatorName,
						params.Reason, params.Duration)
					if err != nil {
						// 数据库保存失败不影响用户反馈，但记录详细错误
						logrus.WithFields(logrus.Fields{
							"用户ID": uid,
							"用户名":  uname,
							"群组":   groupName,
							"错误":   err.Error(),
						}).Error("❌ 数据库保存失败（Telegram操作已成功）")
					}

					// 记录操作日志
					durationPtr := &params.Duration
					h.logService.LogOperation(models.OpTypeMute, uid, uname,
						message.Chat.ID, groupName, message.From.ID, operatorName,
						params.Reason, durationPtr, true, "")
				}(targetUserID, targetUsername, targetName)

				// 发送通知（已经是异步的）
				h.notificationService.SendMuteNotification(message.Chat.ID, groupName, groupUsername,
					targetName, targetUserID, params.Duration, params.Reason, operatorName, message.From.ID)

				logrus.WithFields(logrus.Fields{
					"用户ID":  targetUserID,
					"用户名":   targetName,
					"成功群组数": muteSuccess,
					"失败群组数": muteFailed,
					"总群组数":  len(authorizedGroups),
				}).Info("✅ 禁言操作完成")
			} else {
				failedCount++
			}
		}

		// 更新消息状态
		var resultText string
		if params.IsBatch {
			// 批量操作显示详细结果
			if failedCount == 0 {
				resultText = fmt.Sprintf("✅ 禁言操作成功（%d/%d）", successCount, len(params.TargetUsers))
			} else {
				resultText = fmt.Sprintf("⚠️ 禁言操作完成，成功 %d，失败 %d", successCount, failedCount)
			}
		} else {
			// 单用户操作简单反馈
			if successCount > 0 {
				resultText = "✅ 禁言操作成功"
			} else {
				resultText = "❌ 禁言操作失败"
			}
		}

		// 更新处理中的消息
		if processingMsg != nil {
			h.editMessage(message.Chat.ID, processingMsg.MessageID, resultText)
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, resultText)
		}
	}()
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
	params, err := ParseCommand(message, h.bot, h.userCacheService)
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
	if params.IsBatch {
		// 批量操作显示详细结果
		if failedCount == 0 {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 解除禁言操作成功（%d/%d）", successCount, len(params.TargetUsers)))
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("⚠️ 解除禁言操作完成，成功 %d，失败 %d", successCount, failedCount))
		}
	} else {
		// 单用户操作简单反馈
		if successCount > 0 {
			h.sendReply(message.Chat.ID, message.MessageID, "✅ 解除禁言操作成功")
		} else {
			h.sendReply(message.Chat.ID, message.MessageID, "❌ 解除禁言操作失败")
		}
	}
}

// handleConfig 处理配置命令（仅作者）
func (h *Handler) handleConfig(message *tgbotapi.Message) {
	// 只允许作者在私聊中使用
	if !h.cfg.Telegram.IsAuthor(message.From.ID) {
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
	msg.DisableWebPagePreview = true // 禁用链接预览，避免占用空间

	_, err := h.bot.Send(msg)
	if err != nil {
		logrus.Errorf("Failed to send reply: %v", err)
	}
}

// sendReplyAndGetMessage 发送回复消息并返回消息对象
func (h *Handler) sendReplyAndGetMessage(chatID int64, replyToMessageID int, text string) *tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToMessageID
	msg.DisableWebPagePreview = true // 禁用链接预览，避免占用空间

	sentMsg, err := h.bot.Send(msg)
	if err != nil {
		logrus.Errorf("Failed to send reply: %v", err)
		return nil
	}
	return &sentMsg
}

// editMessage 编辑消息内容
func (h *Handler) editMessage(chatID int64, messageID int, text string) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.DisableWebPagePreview = true

	_, err := h.bot.Send(msg)
	if err != nil {
		logrus.Errorf("Failed to edit message: %v", err)
	}
}

// CheckBotAddedToGroup 检查机器人是否被添加到群组
func (h *Handler) CheckBotAddedToGroup(message *tgbotapi.Message, botID int64) {
	if len(message.NewChatMembers) == 0 {
		return
	}

	// 检查新成员中是否包含机器人自己
	var isBotAdded bool
	for _, member := range message.NewChatMembers {
		if member.ID == botID {
			isBotAdded = true
			break
		}
	}

	if !isBotAdded {
		return
	}

	// 机器人被添加到群组，检查是否为授权群组
	groupID := message.Chat.ID
	groupName := message.Chat.Title
	groupUsername := message.Chat.UserName

	// 检查群组是否已授权
	isAuthorized, err := h.groupService.IsAuthorized(groupID)
	if err != nil {
		logrus.Errorf("Failed to check group authorization: %v", err)
		return
	}

	if isAuthorized {
		// 已授权群组，更新群组信息
		err = h.groupService.UpdateGroupInfo(groupID, groupName, groupUsername)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"群组ID": groupID,
				"错误":   err.Error(),
			}).Error("❌ 更新群组信息失败")
		} else {
			logrus.WithFields(logrus.Fields{
				"群组ID": groupID,
				"群组名":  groupName,
				"用户名":  groupUsername,
			}).Info("✅ 机器人加入授权群组，已更新群组信息")
		}
	} else {
		// 未授权群组，记录日志（后续会由 CheckUnauthorizedGroup 处理退出）
		logrus.WithFields(logrus.Fields{
			"群组ID": groupID,
			"群组名":  groupName,
		}).Warn("⚠️ 机器人被添加到未授权群组")
	}
}

// CheckNewMember 检查新成员
func (h *Handler) CheckNewMember(message *tgbotapi.Message) {
	if len(message.NewChatMembers) == 0 {
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

// CheckUnauthorizedGroup 检查未授权群组（优化版：特殊保护 + 并发安全）
func (h *Handler) CheckUnauthorizedGroup(message *tgbotapi.Message) {
	// 如果是群组消息
	if !message.Chat.IsGroup() && !message.Chat.IsSuperGroup() {
		return
	}

	chatID := message.Chat.ID

	// 1. 特殊保护：通知频道永不退出
	authCache := cache.GetAuthCache()
	if authCache.IsNotificationChannel(chatID) {
		logrus.WithField("频道ID", chatID).Debug("✅ 通知频道，跳过未授权检查")
		return
	}

	// 2. 特殊保护：作者所在群组发出警告但不自动退出
	if message.From != nil && h.cfg.Telegram.IsAuthor(message.From.ID) {
		// 作者在群里，即使未授权也不退出，只记录日志
		isAuthorized := h.permissionChecker.IsGroupAuthorized(chatID)
		if !isAuthorized {
			// 只在第一次检测到时发送警告
			if !h.notifiedMutex.Has(chatID) {
				h.notifiedMutex.Set(chatID)
				
				groupName := GetChatTitle(message.Chat)
				text := fmt.Sprintf("⚠️ *警告：作者在未授权群组中*\n\n*群组*：%s\n*ID*：`%d`\n\n💡 由于作者在群内，机器人不会自动退出。如需管理此群组，请使用 /config 命令添加授权。",
					utils.EscapeMarkdown(groupName), chatID)
				
				// 只通知作者本人
				h.notificationService.SendTextMessage(message.From.ID, text)
				
				logrus.WithFields(logrus.Fields{
					"群组名称": groupName,
					"群组ID": chatID,
				}).Warn("⚠️ 检测到作者在未授权群组中，已发送警告但保留")
			}
			return
		}
	}

	// 3. 检查群组是否已授权
	isAuthorized := h.permissionChecker.IsGroupAuthorized(chatID)
	if !isAuthorized {
		// 4. 并发安全检查：是否已经通知过这个群组
		if h.notifiedMutex.Has(chatID) {
			return // 已经通知过，不重复通知
		}

		// 5. 标记为已通知（并发安全）
		h.notifiedMutex.Set(chatID)

		// 6. 发送通知
		groupName := GetChatTitle(message.Chat)
		groupUsername := GetChatUsername(message.Chat)

		var text string
		if groupUsername != "" {
			// 公开群组，显示为超链接
			text = fmt.Sprintf("⚠️ *检测到未授权群组*\n\n*群组*：[%s](https://t.me/%s)\n*ID*：`%d`\n\n机器人将退出该群组",
				utils.EscapeMarkdown(groupName), groupUsername, chatID)
		} else {
			// 私密群组
			text = fmt.Sprintf("⚠️ *检测到未授权群组*\n\n*群组*：%s\n*ID*：`%d`\n\n机器人将退出该群组",
				utils.EscapeMarkdown(groupName), chatID)
		}

		// 通知所有作者
		for _, authorID := range h.cfg.Telegram.AuthorIDs {
			h.notificationService.SendTextMessage(authorID, text)
		}

		// 7. 退出群组
		leaveConfig := tgbotapi.LeaveChatConfig{
			ChatID: chatID,
		}
		_, err := h.bot.Request(leaveConfig)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"群组ID": chatID,
				"错误":   err.Error(),
			}).Error("❌ 退出未授权群组失败")
		} else {
			// 退出成功后，从已通知列表中移除（因为已经退出了）
			h.notifiedMutex.Delete(chatID)
			
			logrus.WithFields(logrus.Fields{
				"群组名称": groupName,
				"群组ID": chatID,
			}).Info("⚠️ 检测到未授权群组，已自动退出")
		}
	}
}
