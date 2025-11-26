#!/bin/bash

# 本地 CI 检查脚本
# 运行与 GitHub Actions CI 相同的检查

set -e  # 遇到错误立即退出

echo "=========================================="
echo "Running local CI checks..."
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查计数器
ERRORS=0

# 1. Go mod 检查
echo ""
echo "1. Checking go mod..."
if go mod download && go mod verify; then
    echo -e "${GREEN}✓ go mod check passed${NC}"
else
    echo -e "${RED}✗ go mod check failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 2. Go vet 检查
echo ""
echo "2. Running go vet..."
if go vet ./...; then
    echo -e "${GREEN}✓ go vet passed${NC}"
else
    echo -e "${RED}✗ go vet failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 3. Go fmt 检查
echo ""
echo "3. Checking code formatting..."
UNFORMATTED=$(gofmt -s -l .)
if [ -z "$UNFORMATTED" ]; then
    echo -e "${GREEN}✓ code formatting check passed${NC}"
else
    echo -e "${RED}✗ code formatting check failed${NC}"
    echo "The following files are not formatted:"
    echo "$UNFORMATTED"
    echo ""
    echo "Run 'go fmt ./...' to fix"
    ERRORS=$((ERRORS + 1))
fi

# 4. Go test
echo ""
echo "4. Running tests..."
if go test -v ./...; then
    echo -e "${GREEN}✓ tests passed${NC}"
else
    echo -e "${YELLOW}⚠ tests failed (continuing...)${NC}"
    # 注意：CI 中 continue-on-error: true，所以这里不增加错误计数
fi

# 5. Go build
echo ""
echo "5. Building..."
if go build -v ./...; then
    echo -e "${GREEN}✓ build passed${NC}"
else
    echo -e "${RED}✗ build failed${NC}"
    ERRORS=$((ERRORS + 1))
fi

# 6. golangci-lint (如果已安装)
echo ""
echo "6. Running golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    if golangci-lint run --timeout=5m; then
        echo -e "${GREEN}✓ golangci-lint passed${NC}"
    else
        echo -e "${RED}✗ golangci-lint failed${NC}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${YELLOW}⚠ golangci-lint not installed, skipping...${NC}"
    echo "  Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# 总结
echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}All checks passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}Found $ERRORS error(s)${NC}"
    exit 1
fi

