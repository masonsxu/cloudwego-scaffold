package wire

import (
	"log/slog"
	"os"

	"github.com/masonsxu/cloudwego-scaffold/rpc/identity-srv/config"
	"gorm.io/gorm"
)

// =============================================================================
// Provider Functions - 具体的依赖提供者实现
// =============================================================================

// ProvideDB 提供数据库连接实例
// Wire 依赖注入提供者，委托给 config 层处理所有初始化逻辑
func ProvideDB(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	return config.InitDB(cfg, logger)
}

// ProvideLogger 提供结构化日志实例
// 根据配置提供不同环境的日志配置
func ProvideLogger(cfg *config.Config) *slog.Logger {
	var (
		level   slog.Level
		handler slog.Handler
	)

	// 根据配置确定日志级别

	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 根据环境选择日志格式
	if cfg.App.Environment == "development" || cfg.App.Debug {
		// 开发环境使用可读性更好的文本格式
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	} else {
		// 生产环境使用结构化的 JSON 格式，便于日志收集和分析
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: false,
		})
	}

	logger := slog.New(handler)

	// 设置为默认日志器
	slog.SetDefault(logger)

	return logger
}

// ProvideCasbinConfig 提供 Casbin 配置
func ProvideCasbinConfig(cfg *config.Config) *config.CasbinConfig {
	return &cfg.Casbin
}

// =============================================================================
// Provider Options - 高级配置选项
// =============================================================================

// DBOption 数据库配置选项
type DBOption func(*gorm.DB) error

// ProvideDBWithOptions 提供带自定义选项的数据库连接
func ProvideDBWithOptions(
	cfg *config.Config,
	logger *slog.Logger,
	opts ...DBOption,
) (*gorm.DB, error) {
	db, err := ProvideDB(cfg, logger)
	if err != nil {
		return nil, err
	}

	// 应用自定义选项
	for _, opt := range opts {
		if err := opt(db); err != nil {
			return nil, err
		}
	}

	return db, nil
}

// WithDBDebugMode 启用数据库调试模式
func WithDBDebugMode() DBOption {
	return func(db *gorm.DB) error {
		return nil // db.Debug() 返回的是新实例，这里需要根据实际情况调整
	}
}

// WithDBMigration 执行数据库迁移
func WithDBMigration(models ...interface{}) DBOption {
	return func(db *gorm.DB) error {
		return db.AutoMigrate(models...)
	}
}

// LoggerOption 日志器配置选项
type LoggerOption func(*slog.HandlerOptions)

// WithLoggerSource 添加源码位置信息
func WithLoggerSource() LoggerOption {
	return func(opts *slog.HandlerOptions) {
		opts.AddSource = true
	}
}

// ProvideLoggerWithOptions 提供带自定义选项的日志器
func ProvideLoggerWithOptions(cfg *config.Config, opts ...LoggerOption) *slog.Logger {
	var level slog.Level

	switch cfg.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	// 应用自定义选项
	for _, opt := range opts {
		opt(handlerOpts)
	}

	var handler slog.Handler
	if cfg.App.Environment == "development" || cfg.App.Debug {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
