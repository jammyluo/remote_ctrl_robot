# 机器人控制使用指南

## 概述

本指南介绍如何使用Go2机器人SDK和WebSocket连接来实现远程机器人控制。

## 功能特性

- 🤖 **Go2机器人集成**: 支持真实的Go2机器人控制
- 📱 **移动端控制**: 通过手机网页控制机器人
- 🔄 **实时通信**: WebSocket低延迟通信
- 🎮 **多种控制模式**: 移动、射击、升高、降低等
- 🛡️ **安全保护**: 超时停止、错误处理
- 🎭 **模拟模式**: 无SDK时的模拟运行

## 快速开始

### 1. 环境准备

#### 安装依赖
```bash
# 安装WebSocket客户端
pip install websocket-client

# 安装Go2 SDK (可选)
# 参考 unitree_sdk2_python 目录下的安装说明
```

#### 网络配置
```bash
# 查看网络接口
ifconfig

# 常见网络接口名称
# - lo: 本地回环
# - eth0: 有线网络
# - wlan0: 无线网络
# - en0: macOS网络接口
```

### 2. 启动机器人客户端

#### 基本用法
```bash
# 使用默认网络接口
python3 test_robot_real.py robot_001

# 指定网络接口
python3 test_robot_real.py robot_001 eth0
python3 test_robot_real.py robot_001 en0  # macOS
```

#### 运行示例
```bash
$ python3 test_robot_real.py robot_001
🤖 启动机器人客户端
   UCode: robot_001
   网络接口: lo
   Go2 SDK: 可用
🔧 初始化Go2 SDK，网络接口: lo
✅ Go2 SDK 初始化成功
🤖 机器人已站立
🔗 连接到服务器: ws://localhost:8000/ws/control
📤 发送注册消息: {...}
✅ 机器人 robot_001 连接成功
🔄 机器人 robot_001 运行中... (按 Ctrl+C 退出)
📱 可以通过移动端控制机器人
```

### 3. 移动端控制

1. **打开控制页面**: 访问 `www/mobile_operator.html`
2. **设置连接**: 点击"设置"按钮
3. **输入参数**:
   - 服务器地址: 机器人所在服务器的IP
   - 机器人UCode: `robot_001`
   - 操作者ID: `operator_001`
4. **连接控制**: 点击"保存并连接"
5. **开始控制**: 使用摇杆和功能按钮控制机器人

## 控制命令详解

### 移动命令 (Move)
```json
{
  "action": "Move",
  "params": {
    "priority": "1",
    "vx": "0.076",      // 前进速度 (m/s)
    "vy": "-0.22",      // 侧向速度 (m/s)
    "vyaw": "0"         // 转向速度 (rad/s)
  },
  "timestamp": 1751973741607
}
```

**参数说明**:
- `vx`: 前进/后退速度，正值前进，负值后退
- `vy`: 左转/右转速度，负值左转，正值右转
- `vyaw`: 原地转向速度，正值逆时针，负值顺时针
- `priority`: 命令优先级 (1-10)

### 功能命令

#### 射击命令 (shoot)
```json
{
  "action": "shoot",
  "timestamp": 1751973741608
}
```

#### 升高命令 (raise)
```json
{
  "action": "raise",
  "timestamp": 1751973741609
}
```

#### 降低命令 (lower)
```json
{
  "action": "lower",
  "timestamp": 1751973741610
}
```

## 机器人状态

### 状态更新
机器人会定期发送状态信息：
```json
{
  "type": "Request",
  "command": "CMD_UPDATE_ROBOT_STATUS",
  "data": {
    "status": "moving",           // 状态: idle, moving, error
    "battery_level": 85,          // 电池电量 (%)
    "temperature": 28.5,          // 温度 (°C)
    "base_position": [1.2, 0.5, 0.0],     // 位置 [x, y, z]
    "base_orientation": [0, 0, 0, 1],     // 姿态 [x, y, z, w]
    "error_code": 0,              // 错误代码
    "error_message": "",          // 错误信息
    "timestamp": 1751973741607
  }
}
```

### 状态说明
- `idle`: 空闲状态
- `moving`: 移动中
- `error`: 错误状态

## 安全特性

### 1. 超时保护
- 移动命令超时时间: 0.5秒
- 超时后自动停止机器人

### 2. 连接保护
- 连接断开时自动停止机器人
- 程序退出时机器人蹲下

### 3. 错误处理
- SDK初始化失败时使用模拟模式
- 网络异常时自动重连

## 测试工具

### 1. 控制命令解析测试
```bash
python3 test_control_parser.py
```

### 2. WebSocket连接测试
```bash
# 打开 test_websocket_connection.html
# 测试服务器连接
```

### 3. 模拟机器人测试
```bash
# 使用模拟模式测试
python3 test_robot_real.py robot_test
```

## 故障排除

### 常见问题

#### 1. Go2 SDK初始化失败
```
❌ Go2 SDK 初始化失败，错误码: -1
```
**解决方案**:
- 检查网络接口名称
- 确认机器人已开机并连接网络
- 检查SDK安装是否正确

#### 2. WebSocket连接失败
```
❌ 连接超时
```
**解决方案**:
- 检查服务器是否运行
- 确认网络连接正常
- 检查防火墙设置

#### 3. 移动命令不响应
```
⚠️ 未知的控制命令: unknown_action
```
**解决方案**:
- 检查命令格式是否正确
- 确认action字段值有效

### 调试技巧

#### 1. 查看详细日志
```python
# 在代码中添加更多print语句
print(f"🎮 处理控制命令: {action}")
print(f"🚶 移动命令: vx={vx:.3f}, vy={vy:.3f}, vyaw={vyaw:.3f}")
```

#### 2. 网络调试
```bash
# 检查网络连接
ping <服务器IP>

# 检查端口
telnet <服务器IP> 8000
```

#### 3. 模拟模式测试
```bash
# 在没有SDK的环境中测试
python3 test_robot_real.py robot_sim
```

## 扩展开发

### 1. 添加新的控制命令
```python
def handle_custom_command(self, params):
    """处理自定义命令"""
    print("🎯 执行自定义命令")
    if self.sport_client:
        # 实现具体的机器人动作
        pass
```

### 2. 集成其他机器人
```python
# 可以扩展支持其他型号的机器人
class B2RobotClient(RobotClient):
    def init_robot_sdk(self):
        # 初始化B2机器人SDK
        pass
```

### 3. 添加传感器数据
```python
def get_sensor_data(self):
    """获取传感器数据"""
    return {
        "imu": self.get_imu_data(),
        "lidar": self.get_lidar_data(),
        "camera": self.get_camera_data()
    }
```

## 更新日志

- **v1.0**: 基础WebSocket连接
- **v1.1**: 集成Go2 SDK
- **v1.2**: 添加移动控制
- **v1.3**: 添加功能命令
- **v1.4**: 完善状态管理
- **v1.5**: 添加安全保护 