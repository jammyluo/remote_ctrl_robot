#!/bin/bash

# 极简机器人客户端构建脚本

set -e

echo "🚀 开始构建极简机器人客户端..."

# 清理旧的构建文件
echo "🧹 清理旧文件..."
rm -f robot-client
rm -f robot-client-*

# 更新依赖
echo "📦 更新依赖..."
go mod tidy

# 运行测试
echo "🧪 运行测试..."
go test -v ./...

# 构建主程序
echo "🔨 构建主程序..."
go build -o robot-client .

# 检查构建结果
if [ -f "robot-client" ]; then
    echo "✅ 构建成功: robot-client"
    ls -lh robot-client
else
    echo "❌ 构建失败"
    exit 1
fi

echo "🎉 构建完成！"
echo ""
echo "使用方法:"
echo "  ./robot-client -config config.yaml"
echo "  或者使用默认配置:"
echo "  ./robot-client"
echo ""
echo "配置文件说明:"
echo "  - config.yaml: 包含机器人、服务器、日志等所有配置"
echo "  - 支持环境变量覆盖配置"
echo "  - 支持命令行参数指定配置文件路径" 