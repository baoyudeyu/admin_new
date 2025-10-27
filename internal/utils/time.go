package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration 解析时间字符串，如 "10s", "5m", "2h", "1d"
func ParseDuration(duration string) (int, error) {
	if duration == "" {
		return 0, nil
	}

	duration = strings.TrimSpace(duration)
	if len(duration) < 2 {
		return 0, fmt.Errorf("invalid duration format: %s", duration)
	}

	// 提取数字和单位
	unit := duration[len(duration)-1:]
	valueStr := duration[:len(duration)-1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", valueStr)
	}

	if value < 0 {
		return 0, fmt.Errorf("duration value cannot be negative")
	}

	// 转换为秒
	var seconds int
	switch strings.ToLower(unit) {
	case "s":
		seconds = value
	case "m":
		seconds = value * 60
	case "h":
		seconds = value * 3600
	case "d":
		seconds = value * 86400
	default:
		return 0, fmt.Errorf("invalid duration unit: %s (use s/m/h/d)", unit)
	}

	return seconds, nil
}

// FormatDuration 格式化秒数为可读字符串
func FormatDuration(seconds int) string {
	if seconds == 0 {
		return "永久"
	}

	if seconds < 60 {
		return fmt.Sprintf("%d 秒", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		return fmt.Sprintf("%d 分钟", minutes)
	} else if seconds < 86400 {
		hours := seconds / 3600
		return fmt.Sprintf("%d 小时", hours)
	} else {
		days := seconds / 86400
		return fmt.Sprintf("%d 天", days)
	}
}

// FormatRemainingTime 格式化剩余时间
func FormatRemainingTime(expireAt time.Time) string {
	remaining := time.Until(expireAt)
	if remaining <= 0 {
		return "已到期"
	}

	seconds := int(remaining.Seconds())
	return FormatDuration(seconds)
}

// CalculateExpireTime 计算过期时间
func CalculateExpireTime(seconds int) *time.Time {
	if seconds == 0 {
		return nil // 永久
	}
	expireTime := time.Now().Add(time.Duration(seconds) * time.Second)
	return &expireTime
}

// FormatTimestamp 格式化时间戳
func FormatTimestamp(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

