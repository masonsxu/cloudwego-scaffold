//go:build wireinject

// Package wire Wire依赖注入配置
// 使用Google Wire进行依赖注入管理，实现清晰的分层架构
package wire

import (
	"log/slog"

	"github.com/google/wire"
)

// AllSet 所有依赖注入集合
// 按照分层架构组织：基础设施层 -> 应用层 -> 领域层 -> 中间件层
var AllSet = wire.NewSet(
	// 基础设施层
	InfrastructureSet,

	// 应用层
	ApplicationSet,

	// 领域服务层
	DomainServiceSet,

	// 中间件层
	MiddlewareSet,

	// 容器
	NewServiceContainer,
)

// InitializeService 初始化服务容器
// Wire会自动生成依赖注入代码
func InitializeService() (*ServiceContainer, error) {
	wire.Build(AllSet)
	return &ServiceContainer{}, nil
}

// InitializeMiddleware 初始化中间件容器
// Wire会自动生成依赖注入代码
func InitializeMiddleware() (*MiddlewareContainer, error) {
	wire.Build(AllSet)
	return &MiddlewareContainer{}, nil
}

// InitializeLogger 初始化日志服务
// 提供给外部模块使用的日志实例
func InitializeLogger() *slog.Logger {
	wire.Build(AllSet)
	return &slog.Logger{}
}
