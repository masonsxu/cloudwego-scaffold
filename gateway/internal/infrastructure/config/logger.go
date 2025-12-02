package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
)

// CreateLogger 根据配置创建slog.Logger实例
// 支持标准输出和文件输出，文件输出时自动启用日志轮转
func CreateLogger(cfg *Configuration) (*slog.Logger, error) {
	var level slog.Level

	// 解析日志级别
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

	// 创建日志选项
	opts := &slog.HandlerOptions{
		Level: level,
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

	// 根据格式创建对应的handler
	var handler slog.Handler

	switch cfg.Log.Format {
	case "json":
		handler = slog.NewJSONHandler(outputWriter, opts)
	default:
		handler = slog.NewTextHandler(outputWriter, opts)
	}

	logger := slog.New(handler)

	// 记录日志初始化信息
	logger.Info("Logger initialized",
		slog.String("level", cfg.Log.Level),
		slog.String("format", cfg.Log.Format),
		slog.String("output", cfg.Log.Output),
		slog.String("file_path", cfg.Log.FilePath),
		slog.Int("max_size_mb", cfg.Log.MaxSize),
		slog.Int("max_age_days", cfg.Log.MaxAge),
		slog.Int("max_backups", cfg.Log.MaxBackups),
	)

	return logger, nil
}

// createLogWriter 创建支持轮转的日志writer
func createLogWriter(cfg *Configuration) (*lumberjack.Logger, error) {
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
