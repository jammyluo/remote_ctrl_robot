# 极简机器人客户端

一个基于Go语言的极简WebSocket机器人客户端，采用分层架构设计，支持自动重连、结构化日志和配置管理。

## 项目结构

```
cmd/robot/
├── main.go              # 主程序入口和机器人业务逻辑
├── config.go            # 配置管理和YAML解析
├── websocket_service.go # WebSocket连接管理服务
├── config.yaml          # 配置文件
├── build.sh             # 构建脚本
├── main_test.go         # 单元测试
└── README.md            # 项目文档
```

## 架构设计

### 分层架构
- **RobotClient**: 机器人业务逻辑层，处理注册、心跳、状态上报等业务功能
- **WebSocketService**: 连接管理层，负责WebSocket连接、重连、消息收发
- **Config**: 配置管理层，处理YAML配置、环境变量、参数验证

### 设计优势
1. **职责分离**: 业务逻辑与连接管理完全分离
2. **可测试性**: 每层都可以独立测试
3. **可扩展性**: 易于添加新功能和修改现有功能
4. **可维护性**: 代码结构清晰，易于理解和维护

## 核心功能

### 1. WebSocket连接管理
- 自动连接和断开处理
- 连接状态监控
- 并发安全的连接操作

### 2. 自动重连机制
- 指数退避算法
- 随机抖动避免雪崩
- 可配置的重连策略
- 最大重连次数限制

### 3. 结构化日志
- 基于zerolog的高性能日志
- 支持JSON和控制台格式
- 可配置的日志级别
- 结构化字段输出

### 4. 配置管理
- YAML配置文件支持
- 环境变量覆盖
- 命令行参数支持
- 配置验证和默认值

### 5. 并发安全
- 使用sync.Mutex保护共享资源
- 线程安全的序列号生成
- 并发安全的连接操作

## 快速开始

### 1. 构建项目
```bash
./build.sh
```

### 2. 运行客户端
```bash
# 使用默认配置
./robot-client

# 使用自定义配置文件
./robot-client -config my_config.yaml
```

### 3. 查看日志
```bash
# 查看JSON格式日志
./robot-client

# 查看控制台格式日志（debug级别）
./robot-client
```

## 配置说明

### 配置文件 (config.yaml)
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

### 环境变量覆盖
```bash
export ROBOT_UCODE="robot_002"
export SERVER_URL="ws://192.168.1.100:8000/ws/control"
export LOGGING_LEVEL="debug"
./robot-client
```

## API接口

### RobotClient
```go
// 创建客户端
client := NewRobotClient(config)

// 启动客户端
err := client.Start()

// 停止客户端
client.Stop()

// 获取统计信息
stats := client.GetStats()
```

### WebSocketService
```go
// 创建WebSocket服务
wsService := NewWebSocketService(config)

// 设置回调函数
wsService.SetCallbacks(
    onConnect,    // 连接成功回调
    onDisconnect, // 连接断开回调
    onMessage,    // 消息接收回调
    onError,      // 错误处理回调
)

// 启动服务
err := wsService.Start()

// 停止服务
wsService.Stop()

// 检查连接状态
connected := wsService.IsConnected()

// 发送消息
err := wsService.SendMessage(message)

// 获取统计信息
stats := wsService.GetStats()
```

## 测试

### 运行所有测试
```bash
go test -v ./...
```

### 运行并发测试
```bash
go test -race ./...
```

### 测试覆盖率
```bash
go test -cover ./...
```

## 日志系统

### 日志级别
- `debug`: 调试信息，包含详细的执行流程
- `info`: 一般信息，包含重要的状态变化
- `warn`: 警告信息，包含需要注意的问题
- `error`: 错误信息，包含需要处理的错误

### 日志格式
- `json`: 结构化JSON格式，适合日志分析
- `console`: 人类可读的控制台格式，适合开发调试

### 日志示例
```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "ucode": "robot_001",
  "server": "ws://localhost:8000/ws/control",
  "message": "Starting robot client"
}
```

## 自动重连机制

### 重连算法
1. **指数退避**: 延迟时间 = 初始延迟 × (退避倍数 ^ 重连次数)
2. **随机抖动**: 添加±20%的随机抖动，避免多个客户端同时重连
3. **最大延迟**: 限制最大重连延迟，避免无限等待

### 重连配置
- `enabled`: 是否启用自动重连
- `max_attempts`: 最大重连次数
- `initial_delay`: 初始重连延迟
- `max_delay`: 最大重连延迟
- `backoff_multiplier`: 退避倍数

### 重连日志
```
{"level":"info","time":"2024-01-15T10:30:00Z","attempt":1,"max_attempts":5,"delay":"2.1s","message":"Scheduling reconnect"}
{"level":"info","time":"2024-01-15T10:30:02Z","attempt":1,"message":"Attempting to reconnect"}
{"level":"info","time":"2024-01-15T10:30:02Z","attempt":1,"message":"Reconnect successful"}
```

## 并发安全

### 保护机制
- **连接互斥锁**: 保护WebSocket连接的所有操作
- **序列号互斥锁**: 保护序列号生成和递增
- **状态互斥锁**: 保护连接状态和重连状态

### 并发测试
```bash
# 运行并发测试
go test -race ./...

# 测试序列号生成的并发安全性
go test -run TestConcurrentSequenceGeneration -race
```

## 性能优化

### 内存优化
- 使用对象池减少GC压力
- 复用消息结构体
- 限制缓冲区大小

### 网络优化
- 设置合理的超时时间
- 使用连接池
- 实现消息压缩

### 日志优化
- 异步日志写入
- 日志级别过滤
- 结构化日志减少解析开销

## 故障排除

### 常见问题

1. **连接失败**
   ```
   检查服务器地址和端口是否正确
   检查网络连接是否正常
   检查防火墙设置
   ```

2. **重连失败**
   ```
   检查重连配置是否正确
   检查服务器是否可用
   查看重连日志了解详细错误
   ```

3. **日志输出问题**
   ```
   检查日志级别设置
   检查日志格式配置
   检查输出目标设置
   ```

### 调试模式
```bash
# 启用debug级别日志
export LOGGING_LEVEL="debug"
./robot-client

# 使用控制台格式查看详细日志
export LOGGING_FORMAT="console"
./robot-client
```

## 开发指南

### 添加新功能
1. 在相应的层中添加功能
2. 添加配置选项
3. 编写单元测试
4. 更新文档

### 代码规范
- 使用Go标准格式化工具
- 遵循Go命名约定
- 添加适当的注释
- 编写单元测试

### 提交规范
- 使用清晰的提交信息
- 包含功能描述和影响范围
- 关联相关的issue

## 版本历史

### v1.0.0 (2024-01-15)
- 初始版本发布
- 支持基本的WebSocket连接
- 实现自动重连机制
- 添加结构化日志
- 支持配置文件管理

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 联系方式

如有问题，请通过以下方式联系：
- 提交GitHub Issue
- 发送邮件至项目维护者 