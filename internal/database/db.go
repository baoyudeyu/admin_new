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

// Config 数据库配置
type Config struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	Charset         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int // seconds
}

// InitDB 初始化数据库连接
func InitDB(cfg Config) error {
	// 构建DSN连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
	)

	// 使用静默模式，减少GORM日志输出
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束以提高性能
		PrepareStmt:                              true, // 启用预编译语句缓存
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
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// 测试数据库连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	DB = db
	return nil
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	// 定义需要迁移的模型列表
	models := []interface{}{
		&models.AuthorizedGroup{},
		&models.GlobalAdmin{},
		&models.Blacklist{},
		&models.MuteList{},
		&models.OperationLog{},
		&models.SystemConfig{},
	}

	// 逐个迁移表结构，便于定位问题
	for _, model := range models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("表结构迁移失败 %T: %w", model, err)
		}
		logrus.Debugf("✓ 表迁移成功: %T", model)
	}

	return nil
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
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

