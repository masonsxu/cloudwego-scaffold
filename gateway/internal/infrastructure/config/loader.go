package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// loadConfig 内部配置加载函数
func loadConfig() (*Configuration, error) {
	v := viper.New()

	// 设置环境变量支持
	v.SetEnvPrefix("") // 不设置前缀，直接使用环境变量名
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// 尝试加载 .env（可选，找不到不报错）
	loadDotEnvFirst([]string{".", "../..", "../../../"})

	// 设置默认值
	setDefaults(v)

	// 从环境变量映射到配置结构
	mapEnvVarsToConfig(v)

	// 解析配置
	var config Configuration
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}

	// 后处理配置
	postProcessConfig(&config)

	return &config, nil
}

// postProcessConfig 配置后处理
func postProcessConfig(config *Configuration) {
	// 确保日志目录存在
	if config.Log.Output == "file" && config.Log.FilePath != "" {
		if err := os.MkdirAll(config.Log.FilePath, 0o755); err != nil {
			// 日志目录创建失败，降级到标准输出
			config.Log.Output = "stdout"
		}
	}

	// 初始化服务配置映射
	if config.Client.Services == nil {
		config.Client.Services = make(map[string]ServiceConfig)
	}
}
