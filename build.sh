#!/bin/bash

# 获取版本信息
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
LDFLAGS="-X github.com/chenq7an/gstor/cmd.Version=${VERSION} -X github.com/chenq7an/gstor/cmd.BuildTime=${BUILD_TIME} -X github.com/chenq7an/gstor/cmd.GitCommit=${GIT_COMMIT}"

# 构建 Linux amd64
echo "Building for linux/amd64..."
GOARCH=amd64 GOOS=linux go build -ldflags "${LDFLAGS}" -o .build/gstor

# 构建 Linux arm64
echo "Building for linux/arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o .build/gstor_arm64

echo "Build completed!"
echo "Version: ${VERSION}"
echo "Build Time: ${BUILD_TIME}"
echo "Git Commit: ${GIT_COMMIT}"
