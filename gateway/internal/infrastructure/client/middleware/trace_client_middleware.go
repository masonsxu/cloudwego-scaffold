// Package middleware 提供RPC客户端中间件
// 本文件提供追踪信息传播中间件
package middleware

import (
	"context"

	"github.com/cloudwego/kitex/pkg/endpoint"
)

// TraceClientMiddleware Kitex 客户端追踪中间件
// 此中间件会自动从 context 的 metainfo 中读取追踪信息
// 并通过 TTHeader 传递到 RPC 服务端
//
// 使用方式:
//
//	client, err := service.NewClient(
//	    serviceName,
//	    client.WithMiddleware(middleware.TraceClientMiddleware()),
//	    ...
//	)
func TraceClientMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) error {
			// metainfo 中的追踪信息会自动通过 TTHeader 传递到 RPC 服务
			// 无需手动处理，Kitex 框架会自动传播 PersistentValue
			return next(ctx, req, resp)
		}
	}
}
