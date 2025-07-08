# 机器人对抗游戏系统

基于WebSocket的实时多人机器人对抗游戏系统，支持机器人之间的射击、移动、血量管理等游戏功能。

## 功能特性

### 🎮 核心游戏功能
- **多人对抗**: 支持多个机器人同时参与游戏
- **实时射击**: 机器人可以射击其他机器人，造成伤害
- **血量系统**: 每个机器人有血量，被击中会减少血量
- **复活机制**: 机器人死亡后会在指定时间后复活
- **得分系统**: 击杀其他机器人获得得分
- **实时同步**: 所有游戏状态实时同步到所有参与者

### 🔧 技术特性
- **WebSocket通信**: 基于WebSocket的实时双向通信
- **服务器权威**: 服务器作为游戏逻辑的权威源
- **50ms游戏循环**: 每50ms更新一次游戏状态
- **碰撞检测**: 精确的子弹碰撞检测
- **事件系统**: 完整的游戏事件记录和广播

## 系统架构

### 服务器端组件
1. **GameService**: 游戏逻辑管理器
   - 游戏状态管理
   - 机器人状态管理
   - 子弹物理计算
   - 碰撞检测
   - 事件广播

2. **WebSocketHandlers**: WebSocket消息处理器
   - 游戏命令处理
   - 连接管理
   - 消息路由

3. **数据模型**: 游戏数据结构
   - 游戏状态
   - 机器人状态
   - 子弹信息
   - 游戏事件

### 客户端组件
1. **Web客户端**: `game_client.html`
   - 可视化游戏界面
   - 实时状态显示
   - 手动控制功能

2. **Python测试客户端**: `test_game_client.py`
   - 自动化测试
   - 多机器人模拟
   - 性能测试

## 快速开始

### 1. 启动服务器

```bash
# 编译并运行服务器
go run cmd/server/main.go
```

服务器将在 `http://localhost:8080` 启动。

### 2. 使用Web客户端

1. 打开浏览器访问 `http://localhost:8080/game_client.html`
2. 填写机器人信息：
   - UCode: 机器人唯一标识
   - 名称: 机器人显示名称
   - 服务器地址: `ws://localhost:8080/ws/control`
3. 点击"连接服务器"
4. 连接成功后点击"加入游戏"
5. 使用控制面板进行游戏操作

### 3. 使用Python测试客户端

```bash
# 安装依赖
pip install websockets

# 运行测试
python test_game_client.py
```

这将启动3个机器人客户端进行自动化测试。

## 游戏命令

### 机器人命令
- `CMD_JOIN_GAME`: 加入游戏
- `CMD_LEAVE_GAME`: 离开游戏
- `CMD_GAME_SHOOT`: 射击
- `CMD_GAME_MOVE`: 移动
- `CMD_GAME_STATUS`: 获取游戏状态

### 操作者命令
- `CMD_GAME_START`: 开始游戏
- `CMD_GAME_STOP`: 停止游戏

## 游戏配置

默认游戏配置：
```go
GameConfig{
    MaxHealth:     100,    // 最大血量
    BulletDamage:  20,     // 子弹伤害
    BulletRange:   50,     // 射程
    BulletSpeed:   10,     // 子弹速度
    RespawnTime:   10,     // 复活时间(秒)
    GameDuration:  300,    // 游戏时长(秒)
    MapWidth:      100,    // 地图宽度
    MapHeight:     100,    // 地图高度
    ShootCooldown: 1.0,    // 射击冷却时间(秒)
}
```

## 游戏流程

### 1. 游戏准备阶段
1. 机器人连接到服务器并注册
2. 机器人加入游戏房间
3. 等待至少2个机器人加入
4. 操作者可以开始游戏

### 2. 游戏进行阶段
1. 机器人可以自由移动和射击
2. 服务器每50ms更新游戏状态
3. 子弹移动和碰撞检测
4. 伤害计算和血量更新
5. 死亡和复活处理
6. 实时状态广播

### 3. 游戏结束阶段
1. 游戏时间结束或手动停止
2. 计算最终得分和排名
3. 确定获胜者
4. 清理游戏状态

## 消息格式

### 注册消息
```json
{
    "type": "Request",
    "command": "CMD_REGISTER",
    "sequence": 1,
    "ucode": "robot_001",
    "client_type": "robot",
    "version": "1.0.0",
    "data": {
        "name": "机器人001"
    }
}
```

### 加入游戏消息
```json
{
    "type": "Request",
    "command": "CMD_JOIN_GAME",
    "sequence": 2,
    "ucode": "robot_001",
    "client_type": "robot",
    "version": "1.0.0",
    "data": {
        "game_id": "default_game",
        "name": "机器人001"
    }
}
```

### 射击消息
```json
{
    "type": "Request",
    "command": "CMD_GAME_SHOOT",
    "sequence": 3,
    "ucode": "robot_001",
    "client_type": "robot",
    "version": "1.0.0",
    "data": {
        "target_x": 10.5,
        "target_y": 20.3,
        "target_z": 0
    }
}
```

### 移动消息
```json
{
    "type": "Request",
    "command": "CMD_GAME_MOVE",
    "sequence": 4,
    "ucode": "robot_001",
    "client_type": "robot",
    "version": "1.0.0",
    "data": {
        "position": {
            "x": 15.0,
            "y": 25.0,
            "z": 0
        },
        "direction": 1.57
    }
}
```

## 游戏状态

### 机器人状态
```json
{
    "ucode": "robot_001",
    "name": "机器人001",
    "health": 80,
    "max_health": 100,
    "position": {
        "x": 15.0,
        "y": 25.0,
        "z": 0
    },
    "direction": 1.57,
    "is_alive": true,
    "score": 30,
    "kills": 3,
    "deaths": 1,
    "shots_fired": 15,
    "shots_hit": 8
}
```

### 游戏状态
```json
{
    "game_id": "default_game",
    "status": "playing",
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T10:05:00Z",
    "duration": 300,
    "robots": {
        "robot_001": { /* 机器人状态 */ },
        "robot_002": { /* 机器人状态 */ }
    },
    "bullets": [
        {
            "id": "bullet_001",
            "shooter_ucode": "robot_001",
            "start_pos": { "x": 10, "y": 20, "z": 0 },
            "current_pos": { "x": 12, "y": 22, "z": 0 },
            "direction": { "x": 0.5, "y": 0.5, "z": 0 },
            "speed": 10,
            "damage": 20,
            "range": 50,
            "is_active": true
        }
    ],
    "config": { /* 游戏配置 */ },
    "winner": "robot_001",
    "statistics": { /* 游戏统计 */ }
}
```

## 扩展功能

### 可能的扩展
1. **地图系统**: 添加障碍物和地形
2. **武器系统**: 不同类型的武器和弹药
3. **团队模式**: 支持团队对抗
4. **AI机器人**: 添加AI控制的机器人
5. **观战模式**: 支持观众观看游戏
6. **排行榜**: 历史战绩和排名系统
7. **自定义配置**: 可配置的游戏参数
8. **回放系统**: 游戏回放功能

### 性能优化
1. **空间分区**: 使用四叉树优化碰撞检测
2. **消息压缩**: 减少网络传输数据量
3. **预测算法**: 客户端预测减少延迟
4. **连接池**: 优化WebSocket连接管理

## 故障排除

### 常见问题
1. **连接失败**: 检查服务器是否启动，端口是否正确
2. **游戏无法开始**: 确保至少有两个机器人加入游戏
3. **射击无效**: 检查射击冷却时间和目标位置
4. **状态不同步**: 检查网络连接和服务器负载

### 调试方法
1. 查看服务器日志了解详细错误信息
2. 使用浏览器开发者工具查看WebSocket消息
3. 运行Python测试客户端进行自动化测试
4. 检查游戏配置参数是否正确

## 开发指南

### 添加新功能
1. 在 `models/game_types.go` 中定义新的数据结构
2. 在 `services/game_service.go` 中实现游戏逻辑
3. 在 `handlers/websocket_handlers.go` 中添加消息处理
4. 更新客户端代码支持新功能
5. 添加相应的测试用例

### 代码结构
```
internal/
├── models/
│   ├── types.go          # 基础数据类型
│   └── game_types.go     # 游戏相关数据类型
├── services/
│   ├── robot_service.go  # 机器人服务
│   └── game_service.go   # 游戏服务
└── handlers/
    ├── api_handlers.go   # API处理器
    └── websocket_handlers.go # WebSocket处理器
```

## 许可证

本项目基于现有代码架构开发，请遵循原项目的许可证要求。 