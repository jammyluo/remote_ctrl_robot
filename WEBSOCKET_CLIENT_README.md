# WebSocket 客户端测试指南

## Python 客户端

### 安装依赖
```bash
pip install websocket-client
```

### 基本使用
```bash
# 使用默认UCODE (123456)
python test_websocket_client.py

# 指定UCODE
python test_websocket_client.py 789012

# 指定服务器地址和UCODE
python test_websocket_client.py 789012 ws://192.168.1.100:8080/ws/control
```

### 功能特性
- ✅ 自动注册UCODE
- ✅ 交互式菜单
- ✅ 自动心跳检测
- ✅ 错误处理和重连
- ✅ 支持多种控制命令

### 交互菜单
```
==================================================
交互菜单:
1. 发送关节位置控制命令
2. 发送速度控制命令
3. 发送紧急停止命令
4. 发送回零命令
5. 请求机器人状态
6. 发送ping
0. 退出
==================================================
```

## 消息格式

### 注册消息
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
    "command_id": "cmd_1234567890",
    "priority": 5,
    "timestamp": 1640995200000,
    "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
    "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
  }
}
```

### 状态请求
```json
{
  "type": "status_request",
  "message": "request robot status"
}
```

### Ping消息
```json
{
  "type": "ping",
  "message": "heartbeat"
}
```

## 服务器响应

### 欢迎消息
```json
{
  "type": "welcome",
  "message": "Connected as UCODE 123456",
  "data": {
    "timestamp": 1640995200000,
    "ucode": "123456"
  }
}
```

### 错误消息
```json
{
  "type": "error",
  "message": "Invalid UCODE",
  "data": {
    "error_type": "invalid_ucode",
    "timestamp": 1640995200000
  }
}
```

### Pong响应
```json
{
  "type": "pong",
  "message": "Pong",
  "data": {
    "timestamp": 1640995200000
  }
}
```

## 故障排除

### 连接失败
1. 确保服务器正在运行
2. 检查服务器地址和端口
3. 检查防火墙设置

### 注册失败
1. 确保UCODE格式正确
2. 检查服务器日志
3. 确认没有重复的UCODE

### 消息发送失败
1. 确保已成功注册
2. 检查消息格式
3. 确认连接状态

## 开发自定义客户端

### 基本连接流程
1. 建立WebSocket连接
2. 发送注册消息
3. 等待欢迎响应
4. 开始正常通信

### 错误处理
- 连接断开时自动重连
- 消息发送失败时重试
- 定期发送心跳保持连接

### 安全考虑
- 生产环境使用WSS (WebSocket Secure)
- 添加UCODE验证机制
- 实现消息加密 