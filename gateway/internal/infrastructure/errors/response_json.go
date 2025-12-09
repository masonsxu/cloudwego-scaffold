// Package errors 提供了 Hertz 网关层的统一错误处理机制
// 本文件提供统一的 JSON 响应发送功能，自动填充追踪字段
package errors

import (
	"reflect"

	"github.com/cloudwego/hertz/pkg/app"
)

// JSON 发送 JSON 响应并自动填充时间戳字段
// 此函数会自动检测响应对象中的 BaseResp 字段并填充 timestamp
// 注意：RequestID 已从响应体中移除，改为通过 HTTP Header (X-Request-ID) 传递，由 requestid 中间件自动处理
//
// 使用示例:
//
//	resp, err := service.GetData(ctx, req)
//	if err != nil {
//	    errors.HandleServiceError(c, err, "获取数据失败")
//	    return
//	}
//	errors.JSON(c, consts.StatusOK, resp)
func JSON(c *app.RequestContext, code int, obj interface{}) {
	fillBaseRespReflection(c, obj)
	c.JSON(code, obj)
}

// fillBaseRespReflection 使用反射自动填充 BaseResp 字段
// 支持以下场景:
// 1. 直接的 BaseResponseDTO
// 2. 包含 BaseResp 字段的结构体
// 3. 字段为空时自动填充，已有值时不覆盖
// 注意：只填充时间戳，RequestID 已从响应体中移除，改为通过 HTTP Header 传递
func fillBaseRespReflection(c *app.RequestContext, obj interface{}) {
	if obj == nil {
		return
	}

	val := reflect.ValueOf(obj)

	// 处理指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}

		val = val.Elem()
	}

	// 只处理结构体
	if val.Kind() != reflect.Struct {
		return
	}

	// 查找 BaseResp 字段
	baseRespField := findBaseRespField(val)
	if !baseRespField.IsValid() {
		return
	}

	// BaseResp 字段必须是指针类型
	if baseRespField.Kind() != reflect.Ptr || baseRespField.IsNil() {
		return
	}

	// 获取 BaseResp 指向的实际对象
	baseRespValue := baseRespField.Elem()
	if baseRespValue.Kind() != reflect.Struct {
		return
	}
}

// findBaseRespField 查找 BaseResp 字段（支持多种命名）
// 支持的字段名：BaseResp, BaseResponse, base_resp
func findBaseRespField(val reflect.Value) reflect.Value {
	// 尝试常见的字段名
	fieldNames := []string{"BaseResp", "BaseResponse", "base_resp"}

	for _, name := range fieldNames {
		field := val.FieldByName(name)
		if field.IsValid() {
			return field
		}
	}

	return reflect.Value{}
}
