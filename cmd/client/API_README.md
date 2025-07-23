# HTTP API 接口说明

## 概述

机器人客户端提供了简化的HTTP API接口，支持查询和设置机器人名称，以及获取状态信息。

## 基础信息

- **服务地址**: `http://localhost:8080`
- **API版本**: `v1`
- **基础路径**: `/api/v1`
- **内容类型**: `application/json`

## API接口列表

### 1. 健康检查

**接口**: `GET /api/v1/health`

**描述**: 检查服务健康状态

**响应示例**:
```json
{
  "success": true,
  "message": "健康检查通过",
  "data": {
    "status": "healthy",
    "timestamp": 1640995200,
    "robot": {
      "ucode": "robot_001",
      "name": "我的机器人"
    },
    "websocket": {
      "connected": true
    }
  }
}
```

### 2. 查询名称

**接口**: `GET /api/v1/name`

**描述**: 获取机器人当前名称

**响应示例**:
```json
{
  "success": true,
  "message": "获取名称成功",
  "data": {
    "name": "我的机器人"
  }
}
```

### 3. 设置名称

**接口**: `POST /api/v1/name`

**描述**: 设置机器人名称

**请求体**:
```json
{
  "name": "新机器人名称"
}
```

**响应示例**:
```json
{
  "success": true,
  "message": "设置名称成功",
  "data": {
    "name": "新机器人名称"
  }
}
```

**错误响应**:
```json
{
  "success": false,
  "message": "名称不能为空"
}
```

### 4. 获取状态

**接口**: `GET /api/v1/status`

**描述**: 获取机器人详细状态信息

**响应示例**:
```json
{
  "success": true,
  "message": "获取状态成功",
  "data": {
    "ucode": "robot_001",
    "sequence": 42,
    "connected": true,
    "last_heartbeat": "2024-01-01T12:00:00Z",
    "latency": 50,
    "active_clients": 5,
    "total_commands": 100,
    "failed_commands": 2,
    "last_command_time": "2024-01-01T12:00:00Z"
  }
}
```

## 使用示例

### 使用curl命令

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 获取名称
curl http://localhost:8080/api/v1/name

# 设置名称
curl -X POST http://localhost:8080/api/v1/name \
  -H "Content-Type: application/json" \
  -d '{"name": "我的机器人"}'

# 获取状态
curl http://localhost:8080/api/v1/status
```

### 使用Python

```python
import requests

# 基础URL
base_url = "http://localhost:8080/api/v1"

# 健康检查
response = requests.get(f"{base_url}/health")
print(response.json())

# 获取名称
response = requests.get(f"{base_url}/name")
print(response.json())

# 设置名称
data = {"name": "Python机器人"}
response = requests.post(f"{base_url}/name", json=data)
print(response.json())

# 获取状态
response = requests.get(f"{base_url}/status")
print(response.json())
```

### 使用JavaScript

```javascript
// 基础URL
const baseUrl = "http://localhost:8080/api/v1";

// 健康检查
fetch(`${baseUrl}/health`)
  .then(response => response.json())
  .then(data => console.log(data));

// 获取名称
fetch(`${baseUrl}/name`)
  .then(response => response.json())
  .then(data => console.log(data));

// 设置名称
fetch(`${baseUrl}/name`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({name: "JS机器人"})
})
  .then(response => response.json())
  .then(data => console.log(data));

// 获取状态
fetch(`${baseUrl}/status`)
  .then(response => response.json())
  .then(data => console.log(data));
```

## 错误处理

### HTTP状态码

- `200 OK`: 请求成功
- `400 Bad Request`: 请求参数错误
- `405 Method Not Allowed`: 不支持的HTTP方法

### 错误响应格式

```json
{
  "success": false,
  "message": "错误描述信息"
}
```

### 常见错误

1. **名称不能为空**
   - 原因: 设置名称时传入了空字符串
   - 解决: 提供有效的名称

2. **无效的请求格式**
   - 原因: JSON格式错误
   - 解决: 检查JSON语法

3. **Method not allowed**
   - 原因: 使用了不支持的HTTP方法
   - 解决: 使用正确的HTTP方法

## 测试

### 使用测试脚本

项目提供了测试脚本 `test_api.sh`：

```bash
# 运行所有测试
./test_api.sh

# 使用自定义URL
./test_api.sh -u http://192.168.1.100:8080/api/v1

# 查看帮助
./test_api.sh -h
```

### 手动测试

```bash
# 启动机器人客户端
./robot_client

# 在另一个终端运行测试
curl http://localhost:8080/api/v1/health
```

## 注意事项

1. **并发安全**: API接口使用读写锁保证并发安全
2. **配置持久化**: 名称设置仅在内存中，重启后会恢复配置文件中的值
3. **服务依赖**: API服务依赖于WebSocket连接状态
4. **端口配置**: 默认使用8080端口，可通过代码修改

## 扩展

如需添加新的API接口，可以：

1. 在 `api_server.go` 中添加新的处理函数
2. 在 `Start()` 方法中注册新的路由
3. 更新此文档说明新接口的用法 