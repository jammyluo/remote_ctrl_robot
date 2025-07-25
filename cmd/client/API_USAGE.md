# 机器人控制API使用指南

## 快速开始

### 1. 启动服务
```bash
# 编译并运行机器人客户端
go build -o robot_client .
./robot_client
```

### 2. 测试API接口

#### 使用Shell脚本测试
```bash
# 运行自动化测试
./test_api.sh
```

#### 使用Python脚本测试
```bash
# 运行Python演示脚本
python3 demo_api.py
```

#### 使用curl手动测试
```bash
# 查询生命值
curl http://localhost:8080/api/v1/health

# 查询弹药数量
curl http://localhost:8080/api/v1/ammo

# 执行射击
curl -X POST http://localhost:8080/api/v1/shoot

# 更换弹药
curl -X POST http://localhost:8080/api/v1/ammo/change
```

## API接口说明

### 接口列表

| 接口 | 方法 | 功能 | 示例 |
|------|------|------|------|
| `/api/v1/health` | GET | 查询生命值 | `curl http://localhost:8080/api/v1/health` |
| `/api/v1/ammo` | GET | 查询弹药数量 | `curl http://localhost:8080/api/v1/ammo` |
| `/api/v1/shoot` | POST | 执行射击 | `curl -X POST http://localhost:8080/api/v1/shoot` |
| `/api/v1/ammo/change` | POST | 更换弹药 | `curl -X POST http://localhost:8080/api/v1/ammo/change` |

### 响应格式

所有接口都返回JSON格式的响应：

```json
{
  "success": true,
  "message": "Success",
  "data": {
    // 具体数据（可选）
  }
}
```

## 测试工具说明

### 1. test_api.sh
- **功能**: Shell自动化测试脚本
- **特点**: 快速测试所有接口，包含错误处理测试
- **使用**: `./test_api.sh`

### 2. demo_api.py
- **功能**: Python交互式测试脚本
- **特点**: 详细的测试报告，支持交互模式
- **使用**: `python3 demo_api.py`

### 3. 交互模式
在Python脚本中，可以进入交互模式进行手动测试：
```bash
python3 demo_api.py
# 选择进入交互模式
```

可用命令：
- `health` - 查询生命值
- `ammo` - 查询弹药数量
- `shoot` - 执行射击
- `change` - 更换弹药
- `sequence` - 执行操作序列
- `status` - 查询完整状态
- `quit` - 退出

## 开发集成

### Python集成示例
```python
import requests

class RobotAPI:
    def __init__(self, base_url="http://localhost:8080/api/v1"):
        self.base_url = base_url
    
    def get_health(self):
        response = requests.get(f"{self.base_url}/health")
        return response.json()
    
    def get_ammo(self):
        response = requests.get(f"{self.base_url}/ammo")
        return response.json()
    
    def shoot(self):
        response = requests.post(f"{self.base_url}/shoot")
        return response.json()
    
    def change_ammo(self):
        response = requests.post(f"{self.base_url}/ammo/change")
        return response.json()

# 使用示例
robot = RobotAPI()
health = robot.get_health()
print(f"生命值: {health['data']['health']}")
```

### JavaScript集成示例
```javascript
class RobotAPI {
    constructor(baseUrl = 'http://localhost:8080/api/v1') {
        this.baseUrl = baseUrl;
    }
    
    async getHealth() {
        const response = await fetch(`${this.baseUrl}/health`);
        return await response.json();
    }
    
    async getAmmo() {
        const response = await fetch(`${this.baseUrl}/ammo`);
        return await response.json();
    }
    
    async shoot() {
        const response = await fetch(`${this.baseUrl}/shoot`, {
            method: 'POST'
        });
        return await response.json();
    }
    
    async changeAmmo() {
        const response = await fetch(`${this.baseUrl}/ammo/change`, {
            method: 'POST'
        });
        return await response.json();
    }
}

// 使用示例
const robot = new RobotAPI();
robot.getHealth().then(data => {
    console.log('生命值:', data.data.health);
});
```

## 故障排除

### 常见问题

1. **连接失败**
   ```
   ❌ 无法连接到API服务器
   ```
   - 检查机器人客户端是否正在运行
   - 检查端口8080是否被占用
   - 检查防火墙设置

2. **射击失败**
   ```json
   {
     "success": false,
     "message": "射击失败：弹药不足"
   }
   ```
   - 检查弹药数量
   - 尝试更换弹药

3. **设备连接错误**
   ```json
   {
     "success": false,
     "message": "设备连接异常"
   }
   ```
   - 检查机器人硬件连接
   - 重启机器人客户端

### 调试技巧

1. **查看详细日志**
   ```bash
   ./robot_client 2>&1 | tee robot.log
   ```

2. **检查API响应**
   ```bash
   curl -v http://localhost:8080/api/v1/health
   ```

3. **监控网络连接**
   ```bash
   netstat -an | grep 8080
   ```

## 性能优化

### 1. 并发请求
API支持并发请求，但建议：
- 射击操作间隔至少1秒
- 避免频繁的状态查询
- 合理使用弹药更换

### 2. 错误重试
建议实现错误重试机制：
```python
import time

def shoot_with_retry(robot_api, max_retries=3):
    for i in range(max_retries):
        try:
            result = robot_api.shoot()
            if result['success']:
                return result
        except Exception as e:
            print(f"射击失败，重试 {i+1}/{max_retries}: {e}")
            time.sleep(1)
    return None
```

## 安全建议

1. **生产环境部署**
   - 使用HTTPS协议
   - 添加API密钥认证
   - 实现请求频率限制

2. **访问控制**
   - 限制API访问IP
   - 监控异常访问
   - 记录访问日志

3. **数据保护**
   - 加密敏感数据
   - 定期备份配置
   - 安全删除日志

## 扩展开发

### 添加新接口
1. 在`api_server.go`中添加新的处理函数
2. 在路由中注册新接口
3. 更新测试脚本
4. 更新文档

### 自定义响应格式
可以修改`APIResponse`结构体来自定义响应格式。

### 添加认证机制
建议在生产环境中添加JWT或API密钥认证。

---

**更多信息**
- 详细API文档: [API_PROTOCOL.md](API_PROTOCOL.md)
- 实现总结: [HTTP_API_SUMMARY.md](HTTP_API_SUMMARY.md)
- 测试报告: 运行测试脚本查看 