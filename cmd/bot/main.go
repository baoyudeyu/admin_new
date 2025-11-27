package main

import (
	"admin-bot/internal/bot"
	"admin-bot/internal/config"
	"admin-bot/internal/database"
	"admin-bot/internal/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化日志
	if err := initLogger(); err != nil {
		logrus.Fatalf("❌ 日志系统初始化失败: %v", err)
	}

	// 打印欢迎信息
	printWelcome()

	logrus.Info("========================================")
	logrus.Info("正在启动 Telegram 群管机器人...")
	logrus.Info("========================================")

	// 加载配置
	logrus.Info("📝 正在加载配置文件...")
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logrus.Fatalf("❌ 配置文件加载失败: %v", err)
	}
	logrus.Info("✅ 配置文件加载成功")

	// 初始化数据库连接
	logrus.Info("🗄️  正在连接数据库...")
	dbConfig := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Username:        cfg.Database.Username,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		Charset:         cfg.Database.Charset,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	if err := database.InitDB(dbConfig); err != nil {
		logrus.Fatalf("❌ 数据库连接失败: %v", err)
	}
	logrus.WithFields(logrus.Fields{
		"主机": cfg.Database.Host,
		"端口": cfg.Database.Port,
		"库名": cfg.Database.Database,
	}).Info("✅ 数据库连接成功")

	// 自动迁移数据库表结构
	logrus.Info("🔄 正在同步数据库表结构...")
	if err := database.AutoMigrate(); err != nil {
		logrus.Fatalf("❌ 表结构同步失败: %v", err)
	}
	logrus.Info("✅ 表结构同步完成")

	// 创建机器人
	logrus.Info("🤖 正在初始化 Telegram 机器人...")
	botInstance, err := bot.NewBot(cfg)
	if err != nil {
		logrus.Fatalf("❌ 机器人初始化失败: %v", err)
	}
	logrus.WithFields(logrus.Fields{
		"作者IDs": cfg.Telegram.AuthorIDs,
		"频道ID":  cfg.Telegram.NotificationChannelID,
		"限流设置":  cfg.System.RateLimitPerGroup,
		"群管权限":  cfg.System.AdminEnabled,
	}).Info("✅ 机器人初始化成功")

	// 启动机器人
	logrus.Info("🚀 正在启动机器人服务...")
	go func() {
		err := botInstance.Start()
		if err != nil {
			logrus.Errorf("❌ 机器人错误: %v", err)
		}
	}()

	logrus.Info("========================================")
	logrus.Info("✨ 机器人运行中！")
	logrus.Info("📱 等待接收 Telegram 消息...")
	logrus.Info("🛑 按 Ctrl+C 停止运行")
	logrus.Info("========================================")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("")
	logrus.Info("========================================")
	logrus.Info("🛑 收到停止信号")
	logrus.Info("📊 正在停止机器人服务...")
	logrus.Info("========================================")

	// 优雅关闭
	botInstance.Stop()
	database.Close()

	logrus.Info("✅ 机器人已安全停止")
	logrus.Info("👋 再见！")
}

// printWelcome 打印欢迎信息
func printWelcome() {
	welcome := `
╔═══════════════════════════════════════════╗
║                                           ║
║       Telegram 多群组群管机器人            ║
║                                           ║
║           版本: 1.0.0                     ║
║           作者: Admin System              ║
║                                           ║
╚═══════════════════════════════════════════╝
`
	logrus.Info(welcome)
}

// initLogger 初始化日志
func initLogger() error {
	// 调用 utils 包的日志初始化函数
	return utils.InitLogger()
}
