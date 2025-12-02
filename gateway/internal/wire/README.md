# Wire 依赖注入最佳实践文档

本文档说明 API 网关项目中 Wire 依赖注入的最佳实践实现。

## 架构设计

### 分层结构

```
internal/wire/
├── wire.go              # 主要的Wire injector配置
├── wire_gen.go         # Wire自动生成的代码 (不要编辑)
├── infrastructure.go   # 基础设施层依赖注入
├── application.go      # 应用层依赖注入
├── domain.go          # 领域服务层依赖注入
├── middleware.go      # 中间件层依赖注入
├── container.go       # 服务容器定义
└── config.go          # 配置相关(向后兼容)
```

### 依赖注入层级

1. **基础设施层** (Infrastructure): 配置、日志、客户端连接
2. **应用层** (Application): 转换器、组装器
3. **领域服务层** (Domain): 业务逻辑服务
4. **中间件层** (Middleware): HTTP 中间件服务
5. **容器层** (Container): 统一的服务容器

## 核心文件说明

### wire.go - 主配置文件

```go
//go:build wireinject

// AllSet 组合所有依赖注入集合
var AllSet = wire.NewSet(
    InfrastructureSet,  // 基础设施层
    ApplicationSet,     // 应用层
    DomainServiceSet,   // 领域服务层
    MiddlewareSet,      // 中间件层
    NewServiceContainer, // 服务容器
)

// InitializeService 初始化服务容器
func InitializeService() (*ServiceContainer, error)

// InitializeMiddleware 初始化中间件容器
func InitializeMiddleware() (*MiddlewareContainer, error)
```

### container.go - 服务容器

```go
// ServiceContainer 统一管理所有业务服务实例
type ServiceContainer struct {
    AuthService         authservice.AuthService
    UserService         userservice.UserService
    TeamService         teamservice.TeamService
    OrganizationService orgservice.OrganizationService
    AdminService        adminservice.AdminService
}

// MiddlewareContainer 统一管理所有中间件实例
type MiddlewareContainer struct {
    JWTMiddleware     jwtmdw.JWTMiddlewareService
    ContextMiddleware contextmdw.ContextMiddlewareService
    CORSMiddleware    corsmdw.CORSMiddlewareService
}
```

### 各层 Provider 特点

#### infrastructure.go - 基础设施层

- 提供配置服务 (ProvideConfig)
- 提供日志服务 (ProvideLogger)
- 提供外部客户端 (ProvideIdentityClient)
- 提供 JWT 配置 (ProvideJWTConfig)

#### application.go - 应用层

- 提供各种转换器 (BaseConverter, AuthConverter, etc.)
- 负责 HTTP/RPC 数据转换

#### domain.go - 领域服务层

- 提供业务逻辑服务
- 依赖基础设施层和应用层

#### middleware.go - 中间件层

- 提供 HTTP 中间件服务
- 依赖基础设施层和领域服务层

## 使用方法

### 生成 Wire 代码

```bash
cd api/radius_api_gateway/internal/wire
wire
```

### 在 main.go 中使用

```go
import "github.com/masonsxu/cloudwego-scaffold/gateway/internal/wire"

func main() {
    // 初始化配置
    config := wire.NewConfig()

    // 初始化服务
    services, err := wire.InitializeService()
    if err != nil {
        log.Fatalf("failed to init services: %v", err)
    }

    // 初始化中间件
    middlewares, err := wire.InitializeMiddleware()
    if err != nil {
        log.Fatalf("failed to init middlewares: %v", err)
    }

    // 使用服务和中间件...
}
```

## 最佳实践规则

### 1. 分层清晰

- 每一层有明确的职责边界
- 高层依赖低层，避免循环依赖

### 2. 容器模式

- 使用容器统一管理同类型的服务
- 便于测试和生命周期管理

### 3. Provider 命名

- 统一使用 `Provide` 前缀
- 函数名清晰表达提供的服务

### 4. 错误处理

- 在 Provider 中进行错误检查
- 关键错误直接 panic，确保快速失败

### 5. 文档注释

- 为每个 Provider 添加清晰的注释
- 说明服务的用途和依赖关系

## 维护指导

### 添加新服务

1. 在对应层的文件中添加 Provider 函数
2. 将 Provider 加入对应的 Set 中
3. 如果需要，在容器中添加新字段
4. 运行 `wire` 重新生成代码

### 修改依赖关系

1. 修改 Provider 函数的参数
2. 运行 `wire` 检查依赖图
3. 解决任何循环依赖问题

### 性能优化

- Wire 在编译时解析依赖，运行时无反射开销
- 单例模式由 Wire 自动处理
- 避免在 Provider 中进行重复的昂贵操作
