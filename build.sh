#!/bin/bash

# 远程控制机器人编译脚本

echo "🔨 开始编译远程控制机器人服务端..."

# 显示Go版本
echo "📋 Go版本: $(go version)"

# 清理之前的构建
echo "🧹 清理之前的构建文件..."
rm -f server
rm -f robot-client
rm -f server-arm64
rm -f robot-client-arm64
rm -f server-arm
rm -f robot-client-arm

# 下载依赖
echo "📦 下载Go依赖..."
go mod tidy

# 编译x86_64版本
echo "🔨 编译x86_64版本服务端..."
go build -o server cmd/server/main.go

echo "🔨 构建x86_64版本客户端..."
go build -o robot-client cmd/client/main.go 

# 编译ARM64版本
echo "🔨 构建ARM64版本客户端..."
GOOS=linux GOARCH=arm64 go build -o robot-client-arm64 cmd/client/main.go

# 编译ARM32版本
echo "🔨 构建ARM32版本客户端..."
GOOS=linux GOARCH=arm go build -o robot-client-arm cmd/client/main.go

# 设置执行权限
chmod +x server
chmod +x robot-client
chmod +x server-arm64
chmod +x robot-client-arm64
chmod +x server-arm
chmod +x robot-client-arm

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "✅ 编译成功！"
    echo "📁 生成的文件:"
    echo "  - server (x86_64)"
    echo "  - robot-client (x86_64)"
    echo "  - server-arm64 (ARM64)"
    echo "  - robot-client-arm64 (ARM64)"
    echo "  - server-arm (ARM32)"
    echo "  - robot-client-arm (ARM32)"
    echo ""
    echo "🚀 运行命令:"
    echo "  x86_64: ./server 或 ./robot-client"
    echo "  ARM64:  ./server-arm64 或 ./robot-client-arm64"
    echo "  ARM32:  ./server-arm 或 ./robot-client-arm"
else
    echo "❌ 编译失败！"
    exit 1
fi 
