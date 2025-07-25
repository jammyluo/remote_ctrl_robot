# 机器人控制API接口协议说明

## 概述

本文档详细描述了机器人控制系统的HTTP API接口协议，包括请求格式、响应格式、错误处理等规范。

## 基础信息

- **协议**: HTTP/HTTPS
- **数据格式**: JSON
- **字符编码**: UTF-8
- **基础路径**: `/api/v1`

## 通用响应格式

所有API接口都使用统一的响应格式：

```json
{
  "success": true/false,
  "message": "操作结果描述",
  "data": {
    // 具体数据（可选）
  }
}
```

### 响应字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| success | boolean | 是 | 操作是否成功 |
| message | string | 是 | 操作结果描述 |
| data | object | 否 | 返回的数据，失败时通常为空 |

### HTTP状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 405 | 请求方法不支持 |
| 500 | 服务器内部错误 |

## 接口详细说明

### 1. 射击接口

**接口信息**
- **路径**: `/api/v1/shoot`
- **方法**: POST
- **功能**: 执行机器人射击操作

**请求格式**
```http
POST /api/v1/shoot HTTP/1.1
Host: localhost:8080
Content-Type: application/json
```

**请求参数**
无参数

**响应示例**
```json
{
  "success": true,
  "message": "Success"
}
```

**错误响应**
```json
{
  "success": false,
  "message": "射击失败：弹药不足"
}
```

### 2. 弹药查询接口

**接口信息**
- **路径**: `/api/v1/ammo`
- **方法**: GET
- **功能**: 获取当前弹药数量

**请求格式**
```http
GET /api/v1/ammo HTTP/1.1
Host: localhost:8080
```

**请求参数**
无参数

**响应示例**
```json
{
  "success": true,
  "message": "Success",
  "data": 30
}
```

**错误响应**
```json
{
  "success": false,
  "message": "获取弹药信息失败"
}
```

### 3. 弹药更换接口

**接口信息**
- **路径**: `/api/v1/ammo/change`
- **方法**: POST
- **功能**: 更换弹药

**请求格式**
```http
POST /api/v1/ammo/change HTTP/1.1
Host: localhost:8080
Content-Type: application/json
```

**请求参数**
无参数

**响应示例**
```json
{
  "success": true,
  "message": "Success"
}
```

**错误响应**
```json
{
  "success": false,
  "message": "更换弹药失败"
}
```

### 4. 生命值查询接口

**接口信息**
- **路径**: `/api/v1/health`
- **方法**: GET
- **功能**: 获取当前生命值

**请求格式**
```http
GET /api/v1/health HTTP/1.1
Host: localhost:8080
```

**请求参数**
无参数

**响应示例**
```json
{
  "success": true,
  "message": "Success",
  "data": {
    "health": 100
  }
}
```

**错误响应**
```json
{
  "success": false,
  "message": "获取生命值失败"
}
```

## 错误处理规范

### 1. 通用错误码

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查请求参数格式 |
| 405 | 请求方法不支持 | 使用正确的HTTP方法 |
| 500 | 服务器内部错误 | 联系系统管理员 |

### 2. 业务错误码

| 错误类型 | 说明 | 处理建议 |
|----------|------|----------|
| 射击失败 | 弹药不足或设备故障 | 检查弹药状态或设备连接 |
| 弹药查询失败 | 设备连接异常 | 检查设备连接状态 |
| 更换弹药失败 | 设备故障 | 检查设备状态 |
| 生命值查询失败 | 设备连接异常 | 检查设备连接状态 |

## 安全规范

### 1. 访问控制
- 当前版本无认证机制
- 建议在生产环境中添加API密钥认证

### 2. 请求限制
- 建议实现请求频率限制
- 防止API滥用

### 3. 数据传输
- 生产环境建议使用HTTPS
- 敏感数据加密传输

## 使用示例

### Python示例

```python
import requests
import json

BASE_URL = "http://localhost:8080/api/v1"

# 射击
response = requests.post(f"{BASE_URL}/shoot")
print(response.json())

# 查询弹药
response = requests.get(f"{BASE_URL}/ammo")
print(response.json())

# 更换弹药
response = requests.post(f"{BASE_URL}/ammo/change")
print(response.json())

# 查询生命值
response = requests.get(f"{BASE_URL}/health")
print(response.json())
```

### cURL示例

```bash
# 射击
curl -X POST http://localhost:8080/api/v1/shoot

# 查询弹药
curl http://localhost:8080/api/v1/ammo

# 更换弹药
curl -X POST http://localhost:8080/api/v1/ammo/change

# 查询生命值
curl http://localhost:8080/api/v1/health
```

### JavaScript示例

```javascript
const BASE_URL = 'http://localhost:8080/api/v1';

// 射击
async function shoot() {
    const response = await fetch(`${BASE_URL}/shoot`, {
        method: 'POST'
    });
    return await response.json();
}

// 查询弹药
async function getAmmo() {
    const response = await fetch(`${BASE_URL}/ammo`);
    return await response.json();
}

// 更换弹药
async function changeAmmo() {
    const response = await fetch(`${BASE_URL}/ammo/change`, {
        method: 'POST'
    });
    return await response.json();
}

// 查询生命值
async function getHealth() {
    const response = await fetch(`${BASE_URL}/health`);
    return await response.json();
}
```

## 版本控制

### 当前版本
- **版本号**: v1
- **状态**: 稳定版
- **兼容性**: 向后兼容

### 版本更新策略
- 主版本号变更：不兼容的API变更
- 次版本号变更：新增功能，向后兼容
- 修订版本号变更：Bug修复，向后兼容

## 测试规范

### 1. 功能测试
- 验证所有接口的基本功能
- 测试正常和异常情况
- 验证响应格式正确性

### 2. 性能测试
- 测试并发请求处理能力
- 验证响应时间
- 测试系统稳定性

### 3. 安全测试
- 测试输入验证
- 验证错误处理
- 测试访问控制

## 部署说明

### 开发环境
```bash
# 启动服务
./robot_client --config config/config.yaml

# 测试接口
curl http://localhost:8080/api/v1/health
```

### 生产环境
```bash
# 后台运行
nohup ./robot_client --config config/config.yaml > robot.log 2>&1 &

# 检查服务状态
curl http://localhost:8080/api/v1/health

# 停止服务
pkill -f robot_client
```