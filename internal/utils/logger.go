package utils

import (
	"io"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

// InitLogger 初始化日志系统（同时输出到控制台和文件，保留7天）
func InitLogger() error {
	// 确保日志目录存在
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 配置日志轮转
	logFile := filepath.Join(logDir, "bot_%Y%m%d.log")
	writer, err := rotatelogs.New(
		logFile,
		rotatelogs.WithMaxAge(7*24*time.Hour),                            // 保留7天
		rotatelogs.WithRotationTime(24*time.Hour),                        // 每天轮转
		rotatelogs.WithLinkName(filepath.Join(logDir, "bot_latest.log")), // 创建软链接指向最新日志
	)
	if err != nil {
		return err
	}

	// 配置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     false, // 文件中不使用颜色
		DisableColors:   false,
		PadLevelText:    true,
	})

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, writer)
	logrus.SetOutput(multiWriter)

	// 从环境变量读取日志级别
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	logrus.SetLevel(level)

	logrus.WithFields(logrus.Fields{
		"日志目录": logDir,
		"保留天数": 7,
		"日志级别": level.String(),
	}).Info("✅ 日志系统初始化完成")

	return nil
}
