// Package wire 中间件层依赖注入提供者
package wire

import (
	"log/slog"

	"github.com/google/wire"
	corsmdw "github.com/masonsxu/cloudwego-scaffold/gateway/internal/application/middleware/cors_middleware"
	errormw "github.com/masonsxu/cloudwego-scaffold/gateway/internal/application/middleware/error_middleware"
	jwtmdw "github.com/masonsxu/cloudwego-scaffold/gateway/internal/application/middleware/jwt_middleware"
	tracemdw "github.com/masonsxu/cloudwego-scaffold/gateway/internal/application/middleware/trace_middleware"
	identityService "github.com/masonsxu/cloudwego-scaffold/gateway/internal/domain/service/identity"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/config"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/redis"
)

// MiddlewareSet 中间件层依赖注入集合
var MiddlewareSet = wire.NewSet(
	ProvideTraceMiddleware,
	ProvideCORSMiddleware,
	ProvideErrorHandlerMiddleware,
	ProvideJWTMiddleware,
	// ProvideCasbinMiddleware,
	NewMiddlewareContainer,
)

// MiddlewareContainer 中间件容器
// 统一管理所有中间件实例
type MiddlewareContainer struct {
	TraceMiddleware        tracemdw.TraceMiddlewareService
	CORSMiddleware         corsmdw.CORSMiddlewareService
	ErrorHandlerMiddleware errormw.ErrorHandlerMiddlewareService
	JWTMiddleware          jwtmdw.JWTMiddlewareService
	// CasbinMiddleware       casbinmw.CasbinMiddleware
}

// NewMiddlewareContainer 创建中间件容器
func NewMiddlewareContainer(
	traceMiddleware tracemdw.TraceMiddlewareService,
	corsMiddleware corsmdw.CORSMiddlewareService,
	errorHandlerMiddleware errormw.ErrorHandlerMiddlewareService,
	jwtMiddleware jwtmdw.JWTMiddlewareService,
	// casbinMiddleware casbinmw.CasbinMiddleware,
) *MiddlewareContainer {
	return &MiddlewareContainer{
		TraceMiddleware:        traceMiddleware,
		CORSMiddleware:         corsMiddleware,
		ErrorHandlerMiddleware: errorHandlerMiddleware,
		JWTMiddleware:          jwtMiddleware,
		// CasbinMiddleware:       casbinMiddleware,
	}
}

// ProvideTraceMiddleware 提供追踪中间件
// 自动生成和传播请求追踪信息
func ProvideTraceMiddleware(logger *slog.Logger) tracemdw.TraceMiddlewareService {
	middleware := tracemdw.NewTraceMiddleware()

	logger.Info("Trace middleware created successfully")

	return middleware
}

// ProvideJWTMiddleware 提供JWT中间件
// 配置JWT认证中间件，用于API权限控制
func ProvideJWTMiddleware(
	identityService identityService.Service,
	jwtConfig *config.JWTConfig,
	tokenCache redis.TokenCacheService,
	logger *slog.Logger,
) jwtmdw.JWTMiddlewareService {
	middleware, err := jwtmdw.JWTMiddlewareProvider(identityService, jwtConfig, tokenCache, logger)
	if err != nil {
		logger.Error("Failed to create JWT middleware", "error", err)
		panic(err)
	}

	logger.Info("JWT middleware created successfully")

	return middleware
}

// ProvideCORSMiddleware 提供跨域中间件
// 处理跨域资源共享(CORS)配置
func ProvideCORSMiddleware(
	cfg *config.Configuration,
	logger *slog.Logger,
) corsmdw.CORSMiddlewareService {
	middleware := corsmdw.NewCORSMiddleware(&cfg.Middleware.CORS, logger)
	logger.Info("CORS middleware created successfully")

	return middleware
}

// ProvideErrorHandlerMiddleware 提供错误处理中间件
// 统一处理请求中的错误响应
func ProvideErrorHandlerMiddleware(
	cfg *config.Configuration,
	logger *slog.Logger,
) errormw.ErrorHandlerMiddlewareService {
	middleware := errormw.NewErrorHandlerMiddleware(&cfg.Middleware.ErrorHandler, logger)
	logger.Info("Error Handler middleware created successfully")

	return middleware
}

// ProvideCasbinMiddleware 提供Casbin权限中间件
// 使用 hertz-contrib/casbin 官方中间件，遵循最佳实践
// func ProvideCasbinMiddleware(
// 	cfg *config.Configuration,
// 	logger *slog.Logger,
// 	manager *casbin.CasbinManager,
// ) casbinmw.CasbinMiddleware {
// 	middleware, err := casbinmw.NewCasbinMiddleware(manager, logger, nil)
// 	if err != nil {
// 		logger.Error("Failed to create casbin middleware", "error", err)
// 		panic(err)
// 	}

// 	logger.Info("Casbin middleware created successfully")

// 	return middleware
// }
