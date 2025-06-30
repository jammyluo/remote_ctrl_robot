# 🚀 快速开始指南

## 1. 环境准备

### 安装Go
```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# 验证安装
go version  # 应该显示 1.21+
```

### 安装Docker (可选)
```bash
# macOS
brew install --cask docker

# Ubuntu
sudo apt install docker.io docker-compose
```

## 2. 快速启动

### 方法一: 直接运行 (推荐)
```bash
# 1. 进入项目目录
cd remote_ctrl_robot

# 2. 运行启动脚本
./start.sh
```

### 方法二: Docker运行
```bash
# 1. 构建并启动所有服务
docker-compose up -d

# 2. 查看服务状态
docker-compose ps

# 3. 查看日志
docker-compose logs -f robot-control
```

## 3. 测试系统

### 测试API
```bash
# 运行API测试脚本
./test_api.sh
```

### 测试WebSocket
1. 打开浏览器访问: `http://localhost:8080/test_client.html`
2. 点击"连接"按钮
3. 尝试发送控制命令

### 手动测试API
```bash
# 健康检查
curl http://localhost:8080/health

# 获取WebRTC URL
curl http://localhost:8080/api/v1/webrtc/play-url

# 发送控制命令
curl -X POST http://localhost:8080/api/v1/control/command \
  -H "Content-Type: application/json" \
  -d '{
    "type": "joint_position",
    "joint_pos": [0.0, 0.0, 0.0, 0.0, 0.0, 0.0],
    "priority": 5
  }'
```

## 4. 与Janus集成

### 启动Janus服务器
```bash
# 如果使用Docker Compose，Janus会自动启动
# 否则手动启动Janus
docker run -d \
  --name janus \
  -p 8088:8088 \
  -p 8188:8188 \
  -p 8004:8004/udp \
  meetecho/janus-gateway:latest
```

### 推流到Janus
```bash
# 使用FFmpeg推送摄像头视频
ffmpeg -f v4l2 -i /dev/video0 \
  -vcodec libx264 \
  -preset ultrafast \
  -tune zerolatency \
  -f rtp rtp://127.0.0.1:8004
```

## 5. 开发调试

### 启用调试模式
```bash
export LOG_LEVEL=debug
go run main.go
```

### 查看实时日志
```bash
# 如果使用Docker
docker-compose logs -f robot-control

# 如果直接运行
tail -f logs/robot-control.log
```

### 性能监控
```bash
# 查看系统状态
curl http://localhost:8080/api/v1/system/status

# 查看连接状态
curl http://localhost:8080/api/v1/control/connection
```

## 6. 常见问题

### 端口被占用
```bash
# 查看端口占用
lsof -i :8080

# 杀死进程
kill -9 <PID>

# 或者使用不同端口
export PORT=8081
./start.sh
```

### WebSocket连接失败
1. 检查防火墙设置
2. 确认服务器正在运行
3. 检查浏览器控制台错误

### Janus连接失败
1. 确认Janus服务器正在运行
2. 检查端口是否正确
3. 查看Janus日志

## 7. 生产部署

### 使用Docker Compose
```bash
# 生产环境配置
docker-compose -f docker-compose.prod.yml up -d
```

### 使用系统服务
```bash
# 复制服务文件
sudo cp robot-control.service /etc/systemd/system/

# 启用服务
sudo systemctl enable robot-control
sudo systemctl start robot-control

# 查看状态
sudo systemctl status robot-control
```

## 8. 下一步

1. **集成真实机器人**: 修改 `robot_service.go` 连接真实硬件
2. **添加认证**: 实现API密钥或JWT认证
3. **优化性能**: 调整缓冲区大小和并发设置
4. **添加监控**: 集成Prometheus和Grafana
5. **扩展功能**: 添加更多控制模式和传感器支持

## 9. 获取帮助

- 📖 查看完整文档: `README.md`
- 🐛 报告问题: 创建GitHub Issue
- 💬 讨论: 加入项目Discord
- 📧 联系: 发送邮件到项目维护者

---

**🎉 恭喜！您已经成功启动了远程控制机器人系统！** 