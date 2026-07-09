# Gita Makefile
# ============================================================================
# 使用方式：
#   make build          # 为当前平台编译
#   make build-all      # 为所有平台编译（linux/mac, amd64/arm64）
#   make build-linux    # 为 Linux amd64 编译
#   make build-mac      # 为 macOS amd64 编译
#   make build-mac-arm  # 为 macOS arm64（Apple Silicon）编译
#   make test           # 运行全部单元测试
#   make test-e2e       # 运行端到端集成测试
#   make clean          # 清理编译产物
#   make install        # 安装到 /usr/local/bin（需 sudo）
#   make lint           # 静态检查
# ============================================================================

# 编译参数
BINARY_NAME := gita
BUILD_DIR   := build
MODULE      := gita
LDFLAGS     := -s -w                                    # 去除符号表和调试信息，减小二进制体积
GCFLAGS     :=                                          # 可额外注入的 go build 参数

# 版本信息（通过 git describe 获取，若不在 git 仓库则使用默认值）
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# 注入版本信息的 ldflags
VERSION_LDFLAGS := -X '$(MODULE)/internal/version.Version=$(VERSION)' \
                   -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
                   -X '$(MODULE)/internal/version.BuildTime=$(BUILD_TIME)'

.PHONY: all build build-all build-linux build-mac build-mac-arm \
        test test-e2e clean install lint help

# ============================================================================
# 默认目标：编译当前平台
# ============================================================================
all: build

help: ## 显示帮助信息
	@echo "Gita Makefile 可用目标："
	@echo ""
	@echo "  make build          为当前平台编译"
	@echo "  make build-all      为所有平台编译（linux/mac, amd64/arm64）"
	@echo "  make build-linux    为 Linux amd64 编译"
	@echo "  make build-mac      为 macOS amd64（Intel）编译"
	@echo "  make build-mac-arm  为 macOS arm64（Apple Silicon）编译"
	@echo "  make test           运行全部单元测试"
	@echo "  make test-e2e       运行端到端集成测试"
	@echo "  make clean          清理编译产物"
	@echo "  make install        安装到 /usr/local/bin"
	@echo "  make lint           静态检查（go vet）"

# ============================================================================
# 编译
# ============================================================================

# build: 为当前平台编译
build:
	@echo "🔨 编译 $(BINARY_NAME)（当前平台）..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gita
	@echo "✅ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)"


# build-all: 为全部目标平台交叉编译
build-all: build-linux build-linux-arm build-mac build-mac-arm
	@echo "✅ 全平台编译完成，产物在 $(BUILD_DIR)/ 目录下："
	@ls -lh $(BUILD_DIR)/


# build-linux: 为 Linux amd64 编译
build-linux:
	@echo "🔨 编译 $(BINARY_NAME)（linux/amd64）..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gita
	@echo "✅ 完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"


# build-linux-arm: 为 Linux arm64 编译（AWS Graviton / 树莓派 64 位等）
build-linux-arm:
	@echo "🔨 编译 $(BINARY_NAME)（linux/arm64）..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/gita
	@echo "✅ 完成: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"


# build-mac: 为 macOS amd64（Intel 芯片）编译
build-mac:
	@echo "🔨 编译 $(BINARY_NAME)（darwin/amd64）..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		go build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gita
	@echo "✅ 完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"


# build-mac-arm: 为 macOS arm64（Apple Silicon: M1/M2/M3/M4）编译
build-mac-arm:
	@echo "🔨 编译 $(BINARY_NAME)（darwin/arm64）..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		go build -ldflags "$(LDFLAGS) $(VERSION_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gita
	@echo "✅ 完成: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"


# ============================================================================
# 测试
# ============================================================================

test: ## 运行全部单元测试
	@echo "🧪 运行单元测试..."
	go test ./... -count=1

test-e2e: ## 运行端到端集成测试
	@echo "🧪 运行 E2E 集成测试..."
	go test ./cmd/gita/ -v -run "E2E" -count=1 -timeout 30s


# ============================================================================
# 工具
# ============================================================================

install: build ## 编译并安装到 /usr/local/bin
	@echo "📦 安装 $(BINARY_NAME) 到 /usr/local/bin..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ 安装完成，运行 'gita --help' 验证"

lint: ## 静态检查
	@echo "🔍 运行 go vet..."
	go vet ./...

clean: ## 清理编译产物
	@echo "🧹 清理编译产物..."
	rm -rf $(BUILD_DIR)
	@echo "✅ 清理完成"
