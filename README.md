# 远程控制机器人服务端

基于Golang和Janus WebRTC的远程控制机器人服务端，支持多机器人管理和低延迟控制。

## 功能特性

- 🎥 **WebRTC视频流**: 基于Janus的实时视频传输
- 🤖 **多机器人管理**: 支持通过UCODE标识多个机器人
- ⚡ **低延迟控制**: WebSocket实时控制指令传输
- 🔒 **安全认证**: 机器人注册和UCODE验证
- 📊 **状态监控**: 实时机器人状态和系统监控
- 🚀 **高性能**: 基于Golang的高并发处理

## 快速开始

### 1. 启动服务

```bash
# 启动服务器
go run cmd/server/main.go

# 或使用Docker
docker-compose up -d
```

### 2. 机器人连接

机器人需要通过WebSocket连接并注册UCODE：

```javascript
// 连接WebSocket
const ws = new WebSocket('ws://localhost:8080/ws/control');

// 连接后立即发送注册消息
ws.onopen = function() {
    ws.send(JSON.stringify({
        type: "register",
        ucode: "123456"  // 机器人唯一标识
    }));
};
```

### 3. API使用

所有API都需要指定机器人UCODE：

```bash
# 获取WebRTC播放地址
curl "http://localhost:8080/api/v1/webrtc/play-url?ucode=123456"

# 发送控制命令
curl -X POST "http://localhost:8080/api/v1/control/command?ucode=123456" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "joint_position",
    "command_id": "cmd_001",
    "priority": 5,
    "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
    "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
  }'

# 获取机器人状态
curl "http://localhost:8080/api/v1/control/status?ucode=123456"
```

## API接口

### 机器人相关接口（需要UCODE）

#### 1. 获取WebRTC播放地址
```
GET /api/v1/webrtc/play-url?ucode={UCODE}
```

#### 2. 发送控制命令
```
POST /api/v1/control/command?ucode={UCODE}
Content-Type: application/json

{
  "type": "joint_position|velocity|emergency_stop|home",
  "command_id": "unique_command_id",
  "priority": 1-10,
  "timestamp": 1640995200000,
  "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
  "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
}
```

#### 3. 获取机器人状态
```
GET /api/v1/control/status?ucode={UCODE}
```

### 系统接口（无需UCODE）

#### 4. 获取系统状态
```
GET /api/v1/system/status
```

#### 5. 获取连接状态
```
GET /api/v1/control/connection
```

#### 6. 健康检查
```
GET /health
```

## WebSocket接口

### 机器人注册
连接后第一条消息必须为注册消息：

```json
{
  "type": "register",
  "ucode": "123456"
}
```

### 控制命令
```json
{
  "type": "control_command",
  "data": {
    "type": "joint_position",
    "command_id": "cmd_001",
    "priority": 5,
    "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
    "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
  }
}
```

## 配置

配置文件：`config/config.yaml`

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"

janus:
  websocket_url: "ws://localhost:8188"
  http_url: "http://localhost:8088"
  stream_id: 1

robot:
  websocket_url: "ws://localhost:9090"

logging:
  level: "info"
  format: "json"

security:
  enable_cors: true
  allowed_origins: "*"
```

## 测试

### API测试
```bash
# 运行测试脚本
./test_api.sh
```

### WebSocket测试
打开 `test_client.html` 进行WebSocket连接测试。

## 部署

### Docker部署
```bash
# 构建镜像
docker build -t remote-ctrl-robot .

# 运行容器
docker run -p 8080:8080 remote-ctrl-robot
```

### 生产环境
1. 配置HTTPS证书
2. 设置防火墙规则
3. 配置负载均衡
4. 监控和日志收集

## 架构说明

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   客户端/浏览器   │    │   控制服务器     │    │   机器人设备     │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │WebRTC播放器  │◄────┤ │Janus WebRTC │ │    │ │WebSocket    │ │
│ └─────────────┘ │    │ │视频服务器    │ │    │ │客户端       │ │
│                 │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │                 │    │                 │
│ │WebSocket    │◄────┤ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │控制客户端    │ │    │ │Golang API   │ │    │ │机器人控制器  │ │
│ └─────────────┘ │    │ │服务器       │ │    │ └─────────────┘ │
└─────────────────┘    │ └─────────────┘ │    └─────────────────┘
                       └─────────────────┘
```

## 注意事项

1. **UCODE管理**: 每个机器人必须有唯一的UCODE
2. **连接顺序**: 机器人必须先连接WebSocket并注册UCODE
3. **错误处理**: API会返回详细的错误信息
4. **性能优化**: 支持多机器人并发控制
5. **安全考虑**: 生产环境应添加UCODE白名单验证

## 开发

### 项目结构
```
remote_ctrl_robot/
├── cmd/server/          # 主程序入口
├── internal/            # 内部包
│   ├── handlers/        # HTTP和WebSocket处理器
│   ├── models/          # 数据模型
│   └── services/        # 业务逻辑服务
├── config/              # 配置文件
├── test_client.html     # 测试客户端
└── test_api.sh         # API测试脚本
```

### 添加新功能
1. 在 `internal/models/` 中定义数据结构
2. 在 `internal/services/` 中实现业务逻辑
3. 在 `internal/handlers/` 中添加API接口
4. 更新测试脚本和文档

## 许可证

MIT License 