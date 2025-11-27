package service

import (
	"admin-bot/internal/utils"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// NotificationService 通知服务
type NotificationService struct {
	bot                   *tgbotapi.BotAPI
	notificationChannelID int64
	authorIDs             []int64
}

// NewNotificationService 创建通知服务
func NewNotificationService(bot *tgbotapi.BotAPI, channelID int64, authorIDs []int64) *NotificationService {
	return &NotificationService{
		bot:                   bot,
		notificationChannelID: channelID,
		authorIDs:             authorIDs,
	}
}

// SendBanNotification 发送拉黑通知
func (s *NotificationService) SendBanNotification(groupID int64, groupName, groupUsername, userName string,
	userID int64, duration int, reason, operatorName string, operatorID int64) error {

	durationStr := utils.FormatDuration(duration)
	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatBanNotification(groupName, groupUsername, userName, userID, durationStr, reason, operatorName, operatorID, timestamp)

	// 异步发送通知以提升响应速度
	go func() {
		s.sendNotificationWithCheck(message, "拉黑")
	}()

	return nil
}

// SendUnbanNotification 发送解除拉黑通知
func (s *NotificationService) SendUnbanNotification(groupID int64, groupName, groupUsername, userName string,
	userID int64, reason, operatorName string, operatorID int64) error {

	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatUnbanNotification(groupName, groupUsername, userName, userID, reason, operatorName, operatorID, timestamp)

	// 异步发送通知以提升响应速度
	go func() {
		s.sendNotificationWithCheck(message, "解除拉黑")
	}()

	return nil
}

// SendMuteNotification 发送禁言通知
func (s *NotificationService) SendMuteNotification(groupID int64, groupName, groupUsername, userName string,
	userID int64, duration int, reason, operatorName string, operatorID int64) error {

	durationStr := utils.FormatDuration(duration)
	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatMuteNotification(groupName, groupUsername, userName, userID, durationStr, reason, operatorName, operatorID, timestamp)

	// 异步发送通知以提升响应速度
	go func() {
		s.sendNotificationWithCheck(message, "禁言")
	}()

	return nil
}

// SendUnmuteNotification 发送解除禁言通知
func (s *NotificationService) SendUnmuteNotification(groupID int64, groupName, groupUsername, userName string,
	userID int64, reason, operatorName string, operatorID int64) error {

	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatUnmuteNotification(groupName, groupUsername, userName, userID, reason, operatorName, operatorID, timestamp)

	// 异步发送通知以提升响应速度
	go func() {
		s.sendNotificationWithCheck(message, "解除禁言")
	}()

	return nil
}

// SendKickNotification 发送踢出通知
func (s *NotificationService) SendKickNotification(groupID int64, groupName, groupUsername, userName string,
	userID int64, operatorName string, operatorID int64) error {

	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatKickNotification(groupName, groupUsername, userName, userID, operatorName, operatorID, timestamp)

	// 异步发送通知以提升响应速度
	go func() {
		s.sendNotificationWithCheck(message, "踢出")
	}()

	return nil
}

// SendErrorNotification 发送错误通知给作者
func (s *NotificationService) SendErrorNotification(groupName, operationType, userName string,
	userID int64, errorMsg, operatorName string) error {

	timestamp := utils.FormatTimestamp(time.Now())
	message := utils.FormatErrorNotification(groupName, operationType, userName, userID, errorMsg, operatorName, timestamp)

	// 发送给所有作者
	for _, authorID := range s.authorIDs {
		if err := s.sendMessage(authorID, message); err != nil {
			logrus.Errorf("Failed to send error notification to author %d: %v", authorID, err)
		}
	}
	return nil
}

// SendTextMessage 发送文本消息
func (s *NotificationService) SendTextMessage(chatID int64, text string) error {
	return s.sendMessage(chatID, text)
}

// sendMessage 发送消息的内部方法
func (s *NotificationService) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true // 禁用链接预览，避免占用空间

	_, err := s.bot.Send(msg)
	if err != nil {
		// 如果 Markdown 解析失败，尝试不使用 Markdown 重新发送
		logrus.Warnf("Failed to send message with Markdown: %v, retrying without parse mode", err)
		msg.ParseMode = ""
		_, err = s.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

// SendMessageWithButtons 发送带按钮的消息
func (s *NotificationService) SendMessageWithButtons(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true // 禁用链接预览，避免占用空间
	msg.ReplyMarkup = keyboard

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message with buttons: %w", err)
	}

	return nil
}

// EditMessage 编辑消息
func (s *NotificationService) EditMessage(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true // 禁用链接预览，避免占用空间
	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// AnswerCallbackQuery 回应回调查询
func (s *NotificationService) AnswerCallbackQuery(callbackQueryID string, text string, showAlert bool) error {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	callback.ShowAlert = showAlert

	_, err := s.bot.Request(callback)
	return err
}

// SetNotificationChannelID 设置通知频道ID
func (s *NotificationService) SetNotificationChannelID(channelID int64) {
	s.notificationChannelID = channelID
	logrus.WithFields(logrus.Fields{
		"频道ID": channelID,
	}).Info("📢 通知频道已更新")
}

// GetNotificationChannelID 获取通知频道ID
func (s *NotificationService) GetNotificationChannelID() int64 {
	return s.notificationChannelID
}

// sendNotificationWithCheck 发送通知并检查频道是否已配置
func (s *NotificationService) sendNotificationWithCheck(message, operationType string) {
	// 检查是否配置了通知频道
	if s.notificationChannelID == 0 {
		// 未配置通知频道，向所有作者发送提醒
		warningMsg := fmt.Sprintf("⚠️ *未配置通知频道*\n\n操作类型：%s\n\n"+
			"请使用 /config 命令配置通知频道以接收操作通知。\n\n"+
			"功能仍正常执行。", operationType)
		for _, authorID := range s.authorIDs {
			if err := s.sendMessage(authorID, warningMsg); err != nil {
				logrus.Errorf("Failed to send channel warning to author %d: %v", authorID, err)
			}
		}
		logrus.Warnf("通知频道未配置，已提醒作者")
		return
	}

	// 发送到通知频道
	if err := s.sendMessage(s.notificationChannelID, message); err != nil {
		logrus.Errorf("Failed to send notification to channel: %v", err)
		// 发送失败时也通知所有作者
		errorMsg := fmt.Sprintf("⚠️ *通知发送失败*\n\n操作类型：%s\n错误：%s\n\n"+
			"请检查：\n1. 机器人是否仍是频道管理员\n2. 频道ID是否正确", operationType, err.Error())
		for _, authorID := range s.authorIDs {
			s.sendMessage(authorID, errorMsg)
		}
	}
}
