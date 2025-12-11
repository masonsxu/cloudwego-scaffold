package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// CreateLogger 根据配置创建zerolog.Logger实例
// 支持标准输出和文件输出，文件输出时自动启用日志轮转
func CreateLogger(cfg *Config) (*zerolog.Logger, error) {
	// 解析日志级别
	var level zerolog.Level

	switch cfg.Log.Level {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	// 根据输出类型选择输出目标
	var outputWriter io.Writer

	if cfg.Log.Output == "file" && cfg.Log.FilePath != "" {
		// 文件输出：创建日志轮转writer
		writer, err := createLogWriter(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create log writer: %w", err)
		}

		outputWriter = writer
	} else {
		// 标准输出
		outputWriter = os.Stdout
	}

	// 创建 zerolog logger
	logger := zerolog.New(outputWriter).With().
		Timestamp().
		Logger().
		Level(level)

	// 记录日志初始化信息
	logger.Info().
		Str("level", cfg.Log.Level).
		Str("format", cfg.Log.Format).
		Str("output", cfg.Log.Output).
		Str("file_path", cfg.Log.FilePath).
		Int("max_size_mb", cfg.Log.MaxSize).
		Int("max_age_days", cfg.Log.MaxAge).
		Int("max_backups", cfg.Log.MaxBackups).
		Msg("Logger initialized")

	return &logger, nil
}

// createLogWriter 创建支持轮转的日志writer
func createLogWriter(cfg *Config) (*lumberjack.Logger, error) {
	// 确保日志目录存在
	logDir := filepath.Dir(cfg.Log.FilePath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// 创建lumberjack logger
	writer := &lumberjack.Logger{
		Filename:   cfg.Log.FilePath,
		MaxSize:    cfg.Log.MaxSize,    // 单个文件最大尺寸（MB）
		MaxAge:     cfg.Log.MaxAge,     // 文件最大保存天数
		MaxBackups: cfg.Log.MaxBackups, // 最多保留文件数
		Compress:   true,               // 是否压缩旧日志文件
	}

	return writer, nil
}
