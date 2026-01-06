# 项目名称和目录
BIN_DIR=bin

.PHONY: all chat-zug crazy-caller runCZ runCC clean test test-crazy-caller install help

# 默认目标：显示帮助信息
.DEFAULT_GOAL := help

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  make all              - Build all binaries"
	@echo "  make crazy-caller      - Build crazy-caller binary"
	@echo "  make chat-zug         - Build chat-zug binary"
	@echo "  make runCC            - Run crazy-caller directly"
	@echo "  make runCZ            - Run chat-zug directly"
	@echo "  make test             - Run all tests"
	@echo "  make test-crazy-caller - Run crazy-caller tests"
	@echo "  make install          - Install binaries to GOPATH/bin"
	@echo "  make clean            - Remove build artifacts"

# 默认目标：编译所有模块
all: chat-zug crazy-caller

# 编译 chat-zug
chat-zug:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/chat-zug ./cmd/chat-zug
	@echo "Built: $(BIN_DIR)/chat-zug"

# 编译 crazy-caller
crazy-caller:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/crazy-caller ./cmd/crazy-caller
	@echo "Built: $(BIN_DIR)/crazy-caller"

# 运行 chat-zug
runCZ:
	go run ./cmd/chat-zug

# 运行 crazy-caller
runCC:
	go run ./cmd/crazy-caller

# 运行所有测试
test:
	go test ./...

# 运行 crazy-caller 测试
test-crazy-caller:
	go test -v ./cmd/crazy-caller

# 运行测试并显示覆盖率
test-coverage:
	go test -cover ./cmd/crazy-caller
	go test -coverprofile=coverage.out ./cmd/crazy-caller
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 安装到 GOPATH/bin
install: all
	go install ./cmd/crazy-caller
	go install ./cmd/chat-zug
	@echo "Installed to $(GOPATH)/bin"

# 清理编译产物
clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts"