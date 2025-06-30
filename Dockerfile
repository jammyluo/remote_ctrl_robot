# 多阶段构建
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o robot-control-server main.go

# 运行阶段
FROM alpine:latest

# 安装ca-certificates用于HTTPS
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN addgroup -g 1001 -S robot && \
    adduser -u 1001 -S robot -G robot

# 设置工作目录
WORKDIR /app

# 从builder阶段复制编译好的应用
COPY --from=builder /app/robot-control-server .

# 复制配置文件
COPY --from=builder /app/config ./config

# 复制测试客户端
COPY --from=builder /app/test_client.html .

# 更改文件所有者
RUN chown -R robot:robot /app

# 切换到非root用户
USER robot

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./robot-control-server"] 