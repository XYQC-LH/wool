package database

import (
	"fmt"
	"log"
	"time"

	"nexus-api/internal/config"
	"nexus-api/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// 配置 GORM 日志级别
	var logLevel logger.LogLevel
	switch cfg.Server.Mode {
	case "debug":
		logLevel = logger.Info
	case "release":
		logLevel = logger.Warn
	default:
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层 SQL DB
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.Database.ConnMaxLifetime) * time.Second)

	log.Println("Database connection established successfully")
	return nil
}

// AutoMigrate 自动迁移数据库表
//
// ✅ 已确认：所有核心表都已包含在迁移列表中
// - 用户和认证：User, Token, Asset
// - 渠道和模型：Channel, Model
// - 模型源头（核心调度）：ModelProvider, ProviderMetrics, CircuitEventRecord
// - 源头实例（号池调度）：ProviderInstance
// - 多模态配置：ModelCapability, ProviderPricingRule, ProviderRateLimitRule, IdempotencyKey
// - 日志和订单：Log, Order
// - 其他：ResourceAccount, Announcement, GenerationTask, Alert
func AutoMigrate() error {
	log.Println("Starting database migration...")

	err := DB.AutoMigrate(
		// 核心用户和认证
		&model.User{},
		&model.Token{},
		&model.Asset{},

		// 渠道和模型
		&model.Channel{},
		&model.Model{},
		&model.ModelAlias{},

		// ⭐ 模型源头（核心调度表）
		&model.ModelProvider{},
		&model.ProviderMetrics{},
		&model.CircuitEventRecord{},

		// ⭐ 源头实例（号池调度支持）
		&model.ProviderInstance{},

		// ⭐ 多模态配置表（能力/计费/限流/幂等）
		&model.ModelCapability{},
		&model.ProviderPricingRule{},
		&model.ProviderRateLimitRule{},
		&model.IdempotencyKey{},

		// 日志和订单
		&model.Log{},
		&model.Order{},

		// 其他
		&model.ResourceAccount{},
		&model.Announcement{},
		&model.GenerationTask{},
		&model.Alert{},

		// 系统设置（按 section 存储）
		&model.SystemSetting{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// Transaction 执行事务
func Transaction(fn func(tx *gorm.DB) error) error {
	return DB.Transaction(fn)
}

// Paginate 分页查询辅助函数
func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
