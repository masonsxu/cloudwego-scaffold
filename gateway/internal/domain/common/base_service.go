package common

import (
	"context"
	"log/slog"

	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/errors"
)

// BaseService 基础服务结构
// 提供通用的服务方法模板，减少样板代码
type BaseService struct {
	logger          *slog.Logger
	responseBuilder *ResponseBuilder
}

// NewBaseService 创建基础服务
func NewBaseService(logger *slog.Logger) *BaseService {
	return &BaseService{
		logger:          logger,
		responseBuilder: NewResponseBuilder(),
	}
}

// Logger 获取日志记录器
func (bs *BaseService) Logger() *slog.Logger {
	return bs.logger
}

// ResponseBuilder 获取响应构建器
func (bs *BaseService) ResponseBuilder() *ResponseBuilder {
	return bs.responseBuilder
}

// ProcessRPCCall 处理RPC调用的通用模板
// 自动处理日志记录、错误处理等重复逻辑
func (bs *BaseService) ProcessRPCCall(
	ctx context.Context,
	logMsg string,
	rpcCall func(ctx context.Context) (interface{}, error),
	logFields ...interface{},
) (interface{}, error) {
	// 记录调用日志
	bs.logger.InfoContext(ctx, logMsg, logFields...)

	// 执行RPC调用
	result, err := rpcCall(ctx)
	if err != nil {
		// 统一错误处理
		bs.logger.ErrorContext(ctx, logMsg+"失败", "error", err)
		return nil, errors.ProcessRPCError(err, logMsg+"失败")
	}

	return result, nil
}

// ProcessRPCVoidCall 处理无返回值的RPC调用
func (bs *BaseService) ProcessRPCVoidCall(
	ctx context.Context,
	logMsg string,
	rpcCall func(ctx context.Context) error,
	logFields ...interface{},
) error {
	// 记录调用日志
	bs.logger.InfoContext(ctx, logMsg, logFields...)

	// 执行RPC调用
	err := rpcCall(ctx)
	if err != nil {
		// 统一错误处理
		bs.logger.ErrorContext(ctx, logMsg+"失败", "error", err)
		return errors.ProcessRPCError(err, logMsg+"失败")
	}

	return nil
}

// LogInfo 便捷的信息日志方法
func (bs *BaseService) LogInfo(ctx context.Context, msg string, fields ...interface{}) {
	bs.logger.InfoContext(ctx, msg, fields...)
}

// LogError 便捷的错误日志方法
func (bs *BaseService) LogError(ctx context.Context, msg string, err error, fields ...interface{}) {
	allFields := append([]interface{}{"error", err}, fields...)
	bs.logger.ErrorContext(ctx, msg, allFields...)
}
