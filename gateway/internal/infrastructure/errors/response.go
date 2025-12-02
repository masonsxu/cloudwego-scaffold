// Package errors 提供了 Hertz 网关层的统一错误处理机制
// 本文件专门处理错误响应的生成和发送
package errors

import (
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/masonsxu/cloudwego-scaffold/gateway/biz/model/core"
	"github.com/masonsxu/cloudwego-scaffold/gateway/biz/model/http_base"
)

// 错误码到 HTTP 状态码的映射
// 支持网关层错误码（100xxx）和 RPC 业务层认证相关错误码（201xxx）
var httpStatusMap = map[int32]int{
	// 系统级错误
	CodeSuccess:          http.StatusOK,
	CodeInvalidParams:    http.StatusBadRequest,
	CodeUnauthorized:     http.StatusUnauthorized,
	CodeForbidden:        http.StatusForbidden,
	CodeNotFound:         http.StatusNotFound,
	CodeInternalError:    http.StatusInternalServerError,
	CodeMethodNotAllowed: http.StatusMethodNotAllowed,

	// JWT认证相关错误
	CodeJWTTokenMissing:    http.StatusUnauthorized,
	CodeJWTTokenInvalid:    http.StatusUnauthorized,
	CodeJWTTokenExpired:    http.StatusUnauthorized,
	CodeJWTTokenNotActive:  http.StatusUnauthorized,
	CodeJWTTokenMalformed:  http.StatusBadRequest,
	CodeJWTValidationFail:  http.StatusUnauthorized,
	CodeJWTSigningError:    http.StatusInternalServerError,
	CodeJWTCreationFail:    http.StatusInternalServerError,
	CodeInvalidCredentials: http.StatusUnauthorized,

	// 网关特有错误
	CodeGatewayTimeout: http.StatusGatewayTimeout,
	CodeServiceDown:    http.StatusServiceUnavailable,
	CodeRateLimited:    http.StatusTooManyRequests,

	// RPC 业务层认证相关错误 (201xxx - identity_srv)
	// 这些错误来自下游 RPC 服务，需要在网关层映射为正确的 HTTP 状态码
	CodeRPCUserInactive:         http.StatusForbidden,    // 用户未激活
	CodeRPCUserNotFound:         http.StatusUnauthorized, // 用户不存在
	CodeRPCInvalidCredentials:   http.StatusUnauthorized, // 用户名或密码错误
	CodeRPCUserSuspended:        http.StatusForbidden,    // 用户已停用
	CodeRPCMustChangePassword:   http.StatusForbidden,    // 需要修改密码
	CodeRPCUserNoAvailableRoles: http.StatusForbidden,    // 用户没有可用角色
}

// AbortWithError 中断请求并返回错误响应
// 与成功响应保持一致的结构，便于前端统一处理
func AbortWithError(c *app.RequestContext, err APIError) {
	httpStatus := GetHTTPStatus(err.Code())

	// 生成请求ID和追踪ID
	requestID := generateRequestID(c)
	traceID := generateTraceID(c)
	timestamp := time.Now().UnixMilli()

	// 使用 OperationStatusResponseDTO 结构
	response := &http_base.OperationStatusResponseDTO{
		BaseResp: &http_base.BaseResponseDTO{
			Code:      err.Code(),
			Message:   err.Message(),
			RequestID: requestID,
			TraceID:   traceID,
			Timestamp: timestamp,
		},
	}

	c.JSON(httpStatus, response)
	c.Abort()
}

// AbortWithErrorMessage 中断请求并返回自定义错误消息
// 与成功响应保持一致的结构，便于前端统一处理
func AbortWithErrorMessage(c *app.RequestContext, err APIError, message string) {
	httpStatus := GetHTTPStatus(err.Code())

	// 生成请求ID和追踪ID
	requestID := generateRequestID(c)
	traceID := generateTraceID(c)
	timestamp := time.Now().UnixMilli()

	// 使用 OperationStatusResponseDTO 结构
	response := &http_base.OperationStatusResponseDTO{
		BaseResp: &http_base.BaseResponseDTO{
			Code:      err.Code(),
			Message:   message,
			RequestID: requestID,
			TraceID:   traceID,
			Timestamp: timestamp,
		},
	}

	c.JSON(httpStatus, response)
	c.Abort()
}

// generateRequestID 从请求上下文生成或获取请求ID
func generateRequestID(c *app.RequestContext) core.UUID {
	// 1. 优先从 HTTP Header 获取请求ID
	if requestID := c.GetHeader("X-Request-ID"); len(requestID) > 0 {
		return core.UUID(requestID)
	}

	// 2. 尝试从 trace_context 中获取
	if traceCtx, exists := c.Get("trace_context"); exists {
		if trace, ok := traceCtx.(*TraceContext); ok && trace != nil && trace.RequestID != "" {
			return trace.RequestID
		}
	}

	// 3. 如果都没有，生成新的 UUID
	return uuid.New().String()
}

// generateTraceID 从请求上下文生成或获取追踪ID
func generateTraceID(c *app.RequestContext) core.UUID {
	// 1. 优先从 HTTP Header 获取追踪ID
	if traceID := c.GetHeader("X-Trace-ID"); len(traceID) > 0 {
		return core.UUID(traceID)
	}

	// 2. 尝试从 trace_context 中获取
	if traceCtx, exists := c.Get("trace_context"); exists {
		if trace, ok := traceCtx.(*TraceContext); ok && trace != nil && trace.TraceID != "" {
			return trace.TraceID
		}
	}

	// 3. 如果没有，使用请求ID作为追踪ID（避免递归调用 generateRequestID）
	if requestID := c.GetHeader("X-Request-ID"); len(requestID) > 0 {
		return core.UUID(requestID)
	}

	// 4. 从 trace_context 获取 requestID
	if traceCtx, exists := c.Get("trace_context"); exists {
		if trace, ok := traceCtx.(*TraceContext); ok && trace != nil && trace.RequestID != "" {
			return trace.RequestID
		}
	}

	// 5. 最后生成新的
	return uuid.New().String()
}

// GetHTTPStatus 根据业务错误码获取对应的 HTTP 状态码
func GetHTTPStatus(code int32) int {
	if status, exists := httpStatusMap[code]; exists {
		return status
	}

	// 根据错误码规范判断
	switch {
	case code == 0:
		return http.StatusOK
	case code >= 20000 && code < 30000: // RPC 业务错误，统一返回 200
		return http.StatusOK
	default:
		// 系统级错误默认返回 500
		return http.StatusInternalServerError
	}
}

func GenerateRequestID(c *app.RequestContext) string {
	return generateRequestID(c)
}

func GenerateTraceID(c *app.RequestContext) string {
	return generateTraceID(c)
}

// FillBaseResp 填充BaseResponseDTO中的请求追踪字段
// 此函数应在handler层返回响应前调用，用于填充service层创建的空的request_id和trace_id
func FillBaseResp(c *app.RequestContext, baseResp *http_base.BaseResponseDTO) {
	if baseResp == nil {
		return
	}

	// 只在字段为空时填充，避免覆盖已有值
	if baseResp.RequestID == "" {
		baseResp.RequestID = GenerateRequestID(c)
	}

	if baseResp.TraceID == "" {
		baseResp.TraceID = GenerateTraceID(c)
	}
}
