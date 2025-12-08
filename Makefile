# 项目名称和目录
BIN_DIR=bin

.PHONY: all chat-zug crazy-caller runCZ runCC clean

# 默认目标：编译所有模块
all: chat-zug crazy-caller

# 编译 chat-zug
chat-zug:
	go build -o $(BIN_DIR)/chat-zug ./cmd/chat-zug

# 编译 crazy-caller
crazy-caller:
	go build -o $(BIN_DIR)/crazy-caller ./cmd/crazy-caller

# 运行 chat-zug
runCZ:
	go run ./cmd/chat-zug

# 运行 crazy-caller
runCC:
	go run ./cmd/crazy-caller

# 清理编译产物
clean:
	rm -rf $(BIN_DIR)