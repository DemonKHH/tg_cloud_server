package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"tg_cloud_server/internal/common/config"
	"tg_cloud_server/internal/models"
)

// InitMySQL 初始化MySQL数据库连接
func InitMySQL(config *config.MySQLConfig) (*gorm.DB, error) {
	// 构建DSN连接字符串
	dsn := config.GetDSN()

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 默认静默日志
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// 获取底层数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	if config.MaxLifetime != "" {
		if lifetime, err := time.ParseDuration(config.MaxLifetime); err == nil {
			sqlDB.SetConnMaxLifetime(lifetime)
		}
	}

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	// 自动迁移表结构
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// 数据迁移：将旧的 two_way 和 frozen 状态迁移到新的布尔字段
	if err := migrateRestrictionStatus(db); err != nil {
		// 只记录警告，不阻止启动
		fmt.Printf("Warning: failed to migrate restriction status: %v\n", err)
	}

	return db, nil
}

// autoMigrate 自动迁移数据库表结构
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.TGAccount{},
		&models.Task{},
		&models.TaskLog{},
		&models.ProxyIP{},
		&models.RiskLog{},
		&models.VerifyCodeSession{},
	)
}

// migrateRestrictionStatus 迁移旧的 two_way 和 frozen 状态到新的布尔字段
func migrateRestrictionStatus(db *gorm.DB) error {
	// 将 status='frozen' 的账号设置 is_frozen=true，并将 status 改为 normal
	result := db.Model(&models.TGAccount{}).
		Where("status = ?", "frozen").
		Updates(map[string]interface{}{
			"is_frozen": true,
			"status":    "normal",
		})
	if result.Error != nil {
		return fmt.Errorf("failed to migrate frozen accounts: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		fmt.Printf("Migrated %d frozen accounts\n", result.RowsAffected)
	}

	// 将 status='two_way' 的账号设置 is_bidirectional=true，并将 status 改为 normal
	result = db.Model(&models.TGAccount{}).
		Where("status = ?", "two_way").
		Updates(map[string]interface{}{
			"is_bidirectional": true,
			"status":           "normal",
		})
	if result.Error != nil {
		return fmt.Errorf("failed to migrate two_way accounts: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		fmt.Printf("Migrated %d two_way accounts\n", result.RowsAffected)
	}

	return nil
}

// Close 关闭数据库连接
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
