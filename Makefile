.PHONY: build run test clean dev

# 编译后端
build:
	go build -o bin/hfs-v2.exe ./src/cmd/hfs

# 运行后端
run:
	go run ./src/cmd/hfs

# 运行测试
test:
	go test ./src/...

# 清理编译产物
clean:
	rm -rf bin/*

# 开发模式：同时启动后端和前端
dev:
	@echo "Starting backend on :8080..."
	@go run ./src/cmd/hfs &
	@echo "Starting frontend dev server..."
	@cd web && npm run dev
