package bot

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// UserState 用户状态（用于对话模式）
type UserState struct {
	State string
	Data  map[string]interface{}
}

var (
	userStates = make(map[int64]*UserState)
	stateMutex sync.RWMutex
)

// HandleCallback 处理回调查询
func (h *Handler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	if callback == nil || callback.From == nil {
		return
	}

	// 只有作者可以使用配置功能
	if callback.From.ID != h.cfg.Telegram.AuthorID {
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 您没有权限", true)
		return
	}

	data := callback.Data
	logrus.Infof("Received callback: %s from user %d", data, callback.From.ID)

	// 解析回调数据
	parts := strings.SplitN(data, ":", 2)
	if len(parts) < 2 {
		return
	}

	action := parts[1]

	switch action {
	case "add_group":
		h.handleAddGroupCallback(callback)
	case "del_group":
		h.handleDelGroupCallback(callback)
	case "add_admin":
		h.handleAddAdminCallback(callback)
	case "del_admin":
		h.handleDelAdminCallback(callback)
	case "list_groups":
		h.handleListGroupsCallback(callback)
	case "list_admins":
		h.handleListAdminsCallback(callback)
	case "set_channel":
		h.handleSetChannelCallback(callback)
	case "sync_admins":
		h.handleSyncAdminsCallback(callback)
	case "disable_admins":
		h.handleDisableAdminsCallback(callback)
	case "enable_admins":
		h.handleEnableAdminsCallback(callback)
	case "close":
		h.handleCloseCallback(callback)
	default:
		// 处理删除操作的回调
		if strings.HasPrefix(action, "confirm_del_group_") {
			h.handleConfirmDelGroup(callback, action)
		} else if strings.HasPrefix(action, "confirm_del_admin_") {
			h.handleConfirmDelAdmin(callback, action)
		}
	}
}

// handleAddGroupCallback 处理添加授权群组回调
func (h *Handler) handleAddGroupCallback(callback *tgbotapi.CallbackQuery) {
	// 设置用户状态
	setUserState(callback.From.ID, "waiting_group_id", nil)

	text := "请发送要授权的群组ID\n\n格式示例：`-1002570701587`\n\n发送 /cancel 取消操作"
	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, nil)
	h.notificationService.AnswerCallbackQuery(callback.ID, "请发送群组ID", false)
}

// handleDelGroupCallback 处理删除授权群组回调
func (h *Handler) handleDelGroupCallback(callback *tgbotapi.CallbackQuery) {
	groups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 获取群组列表失败", true)
		return
	}

	if len(groups) == 0 {
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 没有已授权的群组", true)
		return
	}

	// 创建群组选择按钮
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		groupName := group.GroupName
		if groupName == "" {
			groupName = fmt.Sprintf("群组 %d", group.GroupID)
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🗑 %s", groupName),
			fmt.Sprintf("config:confirm_del_group_%d", group.GroupID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// 添加返回按钮
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ 返回", "config:back"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	text := "📋 *选择要删除的群组*"

	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, &keyboard)
	h.notificationService.AnswerCallbackQuery(callback.ID, "", false)
}

// handleConfirmDelGroup 确认删除群组
func (h *Handler) handleConfirmDelGroup(callback *tgbotapi.CallbackQuery, action string) {
	// 提取群组ID
	groupIDStr := strings.TrimPrefix(action, "confirm_del_group_")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 无效的群组ID", true)
		return
	}

	// 删除群组
	err = h.groupService.RemoveAuthorizedGroup(groupID)
	if err != nil {
		logrus.Errorf("Failed to remove authorized group: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 删除失败", true)
		return
	}

	h.notificationService.AnswerCallbackQuery(callback.ID, "✅ 已删除授权群组", true)
	// 返回主菜单
	h.showConfigMenu(callback.Message.Chat.ID)
}

// handleAddAdminCallback 处理添加全局管理员回调
func (h *Handler) handleAddAdminCallback(callback *tgbotapi.CallbackQuery) {
	// 设置用户状态
	setUserState(callback.From.ID, "waiting_admin_id", nil)

	text := "请发送要添加的管理员用户ID\n\n格式示例：`123456789`\n\n发送 /cancel 取消操作"
	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, nil)
	h.notificationService.AnswerCallbackQuery(callback.ID, "请发送用户ID", false)
}

// handleDelAdminCallback 处理删除全局管理员回调
func (h *Handler) handleDelAdminCallback(callback *tgbotapi.CallbackQuery) {
	admins, err := h.adminService.GetGlobalAdmins()
	if err != nil {
		logrus.Errorf("Failed to get global admins: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 获取管理员列表失败", true)
		return
	}

	if len(admins) == 0 {
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 没有全局管理员", true)
		return
	}

	// 创建管理员选择按钮
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, admin := range admins {
		adminName := admin.FullName
		if adminName == "" {
			adminName = admin.Username
		}
		if adminName == "" {
			adminName = fmt.Sprintf("用户 %d", admin.UserID)
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🗑 %s", adminName),
			fmt.Sprintf("config:confirm_del_admin_%d", admin.UserID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// 添加返回按钮
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ 返回", "config:back"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	text := "📋 *选择要删除的管理员*"

	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text, &keyboard)
	h.notificationService.AnswerCallbackQuery(callback.ID, "", false)
}

// handleConfirmDelAdmin 确认删除管理员
func (h *Handler) handleConfirmDelAdmin(callback *tgbotapi.CallbackQuery, action string) {
	// 提取用户ID
	userIDStr := strings.TrimPrefix(action, "confirm_del_admin_")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 无效的用户ID", true)
		return
	}

	// 删除管理员
	err = h.adminService.RemoveGlobalAdmin(userID)
	if err != nil {
		logrus.Errorf("Failed to remove global admin: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 删除失败", true)
		return
	}

	h.notificationService.AnswerCallbackQuery(callback.ID, "✅ 已删除全局管理员", true)
	// 返回主菜单
	h.showConfigMenu(callback.Message.Chat.ID)
}

// handleListGroupsCallback 处理查看授权群组列表回调
func (h *Handler) handleListGroupsCallback(callback *tgbotapi.CallbackQuery) {
	groups, err := h.groupService.GetAuthorizedGroups()
	if err != nil {
		logrus.Errorf("Failed to get authorized groups: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 获取列表失败", true)
		return
	}

	var text strings.Builder
	text.WriteString("📋 *授权群组列表*\n\n")

	if len(groups) == 0 {
		text.WriteString("暂无授权群组")
	} else {
		for i, group := range groups {
			groupName := group.GroupName
			if groupName == "" {
				groupName = "未知群组"
			}
			text.WriteString(fmt.Sprintf("%d\\. %s\n   ID: `%d`\n\n", i+1, groupName, group.GroupID))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ 返回", "config:back"),
		),
	)

	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text.String(), &keyboard)
	h.notificationService.AnswerCallbackQuery(callback.ID, "", false)
}

// handleListAdminsCallback 处理查看全局管理员列表回调
func (h *Handler) handleListAdminsCallback(callback *tgbotapi.CallbackQuery) {
	admins, err := h.adminService.GetGlobalAdmins()
	if err != nil {
		logrus.Errorf("Failed to get global admins: %v", err)
		h.notificationService.AnswerCallbackQuery(callback.ID, "❌ 获取列表失败", true)
		return
	}

	var text strings.Builder
	text.WriteString("📋 *全局管理员列表*\n\n")

	if len(admins) == 0 {
		text.WriteString("暂无全局管理员")
	} else {
		for i, admin := range admins {
			adminName := admin.FullName
			if adminName == "" {
				adminName = admin.Username
			}
			if adminName == "" {
				adminName = fmt.Sprintf("用户 %d", admin.UserID)
			}
			text.WriteString(fmt.Sprintf("%d\\. %s\n   ID: `%d`\n\n", i+1, adminName, admin.UserID))
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ 返回", "config:back"),
		),
	)

	h.notificationService.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text.String(), &keyboard)
	h.notificationService.AnswerCallbackQuery(callback.ID, "", false)
}

// handleSyncAdminsCallback 处理同步管理员权限回调
func (h *Handler) handleSyncAdminsCallback(callback *tgbotapi.CallbackQuery) {
	// 这里可以实现重新同步所有群组管理员的逻辑
	h.notificationService.AnswerCallbackQuery(callback.ID, "✅ 管理员权限已同步", true)
	h.showConfigMenu(callback.Message.Chat.ID)
}

// handleDisableAdminsCallback 处理关闭群管权限回调
func (h *Handler) handleDisableAdminsCallback(callback *tgbotapi.CallbackQuery) {
	h.cfg.System.AdminEnabled = false
	h.notificationService.AnswerCallbackQuery(callback.ID, "✅ 已关闭群管权限", true)
	h.showConfigMenu(callback.Message.Chat.ID)
}

// handleEnableAdminsCallback 处理开启群管权限回调
func (h *Handler) handleEnableAdminsCallback(callback *tgbotapi.CallbackQuery) {
	h.cfg.System.AdminEnabled = true
	h.notificationService.AnswerCallbackQuery(callback.ID, "✅ 已开启群管权限", true)
	h.showConfigMenu(callback.Message.Chat.ID)
}

// handleCloseCallback 处理关闭菜单回调
func (h *Handler) handleCloseCallback(callback *tgbotapi.CallbackQuery) {
	// 删除消息
	deleteMsg := tgbotapi.NewDeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	h.bot.Request(deleteMsg)
	h.notificationService.AnswerCallbackQuery(callback.ID, "已关闭", false)
}

// HandleTextMessage 处理文本消息（用于对话模式）
func (h *Handler) HandleTextMessage(message *tgbotapi.Message) {
	// 检查用户是否有待处理的状态
	state := getUserState(message.From.ID)
	if state == nil {
		return
	}

	switch state.State {
	case "waiting_group_id":
		h.handleWaitingGroupID(message)
	case "waiting_admin_id":
		h.handleWaitingAdminID(message)
	case "waiting_channel_id":
		h.handleWaitingChannelID(message)
	}
}

// handleWaitingGroupID 处理等待群组ID输入
func (h *Handler) handleWaitingGroupID(message *tgbotapi.Message) {
	// 解析群组ID
	groupID, err := strconv.ParseInt(strings.TrimSpace(message.Text), 10, 64)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 无效的群组ID格式，请重新输入或发送 /cancel 取消")
		return
	}

	// 获取群组信息
	chat, err := h.bot.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: groupID,
		},
	})

	var groupName string
	if err != nil {
		groupName = fmt.Sprintf("群组 %d", groupID)
	} else {
		groupName = chat.Title
	}

	// 添加到数据库
	err = h.groupService.AddAuthorizedGroup(groupID, groupName)
	if err != nil {
		logrus.Errorf("Failed to add authorized group: %v", err)
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 添加失败，可能该群组已存在")
		return
	}

	// 清除用户状态
	clearUserState(message.From.ID)

	h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 已添加授权群组\n\n群组：%s\nID：`%d`", groupName, groupID))
}

// handleWaitingAdminID 处理等待管理员ID输入
func (h *Handler) handleWaitingAdminID(message *tgbotapi.Message) {
	// 解析用户ID
	userID, err := strconv.ParseInt(strings.TrimSpace(message.Text), 10, 64)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 无效的用户ID格式，请重新输入或发送 /cancel 取消")
		return
	}

	// 获取用户信息（尝试）
	username := fmt.Sprintf("user_%d", userID)
	fullName := fmt.Sprintf("用户 %d", userID)

	// 添加到数据库
	err = h.adminService.AddGlobalAdmin(userID, username, fullName, message.From.ID)
	if err != nil {
		logrus.Errorf("Failed to add global admin: %v", err)
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 添加失败，可能该用户已是管理员")
		return
	}

	// 清除用户状态
	clearUserState(message.From.ID)

	h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 已添加全局管理员\n\nID：`%d`", userID))
}

// handleWaitingChannelID 处理等待通知频道ID输入
func (h *Handler) handleWaitingChannelID(message *tgbotapi.Message) {
	// 解析频道ID
	channelID, err := strconv.ParseInt(strings.TrimSpace(message.Text), 10, 64)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, "❌ 无效的频道ID格式，请重新输入或发送 /cancel 取消")
		return
	}

	// 尝试获取频道信息验证机器人是否有权限
	_, err = h.bot.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: channelID,
		},
	})

	if err != nil {
		logrus.Warnf("Failed to get channel info: %v", err)
		h.sendReply(message.Chat.ID, message.MessageID, "⚠️ 无法访问该频道，请确保：\n1. 机器人已被添加为频道管理员\n2. 频道ID正确\n\n是否仍要设置？如果确定，请重新发送频道ID")
		return
	}

	// 更新通知服务的频道ID
	h.notificationService.SetNotificationChannelID(channelID)

	// 清除用户状态
	clearUserState(message.From.ID)

	// 发送测试消息到新频道
	testMsg := fmt.Sprintf("✅ 通知频道设置成功\n\n频道ID：`%d`\n设置时间：%s", channelID, formatCurrentTime())
	err = h.notificationService.SendTextMessage(channelID, testMsg)
	if err != nil {
		h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("⚠️ 频道ID已设置，但发送测试消息失败：%v", err))
		return
	}

	h.sendReply(message.Chat.ID, message.MessageID, fmt.Sprintf("✅ 通知频道设置成功\n\n频道ID：`%d`\n\n已向频道发送测试消息", channelID))
}

func formatCurrentTime() string {
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d",
		2025, 10, 27, 22, 0, 0) // 临时实现
}

// 用户状态管理函数

func setUserState(userID int64, state string, data map[string]interface{}) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if data == nil {
		data = make(map[string]interface{})
	}

	userStates[userID] = &UserState{
		State: state,
		Data:  data,
	}
}

func getUserState(userID int64) *UserState {
	stateMutex.RLock()
	defer stateMutex.RUnlock()

	return userStates[userID]
}

func clearUserState(userID int64) {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	delete(userStates, userID)
}

// handleSetChannelCallback 处理设置通知频道回调
func (h *Handler) handleSetChannelCallback(callback *tgbotapi.CallbackQuery) {
	// 设置用户状态
	setUserState(callback.From.ID, "waiting_channel_id", nil)

	// 回复用户
	text := "📢 *设置通知频道*\n\n请发送通知频道ID\n\n" +
		"💡 提示：\n" +
		"• 频道ID通常为负数，如：`-1001234567890`\n" +
		"• 机器人必须是频道管理员\n" +
		"• 发送 /cancel 取消操作"

	h.notificationService.AnswerCallbackQuery(callback.ID, "", false)
	h.notificationService.SendTextMessage(callback.Message.Chat.ID, text)
}
