#!/bin/bash

echo "启动机器人远程控制系统（新架构）"
echo "=================================="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go"
    exit 1
fi

# 检查配置文件
if [ ! -f "config/config.yaml" ]; then
    echo "警告: 未找到config.yaml配置文件，将使用默认配置"
fi

if [ ! -f "config/robots.yaml" ]; then
    echo "警告: 未找到robots.yaml配置文件，将使用默认机器人配置"
fi

# 安装依赖
echo "安装依赖..."
go mod tidy

# 编译
echo "编译程序..."
go build -o server cmd/server/main.go

if [ $? -eq 0 ]; then
    echo "编译成功！"
    echo ""
    echo "启动服务器..."
    echo "访问地址: http://localhost:8080"
    echo "WebSocket地址: ws://localhost:8080/ws/control"
    echo ""
    echo "按 Ctrl+C 停止服务"
    echo ""
    
    # 启动服务器
    ./server
else
    echo "编译失败！"
    exit 1
fi 