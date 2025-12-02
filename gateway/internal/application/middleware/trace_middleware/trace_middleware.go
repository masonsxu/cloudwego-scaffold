// Package trace_middleware 提供请求追踪中间件
// 负责生成/提取请求追踪信息并传播到后续调用链
package trace_middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/masonsxu/cloudwego-scaffold/gateway/internal/infrastructure/errors"
)

// TraceMiddlewareImpl 追踪中间件实现
type TraceMiddlewareImpl struct{}

// NewTraceMiddleware 创建追踪中间件实例
func NewTraceMiddleware() TraceMiddlewareService {
	return &TraceMiddlewareImpl{}
}

// MiddlewareFunc 返回追踪中间件函数
// 此中间件执行以下操作：
// 1. 从 HTTP Headers 提取或生成 request_id 和 trace_id
// 2. 将追踪信息存储到 Hertz context 供 handler 层使用
// 3. 将追踪信息注入到 Go context (metainfo) 供 RPC 调用传播
func (m *TraceMiddlewareImpl) MiddlewareFunc() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 提取或生成追踪信息
		trace := errors.ExtractOrGenerateTraceContext(c)

		// 存储到 Hertz context 供 handler 层使用
		c.Set("trace_context", trace)

		// 注入到 Go context (metainfo) 供 RPC 调用传播
		ctx = errors.InjectTraceToContext(ctx, trace)

		// 继续处理请求
		c.Next(ctx)
	}
}
