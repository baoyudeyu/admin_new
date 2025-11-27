package database

import (
	"fmt"
	"time"

	"admin-bot/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Config 数据库配置结构
type Config struct {
	Host            string // 数据库主机地址
	Port            int    // 数据库端口
	Username        string // 数据库用户名
	Password        string // 数据库密码
	Database        string // 数据库名称
	Charset         string // 字符集
	MaxIdleConns    int    // 最大空闲连接数
	MaxOpenConns    int    // 最大打开连接数
	ConnMaxLifetime int    // 连接最大生命周期（秒）
	ConnMaxIdleTime int    // 空闲连接超时（秒）
}

// InitDB 初始化数据库连接
func InitDB(cfg Config) error {
	// 构建DSN连接字符串 - 优化连接参数
	// 使用 charset=utf8mb4 让 MySQL 8.0 使用数据库默认的 collation
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	// 配置GORM选项
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent), // 静默模式，减少日志输出
		DisableForeignKeyConstraintWhenMigrating: true,                                  // 禁用外键约束，提高性能
		PrepareStmt:                              true,                                  // 启用预编译语句缓存
		SkipDefaultTransaction:                   true,                                  // 跳过默认事务，提升性能
	})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 获取底层SQL连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 配置连接池参数 - 优化稳定性和性能
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)                                    // 设置空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)                                    // 设置数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second) // 设置连接可复用的最大时间

	// 设置空闲连接超时，避免长时间空闲连接被服务器断开
	// 建议设置为比 RDS 超时时间短的值，确保在服务器断开前主动关闭
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)
	} else {
		// 默认 5 分钟空闲超时
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	}

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	DB = db

	logrus.WithFields(logrus.Fields{
		"最大空闲连接":   cfg.MaxIdleConns,
		"最大打开连接":   cfg.MaxOpenConns,
		"连接最大生命周期": fmt.Sprintf("%d秒", cfg.ConnMaxLifetime),
		"空闲连接超时":   fmt.Sprintf("%d秒", cfg.ConnMaxIdleTime),
	}).Debug("数据库连接池配置")

	return nil
}

// AutoMigrate 智能迁移数据库表结构
func AutoMigrate() error {
	// 定义需要管理的模型列表
	tableModels := []interface{}{
		&models.AuthorizedGroup{}, // 授权群组表
		&models.GlobalAdmin{},     // 全局管理员表
		&models.Blacklist{},       // 黑名单表
		&models.MuteList{},        // 禁言列表表
		&models.OperationLog{},    // 操作日志表
		&models.SystemConfig{},    // 系统配置表
		&models.UserCache{},       // 用户缓存表
	}

	// 批量迁移所有表结构（GORM 会自动处理表的创建和更新）
	if err := DB.AutoMigrate(tableModels...); err != nil {
		return fmt.Errorf("数据库表结构迁移失败: %w", err)
	}

	logrus.Info("✅ 数据库表结构同步完成")
	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// PingDB 数据库健康检查
func PingDB() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		logrus.Errorf("数据库连接检查失败: %v", err)
		return err
	}

	return nil
}

// ReconnectDB 重新连接数据库
func ReconnectDB(cfg Config) error {
	logrus.Warn("🔄 正在尝试重新连接数据库...")

	// 关闭旧连接
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	// 重新初始化连接
	err := InitDB(cfg)
	if err != nil {
		logrus.Errorf("❌ 数据库重连失败: %v", err)
		return err
	}

	logrus.Info("✅ 数据库重连成功")
	return nil
}

// PingDBWithRetry 带重试的数据库健康检查
func PingDBWithRetry(maxRetries int) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := PingDB()
		if err == nil {
			return nil
		}

		lastErr = err
		if i < maxRetries-1 {
			waitTime := time.Duration(i+1) * time.Second
			logrus.WithFields(logrus.Fields{
				"重试次数": i + 1,
				"等待时间": waitTime,
			}).Warn("⚠️ 数据库连接失败，正在重试...")
			time.Sleep(waitTime)
		}
	}

	return fmt.Errorf("数据库连接失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// GetDBStats 获取数据库连接池统计信息
func GetDBStats() string {
	if DB == nil {
		return "数据库未初始化"
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Sprintf("获取数据库实例失败: %v", err)
	}

	stats := sqlDB.Stats()
	return fmt.Sprintf("打开连接: %d, 使用中: %d, 空闲: %d, 等待: %d",
		stats.OpenConnections,
		stats.InUse,
		stats.Idle,
		stats.WaitCount,
	)
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return fmt.Errorf("获取数据库连接失败: %w", err)
		}
		logrus.Info("🔌 正在关闭数据库连接...")
		return sqlDB.Close()
	}
	return nil
}
