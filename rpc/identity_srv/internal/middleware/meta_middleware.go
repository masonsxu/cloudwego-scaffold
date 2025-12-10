// Package middleware 提供RPC服务端中间件
package middleware

import (
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MetaInfoMiddleware RPC服务端追踪中间件
// 职责：
// 1. 从 metainfo 提取 request_id 和 trace_id
// 2. 如果不存在，自动生成并注入到 metainfo
// 3. 记录追踪信息日志
//
// 设计原则：
// - 直接使用 metainfo，不使用 context.WithValue（避免重复存储）
// - 确保每个请求都有完整的追踪 ID（自动生成缺失的）
// - 聚焦核心功能（只处理 request_id 和 trace_id）
type MetaInfoMiddleware struct {
	logger *zerolog.Logger
}

// NewMetaInfoMiddleware 创建新的MetaInfo中间件实例
func NewMetaInfoMiddleware(logger *zerolog.Logger) *MetaInfoMiddleware {
	if logger == nil {
		defaultLogger := zerolog.Nop()
		logger = &defaultLogger
	}

	return &MetaInfoMiddleware{
		logger: logger,
	}
}

// ServerMiddleware 返回Kitex服务端中间件
func (m *MetaInfoMiddleware) ServerMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) error {
			// 确保追踪 ID 存在，不存在则自动生成
			ctx = m.ensureTraceIDs(ctx)

			// 记录追踪信息
			m.logTraceInfo(ctx)

			// 执行业务逻辑
			return next(ctx, req, resp)
		}
	}
}

// ensureTraceIDs 确保追踪 ID 存在，不存在则自动生成
// 关键设计：
// - 直接操作 metainfo，不使用 context.WithValue
// - 缺失的 ID 会自动生成并注入到 metainfo
// - trace_id 默认使用 request_id 的值
func (m *MetaInfoMiddleware) ensureTraceIDs(ctx context.Context) context.Context {
	var (
		requestID, traceID           string
		needsRequestID, needsTraceID bool
	)

	// 检查 request_id

	if id, ok := metainfo.GetPersistentValue(ctx, "request_id"); ok && id != "" {
		requestID = id
	} else {
		requestID = uuid.New().String()
		needsRequestID = true
	}

	// 检查 trace_id
	if id, ok := metainfo.GetPersistentValue(ctx, "trace_id"); ok && id != "" {
		traceID = id
	} else {
		traceID = requestID // 使用 request_id 作为 trace_id
		needsTraceID = true
	}

	// 只有需要时才注入（性能优化）
	if needsRequestID {
		ctx = metainfo.WithPersistentValue(ctx, "request_id", requestID)
		m.logger.Warn().
			Str("request_id", requestID).
			Str("service", "identity_srv").
			Msg("Generated missing request_id")
	}

	if needsTraceID {
		ctx = metainfo.WithPersistentValue(ctx, "trace_id", traceID)
	}

	return ctx
}

// logTraceInfo 记录追踪信息日志
func (m *MetaInfoMiddleware) logTraceInfo(ctx context.Context) {
	attrs := LoggingAttrs(ctx)
	if len(attrs) > 0 {
		event := m.logger.Info().Str("middleware", "trace")
		for k, v := range attrs {
			event = event.Interface(k, v)
		}

		event.Msg("RPC request received")
	}
}

// Context 访问辅助函数，供业务逻辑使用

// GetRequestID 从 RPC 上下文获取 RequestID
// 直接从 metainfo 读取，不使用 context.Value
func GetRequestID(ctx context.Context) string {
	if id, ok := metainfo.GetPersistentValue(ctx, "request_id"); ok {
		return id
	}

	return ""
}

// GetTraceID 从 RPC 上下文获取 TraceID
// 直接从 metainfo 读取，不使用 context.Value
func GetTraceID(ctx context.Context) string {
	if id, ok := metainfo.GetPersistentValue(ctx, "trace_id"); ok {
		return id
	}

	return ""
}

// LoggingAttrs 返回用于结构化日志的属性
// 返回 map[string]interface{} 用于 zerolog
func LoggingAttrs(ctx context.Context) map[string]interface{} {
	attrs := make(map[string]interface{})

	if requestID := GetRequestID(ctx); requestID != "" {
		attrs["request_id"] = requestID
	}

	if traceID := GetTraceID(ctx); traceID != "" {
		attrs["trace_id"] = traceID
	}

	return attrs
}
