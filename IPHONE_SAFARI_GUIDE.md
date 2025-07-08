# iPhone Safari 使用指南

## 问题说明

在iPhone Safari中，由于浏览器的安全限制和网络环境，WebSocket连接可能会遇到以下问题：

1. **localhost 无法访问**：iPhone Safari无法访问 `localhost`，因为这是指iPhone本机
2. **网络环境限制**：需要确保iPhone和服务器在同一网络环境
3. **HTTPS 要求**：某些情况下可能需要HTTPS连接

## 解决方案

### 1. 服务器地址配置

**错误示例**：
```
localhost:8000
127.0.0.1:8000
```

**正确示例**：
```
192.168.1.100:8000
your-server-domain.com
192.168.0.50:8000
```

### 2. 使用步骤

1. **获取服务器IP地址**
   ```bash
   # 在服务器上运行
   ifconfig  # Linux/Mac
   ipconfig  # Windows
   ```

2. **测试网络连接**
   - 在iPhone上打开Safari
   - 访问 `http://服务器IP:端口/health`
   - 如果能看到响应，说明网络连接正常

3. **测试WebSocket连接**
   - 打开 `tools/test_websocket_connection.html`
   - 输入服务器地址
   - 点击"测试连接"
   - 确认连接成功

4. **使用控制界面**
   - 打开 `www/mobile_operator.html`
   - 点击"设置"
   - 输入正确的服务器地址
   - 输入机器人UCode和操作者ID
   - 点击"保存并连接"

### 3. 常见问题排查

#### 连接失败
- [ ] 检查服务器IP地址是否正确
- [ ] 确认服务器正在运行
- [ ] 检查防火墙设置
- [ ] 确认端口是否开放

#### 连接超时
- [ ] 检查网络连接
- [ ] 确认服务器和iPhone在同一网段
- [ ] 尝试ping服务器IP

#### 连接被拒绝
- [ ] 检查服务器WebSocket端口
- [ ] 确认服务器支持WebSocket
- [ ] 检查CORS设置

### 4. 网络配置建议

#### 开发环境
```bash
# 服务器配置
server:
  host: "0.0.0.0"  # 允许所有IP访问
  port: 8000
```

#### 生产环境
```bash
# 使用域名和HTTPS
server:
  host: "your-domain.com"
  port: 443
  ssl: true
```

### 5. 调试技巧

1. **使用Safari开发者工具**
   - 连接Mac到iPhone
   - 在Mac上打开Safari开发者工具
   - 查看Console日志

2. **网络调试**
   ```bash
   # 在服务器上检查连接
   netstat -an | grep 8000
   
   # 检查WebSocket连接
   lsof -i :8000
   ```

3. **日志查看**
   ```bash
   # 查看服务器日志
   tail -f server.log
   ```

### 6. 安全注意事项

1. **防火墙设置**
   - 只开放必要的端口
   - 限制访问IP范围

2. **HTTPS配置**
   - 生产环境建议使用HTTPS
   - 配置SSL证书

3. **访问控制**
   - 实现用户认证
   - 限制连接数量

## 更新日志

- **v1.1**: 添加服务器地址配置选项
- **v1.2**: 改进错误处理和重连机制
- **v1.3**: 添加连接测试工具
- **v1.4**: 优化移动设备体验 