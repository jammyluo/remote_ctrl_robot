# 机器人远程控制系统架构

## 概述

本系统是一个支持多机器人管理的远程控制系统，采用模块化设计，每个机器人都有独立的服务实例，支持实时状态监控、命令控制和客户端管理。

## 核心组件

### 1. RobotManager（机器人管理器）
- **职责**: 管理所有机器人服务实例
- **功能**: 
  - 机器人注册/注销
  - 命令转发
  - 状态查询
  - 系统统计

### 2. RobotService（单个机器人服务）
- **职责**: 管理单个机器人的连接和状态
- **功能**:
  - 被动等待机器人WebSocket连接
  - 消息处理和状态同步
  - 命令发送
  - 事件管理

### 3. ClientManager（客户端管理器）
- **职责**: 管理操作员客户端连接
- **功能**:
  - 客户端连接管理
  - 消息路由
  - 状态监控

### 4. RobotRegistry（机器人注册表）
- **职责**: 机器人元数据管理
- **功能**:
  - 机器人信息存储
  - 状态跟踪
  - 配置管理

## 数据模型

### Robot（机器人实体）
```go
type Robot struct {
    UCode        string
    Name         string
    Type         RobotType
    Version      string
    IPAddress    string
    Port         int
    Capabilities []string
    Status       *RobotStatus
    service      RobotService
}
```

### RobotStatus（机器人状态）
```go
type RobotStatus struct {
    Online           bool
    LastHeartbeat    time.Time
    Latency          int64
    BatteryLevel     float64
    Temperature      float64
    Position         [3]float64
    Orientation      [4]float64
    ErrorCode        int
    ErrorMessage     string
    TotalCommands    int64
    FailedCommands   int64
    LastCommandTime  time.Time
}
```

### RobotCommand（机器人命令）
```go
type RobotCommand struct {
    Action        string
    Params        map[string]string
    Priority      int
    Timestamp     int64
    OperatorUCode string
}
```

## 通信流程

### 1. 机器人连接
```
机器人 → WebSocket连接 → 注册消息 → WebSocketHandlers → RobotManager → 创建RobotService → 设置连接 → 返回注册结果
```

### 2. 命令发送
```
操作员 → WebSocket消息 → ClientManager → RobotManager → RobotService → 机器人
```

### 3. 状态同步
```
机器人 → 状态更新 → RobotService → RobotRegistry → 通知相关客户端
```

## 配置管理

### robots.yaml 配置示例
```yaml
robots:
  - ucode: "robot_001"
    name: "机器人1"
    type: "b2"
    ip_address: "192.168.1.100"
    port: 8080
    capabilities: ["move", "stop", "reset"]
  
  - ucode: "robot_002"
    name: "机器人2"
    type: "b2"
    ip_address: "192.168.1.101"
    port: 8080
    capabilities: ["move", "stop", "reset"]
```

## API接口

### 机器人管理
- `POST /api/v1/robots/register` - 注册机器人
- `DELETE /api/v1/robots/{ucode}` - 注销机器人
- `GET /api/v1/robots` - 获取所有机器人
- `GET /api/v1/robots/{ucode}/status` - 获取机器人状态

### 命令控制
- `POST /api/v1/robots/{ucode}/command` - 发送命令
- `GET /api/v1/robots/{ucode}/statistics` - 获取统计信息

### 系统状态
- `GET /api/v1/system/status` - 获取系统状态
- `GET /api/v1/clients` - 获取客户端列表

## WebSocket接口

### 消息格式
```json
{
  "type": "Request",
  "command": "CMD_REGISTER",
  "sequence": 1234567890,
  "ucode": "robot_001",
  "client_type": "robot",
  "version": "1.0.0",
  "data": {}
}
```

### 支持的命令
- `CMD_REGISTER` - 注册
- `CMD_BIND_ROBOT` - 绑定机器人
- `CMD_CONTROL_ROBOT` - 控制机器人
- `CMD_UPDATE_ROBOT_STATUS` - 更新状态
- `CMD_PING` - 心跳

## 使用示例

### 启动服务
```bash
go run cmd/server/main.go
```

### 注册机器人
```go
registration := &models.RobotRegistration{
    UCode:        "robot_001",
    Name:         "测试机器人",
    Type:         models.RobotTypeB2,
    Version:      "1.0.0",
    IPAddress:    "192.168.1.100",
    Port:         8080,
    Capabilities: []string{"move", "stop", "reset"},
}

robot, err := robotManager.RegisterRobot(registration)
```

### 发送命令
```go
command := &models.RobotCommand{
    Action:        "move",
    Params:        map[string]string{"direction": "forward", "speed": "0.5"},
    Priority:      5,
    Timestamp:     time.Now().Unix(),
    OperatorUCode: "operator_001",
}

response, err := robotManager.SendCommand("robot_001", command)
```

## 优势特性

1. **多机器人支持**: 每个机器人独立管理，支持同时控制多个机器人
2. **模块化设计**: 清晰的职责分离，易于维护和扩展
3. **实时状态同步**: 机器人状态实时更新，支持状态监控
4. **自动重连**: 连接断开时自动重连，提高系统稳定性
5. **事件驱动**: 基于事件的架构，响应及时
6. **可扩展性**: 支持添加新的机器人类型和功能

## 部署说明

1. 确保Go环境已安装（版本1.19+）
2. 安装依赖: `go mod tidy`
3. 配置机器人信息: 编辑 `config/robots.yaml`
4. 启动服务: `go run cmd/server/main.go`
5. 访问Web界面: `http://localhost:8080`

## 监控和日志

系统使用zerolog进行日志记录，支持不同级别的日志输出：
- INFO: 正常操作日志
- WARN: 警告信息
- ERROR: 错误信息
- DEBUG: 调试信息

可以通过配置文件调整日志级别和格式。 