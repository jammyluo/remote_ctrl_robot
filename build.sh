#!/bin/bash

# 远程控制机器人编译脚本

echo "🔨 开始编译远程控制机器人服务端..."

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go 1.21+"
    exit 1
fi

# 显示Go版本
echo "📋 Go版本: $(go version)"

# 清理之前的构建
echo "🧹 清理之前的构建文件..."
rm -f server
rm -f remote_ctrl_robot

# 下载依赖
echo "📦 下载Go依赖..."
go mod tidy

# 编译
echo "🔨 编译服务端..."
go build -o server cmd/server/main.go

# 检查编译结果
if [ $? -eq 0 ]; then
    echo "✅ 编译成功！"
    echo "📁 可执行文件: ./server"
    echo "🚀 运行命令: ./server"
else
    echo "❌ 编译失败！"
    exit 1
fi 