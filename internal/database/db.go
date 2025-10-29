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
}

// InitDB 初始化数据库连接
func InitDB(cfg Config) error {
	// 构建DSN连接字符串 - 优化字符集配置
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&collation=utf8mb4_unicode_ci",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
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

	// 配置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)                                    // 设置空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)                                    // 设置数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second) // 设置连接可复用的最大时间

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	DB = db

	// 自动设置数据库字符集
	if err := ensureDatabaseCharset(cfg.Database); err != nil {
		logrus.Warnf("设置数据库字符集失败: %v（不影响程序运行）", err)
	}

	return nil
}

// ensureDatabaseCharset 确保数据库使用 utf8mb4 字符集
func ensureDatabaseCharset(dbName string) error {
	// 检查当前数据库字符集
	var charset, collation string
	err := DB.Raw("SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", dbName).
		Row().Scan(&charset, &collation)

	if err != nil {
		return fmt.Errorf("检查数据库字符集失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"数据库":  dbName,
		"字符集":  charset,
		"排序规则": collation,
	}).Info("📊 当前数据库字符集")

	// 如果不是 utf8mb4，则自动设置
	if charset != "utf8mb4" {
		logrus.Warnf("⚠️  数据库字符集为 %s，正在自动修改为 utf8mb4...", charset)

		alterSQL := fmt.Sprintf("ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
		if err := DB.Exec(alterSQL).Error; err != nil {
			return fmt.Errorf("修改数据库字符集失败: %w", err)
		}

		logrus.Info("✅ 数据库字符集已自动设置为 utf8mb4")
	} else {
		logrus.Debug("✓ 数据库字符集已是 utf8mb4，无需修改")
	}

	return nil
}

// AutoMigrate 智能迁移数据库表结构（仅在表不存在时创建）
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

	// 逐个检查并创建表
	for _, model := range tableModels {
		// 检查表是否存在
		if !DB.Migrator().HasTable(model) {
			// 表不存在，创建新表
			if err := DB.AutoMigrate(model); err != nil {
				return fmt.Errorf("创建表失败 %T: %w", model, err)
			}
			logrus.Infof("📋 创建新表: %T", model)
		} else {
			// 表已存在，检查是否需要更新字段
			if err := DB.AutoMigrate(model); err != nil {
				return fmt.Errorf("更新表结构失败 %T: %w", model, err)
			}
			logrus.Debugf("✓ 表已存在: %T", model)
		}
	}

	// 自动转换已存在表的字符集
	if err := convertExistingTables(); err != nil {
		logrus.Warnf("转换表字符集失败: %v（不影响程序运行）", err)
	}

	return nil
}

// convertExistingTables 转换已存在表的字符集为 utf8mb4
func convertExistingTables() error {
	// 需要转换的表名列表
	tables := []string{
		"authorized_groups",
		"global_admins",
		"blacklist",
		"mute_list",
		"operation_log",
		"system_config",
		"user_cache",
	}

	for _, tableName := range tables {
		// 检查表是否存在
		var exists bool
		err := DB.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?)", tableName).
			Scan(&exists).Error

		if err != nil || !exists {
			continue // 表不存在，跳过
		}

		// 检查表的字符集
		var tableCollation string
		err = DB.Raw("SELECT TABLE_COLLATION FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", tableName).
			Scan(&tableCollation).Error

		if err != nil {
			logrus.Warnf("检查表 %s 字符集失败: %v", tableName, err)
			continue
		}

		// 如果表的排序规则不是 utf8mb4，则转换
		if !startsWith(tableCollation, "utf8mb4") {
			logrus.Infof("🔄 转换表 %s 字符集为 utf8mb4...", tableName)

			convertSQL := fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", tableName)
			if err := DB.Exec(convertSQL).Error; err != nil {
				logrus.Warnf("转换表 %s 字符集失败: %v", tableName, err)
				continue
			}

			logrus.Infof("✅ 表 %s 已转换为 utf8mb4", tableName)
		} else {
			logrus.Debugf("✓ 表 %s 已是 utf8mb4，无需转换", tableName)
		}
	}

	return nil
}

// startsWith 检查字符串是否以指定前缀开头
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
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
