package utils

import (
	"fmt"
	"strings"
)

// FormatUserMention 格式化用户提及链接
func FormatUserMention(userID int64, fullName string) string {
	return fmt.Sprintf("[%s](tg://user?id=%d)", EscapeMarkdown(fullName), userID)
}

// FormatGroupName 格式化群组名称（公开群组显示为超链接，私密群显示为普通文本）
func FormatGroupName(groupName, groupUsername string) string {
	if groupUsername != "" {
		// 公开群组，显示为超链接
		return fmt.Sprintf("[%s](https://t.me/%s)", EscapeMarkdown(groupName), groupUsername)
	}
	// 私密群组，显示为普通文本
	return EscapeMarkdown(groupName)
}

// EscapeMarkdown 转义 Markdown 特殊字符
func EscapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// FormatBanNotification 格式化拉黑通知
func FormatBanNotification(groupName, groupUsername, userName string, userID int64, duration, reason, operatorName string, operatorID int64, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("🚫 *拉黑通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", FormatGroupName(groupName, groupUsername)))
	sb.WriteString(fmt.Sprintf("*用户*：%s\n", FormatUserMention(userID, userName)))
	sb.WriteString(fmt.Sprintf("*ID*：`%d`\n", userID))
	sb.WriteString(fmt.Sprintf("*时长*：%s\n", EscapeMarkdown(duration)))
	if reason != "" {
		sb.WriteString(fmt.Sprintf("*理由*：%s\n", EscapeMarkdown(reason)))
	}
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`\n", timestamp))
	sb.WriteString(fmt.Sprintf("*操作人*：%s", FormatUserMention(operatorID, operatorName)))
	return sb.String()
}

// FormatUnbanNotification 格式化解除拉黑通知
func FormatUnbanNotification(groupName, groupUsername, userName string, userID int64, reason, operatorName string, operatorID int64, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("🔓 *解除拉黑通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", FormatGroupName(groupName, groupUsername)))
	sb.WriteString(fmt.Sprintf("*用户*：%s\n", FormatUserMention(userID, userName)))
	sb.WriteString(fmt.Sprintf("*ID*：`%d`\n", userID))
	if reason != "" {
		sb.WriteString(fmt.Sprintf("*理由*：%s\n", EscapeMarkdown(reason)))
	}
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`\n", timestamp))
	sb.WriteString(fmt.Sprintf("*操作人*：%s", FormatUserMention(operatorID, operatorName)))
	return sb.String()
}

// FormatMuteNotification 格式化禁言通知
func FormatMuteNotification(groupName, groupUsername, userName string, userID int64, duration, reason, operatorName string, operatorID int64, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("🔇 *禁言通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", FormatGroupName(groupName, groupUsername)))
	sb.WriteString(fmt.Sprintf("*用户*：%s\n", FormatUserMention(userID, userName)))
	sb.WriteString(fmt.Sprintf("*ID*：`%d`\n", userID))
	sb.WriteString(fmt.Sprintf("*时长*：%s\n", EscapeMarkdown(duration)))
	if reason != "" {
		sb.WriteString(fmt.Sprintf("*理由*：%s\n", EscapeMarkdown(reason)))
	}
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`\n", timestamp))
	sb.WriteString(fmt.Sprintf("*操作人*：%s", FormatUserMention(operatorID, operatorName)))
	return sb.String()
}

// FormatUnmuteNotification 格式化解除禁言通知
func FormatUnmuteNotification(groupName, groupUsername, userName string, userID int64, reason, operatorName string, operatorID int64, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("🔈 *解除禁言通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", FormatGroupName(groupName, groupUsername)))
	sb.WriteString(fmt.Sprintf("*用户*：%s\n", FormatUserMention(userID, userName)))
	sb.WriteString(fmt.Sprintf("*ID*：`%d`\n", userID))
	if reason != "" {
		sb.WriteString(fmt.Sprintf("*理由*：%s\n", EscapeMarkdown(reason)))
	}
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`\n", timestamp))
	sb.WriteString(fmt.Sprintf("*操作人*：%s", FormatUserMention(operatorID, operatorName)))
	return sb.String()
}

// FormatKickNotification 格式化踢出通知
func FormatKickNotification(groupName, groupUsername, userName string, userID int64, operatorName string, operatorID int64, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("👢 *踢出通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", FormatGroupName(groupName, groupUsername)))
	sb.WriteString(fmt.Sprintf("*用户*：%s\n", FormatUserMention(userID, userName)))
	sb.WriteString(fmt.Sprintf("*ID*：`%d`\n", userID))
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`\n", timestamp))
	sb.WriteString(fmt.Sprintf("*操作人*：%s", FormatUserMention(operatorID, operatorName)))
	return sb.String()
}

// FormatErrorNotification 格式化错误通知
func FormatErrorNotification(groupName, operationType, userName string, userID int64, errorMsg, operatorName, timestamp string) string {
	var sb strings.Builder
	sb.WriteString("⚠️ *操作失败通知*\n\n")
	sb.WriteString(fmt.Sprintf("*群组*：%s\n", EscapeMarkdown(groupName)))
	sb.WriteString(fmt.Sprintf("*操作类型*：%s\n", EscapeMarkdown(operationType)))
	sb.WriteString(fmt.Sprintf("*目标用户*：%s \\(ID: %d\\)\n", EscapeMarkdown(userName), userID))
	sb.WriteString(fmt.Sprintf("*失败原因*：%s\n", EscapeMarkdown(errorMsg)))
	sb.WriteString(fmt.Sprintf("*操作人*：%s\n", EscapeMarkdown(operatorName)))
	sb.WriteString(fmt.Sprintf("*操作时间*：`%s`", timestamp))
	return sb.String()
}
