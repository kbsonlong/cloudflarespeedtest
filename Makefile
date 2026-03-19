.PHONY: all build run test clean lint fmt vet install docker help

# 项目配置
BINARY_NAME=cfst
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"

# Go 配置
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# 目录
CMD_DIR=./cmd/cfst
BUILD_DIR=./build
CONFIGS_DIR=./configs

all: fmt vet test build

## build: 编译项目
build:
	@echo "编译 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

## build-all: 交叉编译多平台版本
build-all:
	@echo "交叉编译..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/main.go
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go

## run: 运行程序
run: build
	@echo "运行 $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## test: 运行测试
test:
	@echo "运行测试..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## test-short: 运行快速测试
test-short:
	@echo "运行快速测试..."
	$(GOTEST) -short ./...

## bench: 运行基准测试
bench:
	@echo "运行基准测试..."
	$(GOTEST) -bench=. -benchmem ./...

## clean: 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## fmt: 格式化代码
fmt:
	@echo "格式化代码..."
	$(GOFMT) -s -w .

## vet: 代码静态检查
vet:
	@echo "运行 go vet..."
	$(GOVET) ./...

## lint: 代码质量检查
lint:
	@echo "运行 golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint 未安装，跳过"; \
	fi

## mod: 更新依赖
mod:
	@echo "整理依赖..."
	$(GOMOD) tidy
	$(GOMOD) verify

## install: 安装到本地
install:
	@echo "安装 $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) $(CMD_DIR)/main.go

## docker: 构建 Docker 镜像
docker:
	@echo "构建 Docker 镜像..."
	docker build -t cfst:$(VERSION) .

## docker-push: 推送 Docker 镜像
docker-push: docker
	docker tag cfst:$(VERSION) cfst:latest
	docker push cfst:$(VERSION)
	docker push cfst:latest

## deps: 下载依赖
deps:
	@echo "下载依赖..."
	$(GOGET) -v ./...

## update-deps: 更新依赖
update-deps:
	@echo "更新依赖..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

## config: 生成配置文件
config:
	@echo "生成配置文件..."
	@mkdir -p $(CONFIGS_DIR)
	@cp configs/config.example.yaml $(CONFIGS_DIR)/config.yaml

## tools: 安装开发工具
tools:
	@echo "安装开发工具..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/goreleaser/goreleaser

## release: 生成发布包
release:
	@echo "生成发布包..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release; \
	else \
		echo "goreleaser 未安装"; \
	fi

## snapshot: 生成快照包
snapshot:
	@echo "生成快照包..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser --snapshot; \
	else \
		echo "goreleaser 未安装"; \
	fi

## help: 显示帮助信息
help:
	@echo "可用命令:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | column -t -s ':'

.DEFAULT_GOAL := help
