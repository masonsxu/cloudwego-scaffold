#!/bin/bash

# =============================================================================
# CloudWeGo Scaffold Services - 部署脚本
# =============================================================================
# 用途：简化 Docker Compose 部署操作
# 支持：本地开发和生产环境部署
#
# 快速开始：
#   ./deploy.sh dev up        # 启动开发环境
#   ./deploy.sh logs          # 查看日志
#   ./deploy.sh down          # 停止服务
#   ./deploy.sh --help        # 查看帮助
# =============================================================================

set -euo pipefail

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." &> /dev/null && pwd)"

# 默认配置
ENVIRONMENT="${ENVIRONMENT:-dev}"
CONTAINER_CMD="${CONTAINER_CMD:-docker}"
PROJECT_NAME="${PROJECT_NAME:-backend}"

# Compose 文件配置（根据环境自动选择）
COMPOSE_BASE="$SCRIPT_DIR/docker-compose.base.yml"
COMPOSE_DEV="$SCRIPT_DIR/docker-compose.dev.yml"
COMPOSE_PROD="$SCRIPT_DIR/docker-compose.prod.yml"

# 服务分组定义
BASE_SERVICES="postgres etcd rustfs"
APP_SERVICES="identity_srv gateway"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
${GREEN}CloudWeGo Scaffold Services - 部署脚本${NC}

${BLUE}用法：${NC}
    $(basename "$0") [环境] [操作] [选项]

${BLUE}环境（可选）：${NC}
    dev             开发环境（默认）
    prod            生产环境

${BLUE}操作：${NC}
    up              启动所有服务
    up-base         仅启动基础服务（postgres、etcd、rustfs）
    up-app          仅启动应用服务（需要基础服务已运行）
    down            停止所有服务
    restart         重启所有服务
    restart-app     仅重启应用服务（不重新构建）
    update-app      快速更新应用服务（推荐日常使用）
    rebuild-app     重新构建应用服务（不使用缓存）
    rebuild-base    重新构建基础服务（不使用缓存，会删除所有容器）
    build           构建所有镜像
    rebuild         重新构建镜像（不使用缓存）
    logs [service]  查看服务日志（可选指定服务名）
    follow <service> 实时跟踪服务日志（仅支持单个服务）
    ps              查看服务状态
    status          分组显示服务状态
    exec <service>  进入服务容器
    clean           清理所有容器和卷
    help            显示此帮助信息

${BLUE}选项：${NC}
    -d, --detach    后台运行（适用于 up、up-base、up-app、update-app、rebuild-app、rebuild-base）
    --no-build      启动时不构建镜像
    --build-only    仅构建镜像，不启动服务
    -f, --force     强制执行（用于 clean）
    --podman        使用 Podman 替代 Docker

${BLUE}示例：${NC}
    # 首次部署：启动所有服务（自动构建）
    $(basename "$0") dev up

    # 日常开发：快速更新应用代码（前台运行，可看到日志）
    $(basename "$0") dev update-app

    # 日常开发：快速更新应用代码（后台运行）
    $(basename "$0") dev update-app -d

    # 只重启应用服务（不重新构建）
    $(basename "$0") dev restart-app

    # 完全重新构建应用服务（后台运行）
    $(basename "$0") dev rebuild-app -d

    # 重新构建基础服务（会删除所有容器并重新构建基础服务）
    $(basename "$0") dev rebuild-base -d

    # 只构建镜像，不启动服务
    $(basename "$0") dev up --build-only

    # 只构建应用服务镜像
    $(basename "$0") dev up-app --build-only

    # 启动生产环境（后台运行）
    $(basename "$0") prod up -d

    # 只启动基础服务
    $(basename "$0") dev up-base

    # 只启动应用服务（假设基础服务已运行）
    $(basename "$0") dev up-app -d

    # 查看 API Gateway 日志
    $(basename "$0") logs api_gateway

    # 实时跟踪 API Gateway 日志
    $(basename "$0") follow api_gateway

    # 分组查看服务状态
    $(basename "$0") status

    # 进入 identity_srv 容器
    $(basename "$0") exec identity_srv

    # 清理环境
    $(basename "$0") clean -f

${BLUE}服务列表：${NC}
    - postgres          PostgreSQL 数据库
    - etcd              服务注册与发现
    - rustfs            S3 兼容对象存储
    - identity_srv      身份认证服务
    - permission_srv    权限管理服务
    - api_gateway       API 网关

${BLUE}环境变量：${NC}
    ENVIRONMENT         部署环境 (dev|prod)
    CONTAINER_CMD       容器命令 (docker|podman)
    PROJECT_NAME        项目名称（默认: cloudwego-scafflod-backend）

${YELLOW}注意事项：${NC}
    1. 首次运行前请复制 .env.example 为 .env 并修改配置
    2. 生产环境请务必修改默认密码和密钥
    3. 使用 './deploy.sh logs' 可查看所有服务的实时日志
    4. 使用 './deploy.sh ps' 可查看所有服务的运行状态
EOF
}

# 检查依赖
check_dependencies() {
    if ! command -v "${CONTAINER_CMD}" &> /dev/null; then
        log_error "${CONTAINER_CMD} 未安装，请先安装 Docker 或 Podman"
        exit 1
    fi

    if [ "${CONTAINER_CMD}" = "docker" ]; then
        if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null 2>&1; then
            log_error "Docker Compose 未安装"
            exit 1
        fi

        # 优先使用 docker compose 而不是 docker-compose
        if docker compose version &> /dev/null 2>&1; then
            COMPOSE_CMD="docker compose -p ${PROJECT_NAME}"
        else
            COMPOSE_CMD="docker-compose -p ${PROJECT_NAME}"
        fi
    else
        if ! command -v podman-compose &> /dev/null; then
            log_error "Podman Compose 未安装"
            exit 1
        fi
        COMPOSE_CMD="podman-compose -p ${PROJECT_NAME}"
    fi
}

# 获取 Compose 文件组合
get_compose_files() {
    if [ "$ENVIRONMENT" = "prod" ]; then
        echo "-f $COMPOSE_BASE -f $COMPOSE_PROD"
    else
        echo "-f $COMPOSE_BASE -f $COMPOSE_DEV"
    fi
}

# 检查 .env 文件
check_env_file() {
    local env_example
    if [ "$ENVIRONMENT" = "prod" ]; then
        env_example="$SCRIPT_DIR/.env.prod.example"
    else
        env_example="$SCRIPT_DIR/.env.dev.example"
    fi

    if [ ! -f "$SCRIPT_DIR/.env" ]; then
        log_warn ".env 文件不存在"
        log_info "正在从 $(basename "$env_example") 创建 .env 文件..."
        cp "$env_example" "$SCRIPT_DIR/.env"
        log_info ".env 文件已创建，请根据需要修改配置"

        if [ "$ENVIRONMENT" = "prod" ]; then
            log_warn "⚠️  生产环境必须修改 .env 文件中的所有 CHANGE_ME 值！"
        fi
        echo
    fi
}

# 构建镜像
build_images() {
    local no_cache=""
    local compose_files=$(get_compose_files)

    if [ "${1:-}" = "--no-cache" ]; then
        no_cache="--no-cache"
        log_info "构建镜像（不使用缓存）..."
    else
        log_info "构建镜像..."
    fi

    cd "$SCRIPT_DIR"
    $COMPOSE_CMD $compose_files build $no_cache
    log_info "镜像构建完成"
}

# 启动服务
start_services() {
    local detach=""
    local build="--build"
    local build_only=false
    local services=""
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
            --no-build)
                build=""
                ;;
            --build-only)
                build_only=true
                ;;
            --app-only)
                services="$APP_SERVICES"
                ;;
            --base-only)
                services="$BASE_SERVICES"
                ;;
        esac
    done

    cd "$SCRIPT_DIR"

    # 如果仅构建镜像，不启动服务
    if [ "$build_only" = true ]; then
        log_info "仅构建镜像（环境: $ENVIRONMENT）..."
        $COMPOSE_CMD $compose_files build $services
        log_info "镜像构建完成"
        return
    fi

    log_info "启动服务（环境: $ENVIRONMENT）..."

    if [ -n "$build" ]; then
        log_info "检查并构建镜像..."
    fi

    $COMPOSE_CMD $compose_files up $detach $build $services

    if [ -n "$detach" ]; then
        echo
        show_service_info
    fi
}

# 停止服务
stop_services() {
    local compose_files=$(get_compose_files)
    cd "$SCRIPT_DIR"

    # 检查是否有运行的容器
    local running_containers
    running_containers=$($COMPOSE_CMD $compose_files ps -q 2>/dev/null || true)

    if [ -z "$running_containers" ]; then
        log_info "没有运行的容器，无需停止"
        return 0
    fi

    log_info "停止服务..."
    # 添加错误容忍，避免在资源不存在时报错
    $COMPOSE_CMD $compose_files down 2>&1 | grep -v "no container\|no pod\|not found" || true

    log_info "服务已停止"
}

# 重启服务
restart_services() {
    local compose_files=$(get_compose_files)
    log_info "重启服务..."
    cd "$SCRIPT_DIR"
    $COMPOSE_CMD $compose_files restart

    echo
    log_info "服务已重启"
    $COMPOSE_CMD $compose_files ps
}

# 查看日志
show_logs() {
    local service="${1:-}"
    local compose_files=$(get_compose_files)
    cd "$SCRIPT_DIR"

    if [ -n "$service" ]; then
        log_info "查看 $service 服务日志（最近 100 条）..."
        $COMPOSE_CMD $compose_files logs --tail=100 "$service"
    else
        log_info "查看所有服务日志（最近 50 条）..."
        $COMPOSE_CMD $compose_files logs --tail=50
    fi
}

# 实时跟踪日志（使用原生容器命令）
follow_logs() {
    local service="${1:-}"

    if [ -z "$service" ]; then
        log_error "请指定服务名称"
        echo "可用服务: identity_srv, api_gateway, postgres, etcd, rustfs"
        exit 1
    fi

    local container_name="cloudwego-scafflod-${service//_/-}"

    # 检查容器是否存在且运行中
    if ! ${CONTAINER_CMD} ps --filter "name=${container_name}" --filter "status=running" --format "{{.Names}}" | grep -q "${container_name}"; then
        log_error "服务 $service 未运行或容器不存在"
        exit 1
    fi

    log_info "实时跟踪 $service 服务日志（按 Ctrl+C 退出）..."
    ${CONTAINER_CMD} logs -f "${container_name}"
}

# 查看状态
show_status() {
    local compose_files=$(get_compose_files)
    log_info "CloudWeGo Scaffold 服务状态:"
    cd "$SCRIPT_DIR"
    $COMPOSE_CMD $compose_files ps
}

# 进入容器
exec_container() {
    local service="${1:-}"
    local compose_files=$(get_compose_files)

    if [ -z "$service" ]; then
        log_error "请指定服务名称"
        echo "可用服务: identity_srv, api_gateway, postgres, etcd, rustfs"
        exit 1
    fi

    log_info "进入 $service 容器..."
    cd "$SCRIPT_DIR"
    $COMPOSE_CMD $compose_files exec "$service" /bin/sh
}

# 清理环境
clean_environment() {
    local force="${1:-}"
    local compose_files=$(get_compose_files)

    if [ "$force" != "-f" ] && [ "$force" != "--force" ]; then
        echo -e "${YELLOW}警告：此操作将删除所有容器、卷和数据！${NC}"
        read -p "确定要继续吗？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
    fi

    log_warn "清理环境..."
    cd "$SCRIPT_DIR"

    # 停止并移除容器和卷
    # 添加错误容忍，避免在资源不存在时报错
    $COMPOSE_CMD $compose_files down -v --remove-orphans 2>&1 | grep -v "no container\|no pod\|not found" || true

    log_info "环境清理完成"
}

# 启动基础服务
start_base_services() {
    local detach=""
    local build="--build"
    local build_only=false
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
            --no-build)
                build=""
                ;;
            --build-only)
                build_only=true
                ;;
        esac
    done

    cd "$SCRIPT_DIR"

    # 如果仅构建镜像，不启动服务
    if [ "$build_only" = true ]; then
        log_info "仅构建基础服务镜像（环境: $ENVIRONMENT）..."
        $COMPOSE_CMD $compose_files build $BASE_SERVICES
        log_info "基础服务镜像构建完成"
        return
    fi

    log_info "启动基础服务（环境: $ENVIRONMENT）..."

    if [ -n "$build" ]; then
        log_info "检查基础服务镜像..."
    fi

    $COMPOSE_CMD $compose_files up $detach $build $BASE_SERVICES

    if [ -n "$detach" ]; then
        echo
        log_info "基础服务已启动"
        $COMPOSE_CMD $compose_files ps
    fi
}

# 启动应用服务
start_app_services() {
    local detach=""
    local build="--build"
    local build_only=false
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
            --no-build)
                build=""
                ;;
            --build-only)
                build_only=true
                ;;
        esac
    done

    cd "$SCRIPT_DIR"

    # 如果仅构建镜像，不启动服务
    if [ "$build_only" = true ]; then
        log_info "仅构建应用服务镜像（环境: $ENVIRONMENT）..."
        $COMPOSE_CMD $compose_files build $APP_SERVICES
        log_info "应用服务镜像构建完成"
        return
    fi

    log_info "启动应用服务（环境: $ENVIRONMENT）..."

    # 检查基础服务是否运行
    log_debug "检查基础服务状态..."
    local base_running=true
    for service in $BASE_SERVICES; do
        # 使用容器名称检查（兼容 docker 和 podman）
        if ! ${CONTAINER_CMD} ps --filter "name=cloudwego-scafflod-${service}" --filter "status=running" --format "{{.Names}}" | grep -q "cloudwego-scafflod-${service}"; then
            log_warn "基础服务 $service 未运行"
            base_running=false
        fi
    done

    if [ "$base_running" = false ]; then
        log_warn "部分基础服务未运行，建议先执行: $(basename "$0") $ENVIRONMENT up-base"
        read -p "是否继续启动应用服务？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
    fi

    if [ -n "$build" ]; then
        log_info "检查并构建应用镜像..."
    fi

    $COMPOSE_CMD $compose_files up $detach $build $APP_SERVICES

    if [ -n "$detach" ]; then
        echo
        log_info "应用服务已启动"
        $COMPOSE_CMD $compose_files ps
    fi
}

# 重启应用服务
restart_app_services() {
    local compose_files=$(get_compose_files)
    log_info "重启应用服务..."
    cd "$SCRIPT_DIR"
    $COMPOSE_CMD $compose_files restart $APP_SERVICES

    echo
    log_info "应用服务已重启"
    $COMPOSE_CMD $compose_files ps
}

# 重新构建应用服务
rebuild_app_services() {
    local detach=""
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
        esac
    done

    log_info "重新构建应用服务（不使用缓存）..."
    cd "$SCRIPT_DIR"

    $COMPOSE_CMD $compose_files build --no-cache $APP_SERVICES
    log_info "重新启动应用服务..."
    $COMPOSE_CMD $compose_files up $detach $APP_SERVICES

    if [ -n "$detach" ]; then
        echo
        log_info "应用服务已重新构建并启动"
        $COMPOSE_CMD $compose_files ps
    fi
}

# 重新构建基础服务
rebuild_base_services() {
    local detach=""
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
        esac
    done

    log_warn "重新构建基础服务将先停止并删除所有容器（包括应用服务）"
    log_info "停止并删除所有容器..."
    cd "$SCRIPT_DIR"

    # 使用 down 命令停止并删除所有容器（兼容 docker-compose 和 podman-compose）
    $COMPOSE_CMD $compose_files down 2>&1 | grep -v "no container\|no pod\|not found" || true

    log_info "重新构建基础服务（不使用缓存）..."
    $COMPOSE_CMD $compose_files build --no-cache $BASE_SERVICES

    log_info "启动基础服务..."
    $COMPOSE_CMD $compose_files up $detach $BASE_SERVICES

    if [ -n "$detach" ]; then
        echo
        log_info "基础服务已重新构建并启动"
        $COMPOSE_CMD $compose_files ps
        echo
        log_warn "提示：应用服务容器已删除，如需启动请运行："
        echo "    $(basename "$0") $ENVIRONMENT up-app -d"
    fi
}

# 快速更新应用服务
update_app_services() {
    local detach=""
    local compose_files=$(get_compose_files)

    for arg in "$@"; do
        case $arg in
            -d|--detach)
                detach="-d"
                ;;
        esac
    done

    log_info "快速更新应用服务..."
    cd "$SCRIPT_DIR"

    # 停止应用服务
    log_info "停止应用服务..."
    $COMPOSE_CMD $compose_files stop $APP_SERVICES > /dev/null 2>&1

    # 构建镜像
    log_info "构建应用镜像..."
    $COMPOSE_CMD $compose_files build $APP_SERVICES

    # 启动应用服务
    log_info "启动应用服务..."
    $COMPOSE_CMD $compose_files up $detach $APP_SERVICES

    if [ -n "$detach" ]; then
        echo
        log_info "应用服务更新完成"
        echo
        show_service_info
    fi
}

# 分组显示服务状态
status_by_group() {
    local compose_files=$(get_compose_files)
    cd "$SCRIPT_DIR"

    echo -e "${GREEN}=== 基础服务状态 ===${NC}"
    $COMPOSE_CMD $compose_files ps | grep -E "(NAME|postgres|etcd|rustfs)" | grep -v "identity\|gateway"

    echo
    echo -e "${GREEN}=== 应用服务状态 ===${NC}"
    $COMPOSE_CMD $compose_files ps | grep -E "(NAME|identity|gateway)"
}

# 显示服务访问信息
show_service_info() {
    log_info "=== 服务访问地址 ==="
    echo -e "${GREEN}API Gateway:${NC}     http://localhost:8080"
    echo -e "${GREEN}Identity Service:${NC} http://localhost:8891 (RPC), http://localhost:10000 (Health)"
    echo
    echo -e "${GREEN}基础设施:${NC}"
    echo -e "${GREEN}PostgreSQL:${NC}      localhost:5432"
    echo -e "${GREEN}etcd:${NC}            localhost:2379"
    echo -e "${GREEN}RustFS API:${NC}      http://localhost:9000"
    echo -e "${GREEN}RustFS Console:${NC}  http://localhost:9001"
    echo
    log_info "使用 '$(basename "$0") logs' 查看服务日志"
    log_info "使用 '$(basename "$0") ps' 查看服务状态"
}

# 解析参数
parse_args() {
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi

    # 检查第一个参数是否是环境
    if [ "$1" = "dev" ] || [ "$1" = "prod" ]; then
        ENVIRONMENT="$1"
        shift
    fi

    # 检查是否使用 Podman
    for arg in "$@"; do
        if [ "$arg" = "--podman" ]; then
            CONTAINER_CMD="podman"
        fi
    done

    check_dependencies
    check_env_file

    # 解析操作
    case "${1:-help}" in
        up)
            shift
            start_services "$@"
            ;;
        up-base)
            shift
            start_base_services "$@"
            ;;
        up-app)
            shift
            start_app_services "$@"
            ;;
        down)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        restart-app)
            restart_app_services
            ;;
        update-app)
            shift
            update_app_services "$@"
            ;;
        rebuild-app)
            shift
            rebuild_app_services "$@"
            ;;
        rebuild-base)
            shift
            rebuild_base_services "$@"
            ;;
        build)
            build_images
            ;;
        rebuild)
            build_images --no-cache
            ;;
        logs)
            shift
            show_logs "$@"
            ;;
        follow)
            shift
            follow_logs "$@"
            ;;
        ps)
            show_status
            ;;
        status)
            status_by_group
            ;;
        exec)
            shift
            exec_container "$@"
            ;;
        clean)
            shift
            clean_environment "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知操作: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

# 主函数
main() {
    cd "$SCRIPT_DIR"
    parse_args "$@"
}

# 捕获信号
trap 'log_error "脚本被中断"; exit 1' INT TERM

# 执行主函数
main "$@"
