package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzZerolog "github.com/hertz-contrib/logger/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// CreateLogger 根据配置创建zerolog.Logger实例
// 支持标准输出和文件输出，文件输出时自动启用日志轮转
func CreateLogger(cfg *Configuration) (*hertzZerolog.Logger, error) {
	// 解析日志级别
	var level hlog.Level

	logLevel := cfg.Log.Level

	// 如果开启了调试模式且未显式设置日志级别，自动使用 debug 级别
	if cfg.Server.Debug && logLevel == "" {
		logLevel = "debug"
	}

	switch logLevel {
	case "debug":
		level = hlog.LevelDebug
	case "info":
		level = hlog.LevelInfo
	case "warn":
		level = hlog.LevelWarn
	case "error":
		level = hlog.LevelError
	default:
		level = hlog.LevelInfo
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

	// 创建 zerolog logger 选项
	opts := []hertzZerolog.Opt{
		hertzZerolog.WithLevel(level),
		hertzZerolog.WithOutput(outputWriter),
		hertzZerolog.WithTimestamp(),
	}

	// 如果配置了 JSON 格式，zerolog 默认就是 JSON，否则使用格式化输出
	// zerolog 默认是 JSON 格式，如果需要文本格式，可以通过自定义格式实现
	// 这里我们保持 JSON 格式，因为这是 zerolog 的优势

	// 创建 logger
	logger := hertzZerolog.New(opts...)

	// 记录日志初始化信息（使用 hlog 接口）
	logger.Infof(
		"Logger initialized: level=%s, format=%s, output=%s, file_path=%s, max_size_mb=%d, max_age_days=%d, max_backups=%d",
		cfg.Log.Level,
		cfg.Log.Format,
		cfg.Log.Output,
		cfg.Log.FilePath,
		cfg.Log.MaxSize,
		cfg.Log.MaxAge,
		cfg.Log.MaxBackups,
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
