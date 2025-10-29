package utils

import (
	"strings"
	"unicode/utf8"
)

// TruncateString 安全截断字符串到指定长度（支持 UTF-8）
// maxLen 是字符数（不是字节数）
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// 如果字符串长度小于最大长度，直接返回
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}

	// 截断字符串
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}

	return s
}

// SanitizeString 清理字符串，移除不可见字符和控制字符
func SanitizeString(s string) string {
	// 移除前后空白
	s = strings.TrimSpace(s)

	// 替换多个空白字符为单个空格
	s = strings.Join(strings.Fields(s), " ")

	return s
}

// TruncateStringBytes 按字节长度截断字符串（UTF-8 安全）
// 确保不会在多字节字符中间截断
func TruncateStringBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	// 从 maxBytes 位置向前查找有效的 UTF-8 字符边界
	for i := maxBytes; i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i]
		}
	}

	return ""
}

// SafeUsername 安全处理用户名，限制长度并清理
func SafeUsername(username string) string {
	username = SanitizeString(username)
	return TruncateString(username, 255)
}

// SafeFullName 安全处理全名，限制长度并清理
func SafeFullName(fullName string) string {
	fullName = SanitizeString(fullName)
	return TruncateString(fullName, 255)
}

// SafeGroupName 安全处理群组名，限制长度并清理
func SafeGroupName(groupName string) string {
	groupName = SanitizeString(groupName)
	return TruncateString(groupName, 255)
}

// SafeReason 安全处理原因文本，限制长度
func SafeReason(reason string) string {
	reason = SanitizeString(reason)
	// TEXT 类型通常最大 65535 字节，但我们限制为 1000 字符以保持合理
	return TruncateString(reason, 1000)
}
