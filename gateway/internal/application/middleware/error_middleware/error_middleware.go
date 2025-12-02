package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/application/context/auth_context"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/config"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/errors"
)

// ErrorHandlerMiddlewareImpl 错误处理中间件实现
type ErrorHandlerMiddlewareImpl struct {
	config *config.ErrorHandlerConfig
	logger *slog.Logger
}

// NewErrorHandlerMiddleware 创建新的错误处理中间件实例
func NewErrorHandlerMiddleware(
	config *config.ErrorHandlerConfig,
	logger *slog.Logger,
) ErrorHandlerMiddlewareService {
	if logger == nil {
		logger = slog.Default()
	}

	return &ErrorHandlerMiddlewareImpl{
		config: config,
		logger: logger,
	}
}

// MiddlewareFunc 返回Hertz中间件处理函数
func (ehm *ErrorHandlerMiddlewareImpl) MiddlewareFunc() app.HandlerFunc {
	return app.HandlerFunc(func(ctx context.Context, c *app.RequestContext) {
		// 如果中间件被禁用，直接跳过
		if !ehm.config.Enabled {
			c.Next(ctx)
			return
		}

		// 记录请求开始时间
		startTime := time.Now()

		// 如果启用panic恢复，设置recover
		if ehm.config.EnablePanicRecovery {
			defer func() {
				if r := recover(); r != nil {
					ehm.handlePanicError(ctx, c, r)
				}
			}()
		}

		// 记录请求信息
		if ehm.config.EnableRequestLogging {
			ehm.logRequestInfo(ctx, c)
		}

		// 执行下一个处理器
		c.Next(ctx)

		if c.Response.StatusCode() == 500 && len(c.Response.Body()) == 0 {
			ehm.logger.Warn("Casbin permission denied",
				"method", string(c.Method()), "path", string(c.Request.Path()))
			c.SetStatusCode(403)
			c.JSON(403, map[string]string{"msg": "权限不足"})

			return
		}

		// 处理完成后检查是否有错误
		if c.IsAborted() {
			// 请求已被中断，可能是认证失败或其他错误
			return
		}

		// 检查响应状态码，处理HTTP状态错误
		if statusCode := c.Response.StatusCode(); statusCode >= 400 {
			ehm.handleHTTPStatusError(ctx, c, statusCode)
			return
		}

		// 记录响应信息
		if ehm.config.EnableResponseLogging {
			ehm.logResponseInfo(ctx, c, startTime)
		}
	})
}

// handlePanicError 处理panic错误
func (ehm *ErrorHandlerMiddlewareImpl) handlePanicError(
	ctx context.Context,
	c *app.RequestContext,
	r any,
) {
	// 获取堆栈信息
	bufSize := ehm.config.MaxStackTraceSize
	if bufSize <= 0 {
		bufSize = 4096 // 默认值
	}

	buf := make([]byte, bufSize)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	// 记录详细的panic日志
	ehm.logger.Error("Panic recovered in error handler",
		"panic", r,
		"path", string(c.Request.Path()),
		"method", string(c.Method()),
		"user_agent", string(c.UserAgent()),
		"remote_addr", c.RemoteAddr(),
		"stack_trace", stackTrace,
	)

	// 构建错误响应
	bizErr := errors.ErrInternal
	if ehm.config.EnableDetailedErrors {
		bizErr = bizErr.WithMessage(fmt.Sprintf("内部服务器错误：%v", r))
	}

	// 使用统一的错误处理函数
	errors.AbortWithError(c, bizErr)
}

// handleHTTPStatusError 处理HTTP状态码错误
func (ehm *ErrorHandlerMiddlewareImpl) handleHTTPStatusError(
	ctx context.Context,
	c *app.RequestContext,
	statusCode int,
) {
	// 记录用户信息（如果有的话）
	logFields := []any{
		"status_code", statusCode,
		"path", string(c.Request.Path()),
		"method", string(c.Method()),
	}

	// 如果有用户上下文，记录用户信息
	if userID, ok := auth_context.GetCurrentUserProfileID(c); ok && userID != "" {
		logFields = append(logFields, "user_id", userID)
	}

	if orgID, ok := auth_context.GetCurrentOrganizationID(c); ok && orgID != "" {
		logFields = append(logFields, "org_id", orgID)
	}

	ehm.logger.Warn("HTTP status error", logFields...)

	// 如果启用了详细错误信息，可以在这里添加更多上下文
	if ehm.config.EnableDetailedErrors {
		// 可以在这里添加更详细的错误信息处理
		ehm.logger.Debug(
			"Detailed error context",
			"remote_addr",
			c.RemoteAddr(),
			"user_agent",
			string(c.UserAgent()),
		)
	}

	errors.HandleHTTPStatusError(c, statusCode)
}

// logRequestInfo 记录请求信息
func (ehm *ErrorHandlerMiddlewareImpl) logRequestInfo(ctx context.Context, c *app.RequestContext) {
	UserID, hasUserID := auth_context.GetCurrentUserProfileID(c)
	orgID, hasOrg := auth_context.GetCurrentOrganizationID(c)

	logFields := []any{
		"method", string(c.Method()),
		"path", string(c.Request.Path()),
		"query", string(c.Request.QueryString()),
		"remote_addr", c.RemoteAddr(),
	}

	if hasUserID && UserID != "" {
		logFields = append(logFields, "user_id", UserID)
	}

	if hasOrg && orgID != "" {
		logFields = append(logFields, "org_id", orgID)
	}

	ehm.logger.Info("Request started", logFields...)
}

// logResponseInfo 记录响应信息
func (ehm *ErrorHandlerMiddlewareImpl) logResponseInfo(
	ctx context.Context,
	c *app.RequestContext,
	startTime time.Time,
) {
	duration := time.Since(startTime)
	statusCode := c.Response.StatusCode()

	logFields := []any{
		"method", string(c.Method()),
		"path", string(c.Request.Path()),
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"response_size", len(c.Response.Body()),
	}

	if userID, ok := auth_context.GetCurrentUserProfileID(c); ok && userID != "" {
		logFields = append(logFields, "user_id", userID)
	}

	ehm.logger.Info("Request completed", logFields...)
}
