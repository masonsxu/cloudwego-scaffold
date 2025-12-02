// Package errors 提供了 Hertz 网关层的统一错误处理机制
// 本文件提供追踪上下文的管理功能
package errors

import (
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// TraceContext 追踪上下文
// 用于在整个请求链路中传播追踪信息
type TraceContext struct {
	RequestID string // 请求唯一标识
	TraceID   string // 追踪链标识
}

// ExtractOrGenerateTraceContext 从请求中提取或生成追踪上下文
// 优先从 HTTP Headers 中提取，如果不存在则自动生成
func ExtractOrGenerateTraceContext(c *app.RequestContext) *TraceContext {
	requestID := string(c.GetHeader("X-Request-ID"))
	if requestID == "" {
		requestID = generateUUID()
	}

	traceID := string(c.GetHeader("X-Trace-ID"))
	if traceID == "" {
		traceID = requestID // 第一次请求时，trace_id = request_id
	}

	return &TraceContext{
		RequestID: requestID,
		TraceID:   traceID,
	}
}

// InjectTraceToContext 将追踪信息注入到 context 中（用于 RPC 调用）
// 使用 metainfo.WithPersistentValue 确保追踪信息通过 TTHeader 传递到 RPC 服务
func InjectTraceToContext(ctx context.Context, trace *TraceContext) context.Context {
	if trace == nil {
		return ctx
	}

	// 注入 request_id
	if trace.RequestID != "" {
		ctx = metainfo.WithPersistentValue(ctx, "request_id", trace.RequestID)
	}

	// 注入 trace_id
	if trace.TraceID != "" {
		ctx = metainfo.WithPersistentValue(ctx, "trace_id", trace.TraceID)
	}

	return ctx
}

// ExtractTraceFromContext 从 context 中提取追踪信息
// 用于在业务代码中获取追踪上下文
func ExtractTraceFromContext(ctx context.Context) *TraceContext {
	trace := &TraceContext{}

	// 尝试从 metainfo 提取
	if requestID, ok := metainfo.GetPersistentValue(ctx, "request_id"); ok {
		trace.RequestID = requestID
	}

	if traceID, ok := metainfo.GetPersistentValue(ctx, "trace_id"); ok {
		trace.TraceID = traceID
	}

	// 如果 metainfo 中没有，尝试从普通 context 提取
	if trace.RequestID == "" {
		if requestID, ok := ctx.Value("request_id").(string); ok {
			trace.RequestID = requestID
		}
	}

	if trace.TraceID == "" {
		if traceID, ok := ctx.Value("trace_id").(string); ok {
			trace.TraceID = traceID
		}
	}

	return trace
}

// generateUUID 生成唯一标识符
func generateUUID() string {
	return uuid.New().String()
}

// GetRequestIDFromContext 从 context 中获取 request_id（辅助函数）
func GetRequestIDFromContext(ctx context.Context) string {
	trace := ExtractTraceFromContext(ctx)
	return trace.RequestID
}

// GetTraceIDFromContext 从 context 中获取 trace_id（辅助函数）
func GetTraceIDFromContext(ctx context.Context) string {
	trace := ExtractTraceFromContext(ctx)
	return trace.TraceID
}
