// Package wire 服务容器定义
package wire

import (
	identityService "github.com/masonsxu/cloudwego-scaffold/gateway/internal/domain/service/identity"
	permissionService "github.com/masonsxu/cloudwego-scaffold/gateway/internal/domain/service/permission"
)

// ServiceContainer 服务容器
// 统一管理所有业务服务实例
type ServiceContainer struct {
	IdentityService   identityService.Service
	PermissionService permissionService.Service
}

// NewServiceContainer 创建服务容器
func NewServiceContainer(
	identityService identityService.Service,
	permissionService permissionService.Service,
) *ServiceContainer {
	return &ServiceContainer{
		IdentityService:   identityService,
		PermissionService: permissionService,
	}
}
