#!/bin/bash

# FireMail Docker构建脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
IMAGE_NAME="${IMAGE_NAME:-luofengyuan/firemailplus}"
VERSION="${VERSION:-latest}"
GO_BASE_IMAGE="${GO_BASE_IMAGE:-golang:1.24-alpine}"
NODE_BASE_IMAGE="${NODE_BASE_IMAGE:-node:20-alpine}"
DOCKER_BUILD_RETRIES="${DOCKER_BUILD_RETRIES:-3}"
DOCKER_BUILD_RETRY_DELAY="${DOCKER_BUILD_RETRY_DELAY:-10}"
DOCKER_BUILD_PULL="${DOCKER_BUILD_PULL:-true}"
DOCKER_BUILD_EXTRA_ARGS="${DOCKER_BUILD_EXTRA_ARGS:-}"

run_docker_build() {
    local build_cmd=(
        docker build
        --build-arg "GO_BASE_IMAGE=${GO_BASE_IMAGE}"
        --build-arg "NODE_BASE_IMAGE=${NODE_BASE_IMAGE}"
        -t "${IMAGE_NAME}:${VERSION}"
    )

    if [ "${DOCKER_BUILD_PULL}" = "true" ]; then
        build_cmd+=(--pull)
    fi

    if [ -n "${DOCKER_BUILD_EXTRA_ARGS}" ]; then
        # shellcheck disable=SC2206
        local extra_args=(${DOCKER_BUILD_EXTRA_ARGS})
        build_cmd+=("${extra_args[@]}")
    fi

    build_cmd+=(.)
    "${build_cmd[@]}"
}

echo -e "${GREEN}开始构建FireMail Docker镜像...${NC}"

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}错误: Docker未运行或无法访问${NC}"
    exit 1
fi

# 检查必要文件
if [ ! -f "Dockerfile" ]; then
    echo -e "${RED}错误: 未找到Dockerfile${NC}"
    exit 1
fi

if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}错误: 未找到docker-compose.yml${NC}"
    exit 1
fi

# 清理旧的构建缓存（可选）
echo -e "${YELLOW}清理Docker构建缓存...${NC}"
docker builder prune -f > /dev/null 2>&1 || true

# 构建镜像
echo -e "${GREEN}构建Docker镜像: ${IMAGE_NAME}:${VERSION}${NC}"
echo -e "${YELLOW}Go基础镜像: ${GO_BASE_IMAGE}${NC}"
echo -e "${YELLOW}Node基础镜像: ${NODE_BASE_IMAGE}${NC}"
echo -e "${YELLOW}构建重试次数: ${DOCKER_BUILD_RETRIES}, 初始退避: ${DOCKER_BUILD_RETRY_DELAY}s${NC}"

build_success=false
for attempt in $(seq 1 "${DOCKER_BUILD_RETRIES}"); do
    echo -e "${GREEN}Docker build 尝试 ${attempt}/${DOCKER_BUILD_RETRIES}${NC}"
    if run_docker_build; then
        build_success=true
        break
    fi

    if [ "${attempt}" -lt "${DOCKER_BUILD_RETRIES}" ]; then
        sleep_seconds=$((DOCKER_BUILD_RETRY_DELAY * attempt))
        echo -e "${YELLOW}构建失败，${sleep_seconds}s 后重试...${NC}"
        sleep "${sleep_seconds}"
    fi
done

# 检查构建结果
if [ "${build_success}" = "true" ]; then
    echo -e "${GREEN}✅ Docker镜像构建成功!${NC}"
    
    # 显示镜像信息
    echo -e "${YELLOW}镜像信息:${NC}"
    docker images ${IMAGE_NAME}:${VERSION}
    
    # 显示镜像大小
    IMAGE_SIZE=$(docker images ${IMAGE_NAME}:${VERSION} --format "{{.Size}}")
    echo -e "${GREEN}镜像大小: ${IMAGE_SIZE}${NC}"
    
else
    echo -e "${RED}❌ Docker镜像构建失败!${NC}"
    echo -e "${YELLOW}可通过 GO_BASE_IMAGE / NODE_BASE_IMAGE 指向内部镜像缓存后重试。${NC}"
    exit 1
fi

echo -e "${GREEN}构建完成! 可以使用以下命令运行:${NC}"
echo -e "${YELLOW}docker-compose up -d${NC}"
