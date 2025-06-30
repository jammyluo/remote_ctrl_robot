#!/bin/bash

# 远程控制机器人服务器启动脚本

echo "🤖 启动远程控制机器人服务器..."

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "❌ 错误: Go未安装，请先安装Go 1.21+"
    exit 1
fi

# 检查Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ $(echo "$GO_VERSION 1.21" | tr " " "\n" | sort -V | head -n 1) != "1.21" ]]; then
    echo "❌ 错误: Go版本过低，需要1.21+，当前版本: $GO_VERSION"
    exit 1
fi

echo "✅ Go版本检查通过: $GO_VERSION"

# 安装依赖
echo "📦 安装Go依赖..."
go mod tidy

if [ $? -ne 0 ]; then
    echo "❌ 依赖安装失败"
    exit 1
fi

echo "✅ 依赖安装完成"

# 编译项目
echo "🔨 编译项目..."
go build -o robot-control-server main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译完成"

# 设置环境变量
export PORT=${PORT:-8080}
export LOG_LEVEL=${LOG_LEVEL:-info}

echo "🚀 启动服务器..."
echo "   端口: $PORT"
echo "   日志级别: $LOG_LEVEL"
echo "   访问地址: http://localhost:$PORT"
echo "   WebSocket地址: ws://localhost:$PORT/ws/control"
echo "   测试客户端: http://localhost:$PORT/test_client.html"
echo ""
echo "按 Ctrl+C 停止服务器"
echo ""

# 启动服务器
./robot-control-server 