package config

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/masonsxu/cloudwego-scaffold/rpc/identity-srv/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化数据库连接，提供给wire使用的函数
func InitDB(cfg *Config, loggerSvc *slog.Logger) (*gorm.DB, error) {
	return NewDB(&cfg.Database, loggerSvc)
}

// NewDB initializes and returns a new GORM database instance.
func NewDB(cfg *DatabaseConfig, loggerSvc *slog.Logger) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
			cfg.Host,
			cfg.Username,
			cfg.Password,
			cfg.DBName,
			cfg.Port,
			cfg.SSLMode,
			cfg.Timezone,
		)

		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s", cfg.Driver)
	}

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(dialector, config)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层SQL DB失败: %v", err)
	}

	// 设置连接池参数
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		loggerSvc.Error("Failed to ping database", "error", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库自动迁移失败: %v", err)
	}

	// 执行种子数据初始化（幂等）
	if err := SeedDatabase(db, loggerSvc, cfg); err != nil {
		// Seeder 失败只记录警告，不阻止服务启动
		loggerSvc.Warn("⚠️  种子数据初始化失败: %v", "error", err)
	}

	loggerSvc.Info("Database connected successfully",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.DBName,
		"max_idle_conns", cfg.MaxIdleConns,
		"max_open_conns", cfg.MaxOpenConns,
		"max_conn_lifetime", cfg.ConnMaxLifetime,
		"max_conn_idle_time", cfg.ConnMaxIdleTime,
	)

	return db, nil
}

func autoMigrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	log.Println("开始数据库自动迁移...")

	// 按照依赖顺序创建表结构
	// 分布式系统中不使用数据库外键约束，通过应用层维护数据一致性
	err := db.AutoMigrate(
		&models.UserProfile{},
		&models.UserMembership{},
		&models.Organization{},
		&models.Department{},
		&models.OrganizationLogo{},
		&models.RoleDefinition{},
		&models.UserRoleAssignment{},
		&models.Menu{},
	)
	if err != nil {
		return fmt.Errorf("自动迁移失败: %v", err)
	}

	log.Println("数据库自动迁移完成")

	return nil
}
