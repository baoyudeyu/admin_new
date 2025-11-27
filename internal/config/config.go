package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Telegram  TelegramConfig  `mapstructure:"telegram"`
	Database  DatabaseConfig  `mapstructure:"database"`
	System    SystemConfig    `mapstructure:"system"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	BotToken              string  `mapstructure:"bot_token"`
	AuthorIDs             []int64 `mapstructure:"author_ids"`
	NotificationChannelID int64   `mapstructure:"notification_channel_id"`
}

// IsAuthor 检查用户ID是否在作者列表中
func (t *TelegramConfig) IsAuthor(userID int64) bool {
	for _, authorID := range t.AuthorIDs {
		if authorID == userID {
			return true
		}
	}
	return false
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	Charset         string `mapstructure:"charset"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"` // 空闲连接超时
}

// SystemConfig 系统配置
type SystemConfig struct {
	RateLimitPerGroup int    `mapstructure:"rate_limit_per_group"`
	AdminEnabled      bool   `mapstructure:"admin_enabled"`
	LogLevel          string `mapstructure:"log_level"`
	Timezone          string `mapstructure:"timezone"`
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	CheckExpireInterval string `mapstructure:"check_expire_interval"`
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	GlobalConfig = &cfg
	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults() {
	viper.SetDefault("database.charset", "utf8mb4")
	viper.SetDefault("database.max_idle_conns", 20)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", 1800)
	viper.SetDefault("database.conn_max_idle_time", 600)

	viper.SetDefault("system.rate_limit_per_group", 5)
	viper.SetDefault("system.admin_enabled", true)
	viper.SetDefault("system.log_level", "info")
	viper.SetDefault("system.timezone", "Asia/Shanghai")

	viper.SetDefault("scheduler.check_expire_interval", "*/1 * * * *")
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return GlobalConfig
}
