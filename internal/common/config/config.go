package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Database    DatabaseConfig    `mapstructure:"database"`
	Telegram    TelegramConfig    `mapstructure:"telegram"`
	AI          AIConfig          `mapstructure:"ai"`
	RiskControl RiskControlConfig `mapstructure:"risk_control"`
	Logging     LoggingConfig     `mapstructure:"logging"`
	JWT         JWTConfig         `mapstructure:"jwt"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	WebAPI        ServiceConfig `mapstructure:"web_api"`
	TGManager     ServiceConfig `mapstructure:"tg_manager"`
	TaskScheduler ServiceConfig `mapstructure:"task_scheduler"`
	AIService     ServiceConfig `mapstructure:"ai_service"`
}

// ServiceConfig 单个服务配置
type ServiceConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  string `mapstructure:"max_lifetime"`
}

// GetDSN 获取MySQL连接字符串
func (m *MySQLConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		m.Username, m.Password, m.Host, m.Port, m.Database)
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}

// GetAddr 获取Redis地址
func (r *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	APIID          int                  `mapstructure:"api_id"`
	APIHash        string               `mapstructure:"api_hash"`
	ConnectionPool ConnectionPoolConfig `mapstructure:"connection_pool"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
}

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	MaxConnections  int           `mapstructure:"max_connections"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	MessagesPerMinute int           `mapstructure:"messages_per_minute"`
	BurstSize         int           `mapstructure:"burst_size"`
	CooldownDuration  time.Duration `mapstructure:"cooldown_duration"`
}

// AIConfig AI服务配置
type AIConfig struct {
	OpenAI OpenAIConfig `mapstructure:"openai"`
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	APIKey      string        `mapstructure:"api_key"`
	Model       string        `mapstructure:"model"`
	MaxTokens   int           `mapstructure:"max_tokens"`
	Temperature float32       `mapstructure:"temperature"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// RiskControlConfig 风控配置
type RiskControlConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	CheckInterval    time.Duration `mapstructure:"check_interval"`
	MaxFailures      int           `mapstructure:"max_failures"`
	CooldownDuration time.Duration `mapstructure:"cooldown_duration"`
	HealthThreshold  float64       `mapstructure:"health_threshold"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey      string        `mapstructure:"secret_key"`
	ExpirationTime time.Duration `mapstructure:"expiration_time"`
	RefreshTime    time.Duration `mapstructure:"refresh_time"`
}

// globalConfig 全局配置实例
var globalConfig *Config

// Load 加载配置文件
func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置环境变量前缀
	viper.SetEnvPrefix("TG")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	globalConfig = &config
	return nil
}

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load() first")
	}
	return globalConfig
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务默认配置
	viper.SetDefault("server.web_api.host", "0.0.0.0")
	viper.SetDefault("server.web_api.port", 8080)
	viper.SetDefault("server.tg_manager.host", "0.0.0.0")
	viper.SetDefault("server.tg_manager.port", 8081)
	viper.SetDefault("server.task_scheduler.host", "0.0.0.0")
	viper.SetDefault("server.task_scheduler.port", 8082)
	viper.SetDefault("server.ai_service.host", "0.0.0.0")
	viper.SetDefault("server.ai_service.port", 8083)

	// 数据库默认配置
	viper.SetDefault("database.mysql.host", "localhost")
	viper.SetDefault("database.mysql.port", 3306)
	viper.SetDefault("database.mysql.max_open_conns", 100)
	viper.SetDefault("database.mysql.max_idle_conns", 10)
	viper.SetDefault("database.mysql.max_lifetime", "1h")

	viper.SetDefault("database.redis.host", "localhost")
	viper.SetDefault("database.redis.port", 6379)
	viper.SetDefault("database.redis.database", 0)
	viper.SetDefault("database.redis.pool_size", 10)

	// Telegram默认配置
	viper.SetDefault("telegram.connection_pool.max_connections", 1000)
	viper.SetDefault("telegram.connection_pool.idle_timeout", "30m")
	viper.SetDefault("telegram.connection_pool.cleanup_interval", "5m")

	viper.SetDefault("telegram.rate_limit.messages_per_minute", 30)
	viper.SetDefault("telegram.rate_limit.burst_size", 5)
	viper.SetDefault("telegram.rate_limit.cooldown_duration", "1m")

	// AI默认配置
	viper.SetDefault("ai.openai.model", "gpt-3.5-turbo")
	viper.SetDefault("ai.openai.max_tokens", 1000)
	viper.SetDefault("ai.openai.temperature", 0.7)
	viper.SetDefault("ai.openai.timeout", "30s")

	// 风控默认配置
	viper.SetDefault("risk_control.enabled", true)
	viper.SetDefault("risk_control.check_interval", "1m")
	viper.SetDefault("risk_control.max_failures", 3)
	viper.SetDefault("risk_control.cooldown_duration", "30m")
	viper.SetDefault("risk_control.health_threshold", 0.3)

	// 日志默认配置
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_backups", 3)
	viper.SetDefault("logging.max_age", 28)

	// JWT默认配置
	viper.SetDefault("jwt.expiration_time", "24h")
	viper.SetDefault("jwt.refresh_time", "168h") // 7 days
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	// 验证必需的配置
	if config.Database.MySQL.Username == "" {
		return fmt.Errorf("mysql username is required")
	}

	if config.Database.MySQL.Database == "" {
		return fmt.Errorf("mysql database is required")
	}

	if config.Telegram.APIID == 0 {
		return fmt.Errorf("telegram api_id is required")
	}

	if config.Telegram.APIHash == "" {
		return fmt.Errorf("telegram api_hash is required")
	}

	if config.JWT.SecretKey == "" {
		return fmt.Errorf("jwt secret_key is required")
	}

	return nil
}

// GetServiceAddr 获取服务地址
func (c *Config) GetServiceAddr(service string) string {
	switch service {
	case "web_api":
		return fmt.Sprintf("%s:%d", c.Server.WebAPI.Host, c.Server.WebAPI.Port)
	case "tg_manager":
		return fmt.Sprintf("%s:%d", c.Server.TGManager.Host, c.Server.TGManager.Port)
	case "task_scheduler":
		return fmt.Sprintf("%s:%d", c.Server.TaskScheduler.Host, c.Server.TaskScheduler.Port)
	case "ai_service":
		return fmt.Sprintf("%s:%d", c.Server.AIService.Host, c.Server.AIService.Port)
	default:
		return ""
	}
}
