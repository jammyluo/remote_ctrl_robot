# 极简机器人客户端

一个轻量级的机器人客户端，用于连接到远程控制系统，支持配置文件管理和自动重连。

## 🚀 特性

- **极简设计**: 双文件实现，易于理解和维护
- **配置文件支持**: 支持YAML配置文件管理所有设置
- **环境变量覆盖**: 支持环境变量覆盖配置文件
- **结构化日志**: 使用zerolog输出结构化日志
- **自动重连**: 智能重连机制，支持指数退避算法
- **状态模拟**: 模拟机器人状态上报
- **心跳保持**: 定期发送心跳消息

## 📁 项目结构

```
cmd/robot/
├── main.go          # 主程序（包含所有功能）
├── config.go        # 配置管理模块
├── config.yaml      # 配置文件
├── main_test.go     # 单元测试
├── build.sh         # 构建脚本
├── go.mod           # Go模块定义
└── README.md        # 说明文档
```

## 🛠️ 构建

```bash
# 构建可执行文件
./build.sh

# 或者手动构建
go mod tidy
go build -o robot-client main.go config.go
```

## 🚀 使用

### 配置文件方式（推荐）

```bash
# 使用默认配置文件
./robot-client

# 指定配置文件
./robot-client -config config.yaml

# 使用自定义配置文件
./robot-client -config my_config.yaml
```

### 环境变量覆盖

```bash
# 覆盖机器人配置
export ROBOT_UCODE=robot_002
export ROBOT_NAME=生产机器人
export ROBOT_VERSION=2.0.0

# 覆盖服务器配置
export ROBOT_SERVER_URL=ws://prod-server:8000/ws/control
export ROBOT_CONNECT_TIMEOUT=60

# 覆盖日志配置
export ROBOT_LOG_LEVEL=debug
export ROBOT_LOG_FORMAT=console

# 覆盖心跳和状态配置
export ROBOT_HEARTBEAT_INTERVAL=60
export ROBOT_STATUS_INTERVAL=30

# 覆盖重连配置
export ROBOT_RECONNECT_ENABLED=true
export ROBOT_MAX_RECONNECT_ATTEMPTS=10

# 运行客户端
./robot-client
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | `config.yaml` |

## ⚙️ 配置说明

### 配置文件结构

```yaml
# 机器人配置
robot:
  ucode: "robot_001"                    # 机器人唯一标识
  name: "测试机器人"                     # 机器人名称
  version: "1.0.0"                      # 版本号
  client_type: "robot"                  # 客户端类型

# 服务器配置
server:
  url: "ws://localhost:8000/ws/control" # WebSocket服务器地址
  connect_timeout: 30                   # 连接超时时间（秒）
  read_timeout: 30                      # 读取超时时间（秒）
  write_timeout: 10                     # 写入超时时间（秒）

# 心跳配置
heartbeat:
  interval: 30                          # 心跳间隔（秒）
  timeout: 10                           # 心跳超时（秒）

# 状态上报配置
status:
  interval: 10                          # 状态上报间隔（秒）
  enable_simulation: true               # 启用状态模拟

# 日志配置
logging:
  level: "info"                         # 日志级别 (debug, info, warn, error)
  format: "json"                        # 日志格式 (json, console)
  output: "stdout"                      # 输出目标 (stdout, stderr, file)
  file_path: ""                         # 日志文件路径（留空表示不输出到文件）
  max_size: 100                         # 最大文件大小（MB）
  max_backups: 3                        # 最大备份数
  max_age: 28                           # 最大保存天数

# 重连配置
reconnect:
  enabled: true                         # 启用自动重连
  max_attempts: 5                       # 最大重连次数
  initial_delay: 1                      # 初始重连延迟（秒）
  max_delay: 60                         # 最大重连延迟（秒）
  backoff_multiplier: 2.0               # 退避倍数

# 安全配置
security:
  enable_tls: false                     # 启用TLS
  skip_verify: false                    # 跳过证书验证
  auth_token: ""                        # 认证令牌

# 性能配置
performance:
  message_buffer_size: 1000             # 消息缓冲区大小
  worker_pool_size: 4                   # 工作池大小
  enable_metrics: false                 # 启用性能指标
```

### 环境变量映射

| 配置项 | 环境变量 | 说明 |
|--------|----------|------|
| `robot.ucode` | `ROBOT_UCODE` | 机器人唯一标识 |
| `robot.name` | `ROBOT_NAME` | 机器人名称 |
| `robot.version` | `ROBOT_VERSION` | 版本号 |
| `server.url` | `ROBOT_SERVER_URL` | 服务器地址 |
| `server.connect_timeout` | `ROBOT_CONNECT_TIMEOUT` | 连接超时 |
| `logging.level` | `ROBOT_LOG_LEVEL` | 日志级别 |
| `logging.format` | `ROBOT_LOG_FORMAT` | 日志格式 |
| `heartbeat.interval` | `ROBOT_HEARTBEAT_INTERVAL` | 心跳间隔 |
| `status.interval` | `ROBOT_STATUS_INTERVAL` | 状态间隔 |
| `reconnect.enabled` | `ROBOT_RECONNECT_ENABLED` | 启用重连 |
| `reconnect.max_attempts` | `ROBOT_MAX_RECONNECT_ATTEMPTS` | 最大重连次数 |

## 🔄 自动重连功能

### 重连机制

客户端具备智能重连机制，当连接断开时会自动尝试重连：

1. **连接检测**: 在消息读取、心跳发送、状态上报时检测连接状态
2. **自动重连**: 连接断开时自动安排重连
3. **指数退避**: 使用指数退避算法避免频繁重连
4. **随机抖动**: 添加随机抖动避免多个客户端同时重连
5. **最大重试**: 可配置最大重连次数，避免无限重试

### 重连算法

#### 指数退避算法
```
延迟 = 初始延迟 × (退避倍数 ^ 重连次数) + 随机抖动
```

#### 示例重连时间
- 第1次重连: 1-2秒
- 第2次重连: 2-4秒  
- 第3次重连: 4-8秒
- 第4次重连: 8-16秒
- 第5次重连: 16-32秒

### 重连配置

```yaml
reconnect:
  enabled: true              # 启用自动重连
  max_attempts: 5            # 最大重连次数
  initial_delay: 1           # 初始重连延迟（秒）
  max_delay: 60              # 最大重连延迟（秒）
  backoff_multiplier: 2.0    # 退避倍数
```

### 重连日志示例

```json
{"level":"error","ucode":"robot_001","error":"connection lost","time":"2024-01-01T12:00:00Z","message":"Read message error"}
{"level":"info","ucode":"robot_001","attempt":1,"max_attempts":5,"delay":"1.2s","time":"2024-01-01T12:00:00Z","message":"Scheduling reconnect"}
{"level":"info","ucode":"robot_001","attempt":1,"max_attempts":5,"time":"2024-01-01T12:00:01Z","message":"Attempting to reconnect"}
{"level":"info","ucode":"robot_001","attempt":1,"time":"2024-01-01T12:00:01Z","message":"Reconnect successful"}
```

## 📡 功能

### 1. 自动注册
连接成功后自动发送注册消息：
```json
{
  "type": "Request",
  "command": "CMD_REGISTER",
  "sequence": 1,
  "ucode": "robot_001",
  "client_type": "robot",
  "version": "1.0.0",
  "data": {}
}
```

### 2. 心跳保持
每30秒发送一次心跳消息：
```json
{
  "type": "Request",
  "command": "CMD_PING",
  "sequence": 2,
  "ucode": "robot_001",
  "client_type": "robot",
  "version": "1.0.0",
  "data": {}
}
```

### 3. 状态上报
每10秒上报一次机器人状态：
```json
{
  "type": "Request",
  "command": "CMD_UPDATE_ROBOT_STATUS",
  "sequence": 3,
  "ucode": "robot_001",
  "client_type": "robot",
  "version": "1.0.0",
  "data": {
    "base_position": [1.2, 3.4, 0],
    "base_orientation": [0, 0, 0, 1],
    "battery_level": 85.5,
    "temperature": 25.3,
    "status": "idle",
    "error_code": 0,
    "error_message": ""
  }
}
```

### 4. 自动重连
连接断开时自动重连：
- 使用指数退避算法
- 支持随机抖动
- 可配置最大重试次数
- 详细的重连日志

## 📊 日志系统

### 日志级别

- **debug**: 详细调试信息，包括心跳和状态更新
- **info**: 一般信息，包括连接、启动、停止、重连等
- **warn**: 警告信息
- **error**: 错误信息

### 日志格式

#### Info级别及以上（JSON格式）
```json
{"level":"info","ucode":"robot_001","server":"ws://localhost:8000/ws/control","time":"2024-01-01T12:00:00Z","message":"Start robot client"}
{"level":"info","ucode":"robot_001","time":"2024-01-01T12:00:01Z","message":"Connected successfully"}
{"level":"info","ucode":"robot_001","attempt":1,"max_attempts":5,"delay":"1.2s","time":"2024-01-01T12:00:02Z","message":"Scheduling reconnect"}
{"level":"error","ucode":"robot_001","error":"connection lost","time":"2024-01-01T12:00:03Z","message":"Read message error"}
```

#### Debug级别（控制台格式）
```
2024-01-01T12:00:00Z DBG 开始连接服务器 ucode=robot_001 server=ws://localhost:8000/ws/control
2024-01-01T12:00:01Z DBG 发送心跳 ucode=robot_001
2024-01-01T12:00:02Z DBG 发送状态更新 ucode=robot_001
```

### 日志字段

| 字段 | 说明 | 示例 |
|------|------|------|
| `level` | 日志级别 | "info", "debug", "error" |
| `time` | 时间戳 | "2024-01-01T12:00:00Z" |
| `message` | 日志消息 | "Start robot client" |
| `ucode` | 机器人标识 | "robot_001" |
| `server` | 服务器地址 | "ws://localhost:8000/ws/control" |
| `error` | 错误信息 | "connection lost" |
| `command` | 命令类型 | "CMD_PING" |
| `sequence` | 序列号 | 123 |
| `attempt` | 重连尝试次数 | 1 |
| `max_attempts` | 最大重连次数 | 5 |
| `delay` | 重连延迟 | "1.2s" |

## 🔧 开发

### 添加新功能

1. **添加新的命令类型**:
```go
const (
    CMD_TYPE_NEW_COMMAND CommandType = "CMD_NEW_COMMAND"
)
```

2. **添加新的消息处理方法**:
```go
func (r *RobotClient) sendNewCommand() error {
    msg := WebSocketMessage{
        Type:       WSMessageTypeRequest,
        Command:    CMD_TYPE_NEW_COMMAND,
        Sequence:   r.sequence,
        UCode:      r.config.Robot.UCode,
        ClientType: ClientTypeRobot,
        Version:    r.config.Robot.Version,
        Data:       map[string]interface{}{},
    }
    
    r.sequence++
    return r.conn.WriteJSON(msg)
}
```

### 测试

```bash
# 运行测试
go test -v ./...

# 运行并查看详细输出
go test -v -race ./...
```

## 🔄 与旧版本对比

| 特性 | 旧版本 | 极简版本 | 改进 |
|------|--------|----------|------|
| **文件数量** | 7个 | 5个 | -29% |
| **代码行数** | ~750行 | ~700行 | -7% |
| **依赖数量** | 3个 | 3个 | 0% |
| **配置复杂度** | 高 | 中等 | 显著简化 |
| **学习成本** | 高 | 低 | 显著降低 |
| **日志系统** | emoji日志 | 结构化日志 | 更专业 |
| **配置管理** | 环境变量 | 配置文件+环境变量 | 更灵活 |
| **重连机制** | 无 | 智能重连 | 新增功能 |

## 🎯 适用场景

- **快速原型**: 快速搭建机器人客户端
- **测试环境**: 用于系统测试和调试
- **学习示例**: 学习WebSocket客户端开发
- **轻量部署**: 资源受限的环境
- **生产环境**: 结构化日志便于监控和分析
- **配置管理**: 需要灵活配置管理的场景
- **高可用性**: 需要自动重连的场景

## 📝 注意事项

1. **状态模拟**: 当前版本使用随机数据模拟机器人状态
2. **错误处理**: 连接断开时会自动重连
3. **资源清理**: 程序退出时会自动清理资源
4. **并发安全**: 使用channel进行goroutine间通信
5. **日志级别**: 生产环境建议使用info级别，调试时使用debug级别
6. **配置文件**: 支持YAML格式，支持环境变量覆盖
7. **超时控制**: 支持连接、读取、写入超时配置
8. **重连策略**: 使用指数退避算法，避免频繁重连

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目！

## �� 许可证

MIT License 