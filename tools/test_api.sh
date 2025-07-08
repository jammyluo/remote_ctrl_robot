#!/bin/bash

# 远程控制机器人API测试脚本
# 需要指定机器人UCODE进行测试

BASE_URL="http://localhost:8080"
UCODE="123456"  # 测试用的机器人UCODE

echo "=== 远程控制机器人API测试 ==="
echo "测试UCODE: $UCODE"
echo ""

# 1. 健康检查
echo "1. 健康检查"
curl -s -X GET "$BASE_URL/health" | jq .
echo ""

# 2. 获取系统状态
echo "2. 获取系统状态"
curl -s -X GET "$BASE_URL/api/v1/system/status" | jq .
echo ""

# 3. 获取连接状态
echo "3. 获取连接状态"
curl -s -X GET "$BASE_URL/api/v1/control/connection" | jq .
echo ""

# 4. 获取WebRTC播放地址（需要UCODE）
echo "4. 获取WebRTC播放地址 (UCODE: $UCODE)"
curl -s -X GET "$BASE_URL/api/v1/webrtc/play-url?ucode=$UCODE" | jq .
echo ""

# 5. 获取机器人状态（需要UCODE）
echo "5. 获取机器人状态 (UCODE: $UCODE)"
curl -s -X GET "$BASE_URL/api/v1/control/status?ucode=$UCODE" | jq .
echo ""

# 6. 发送关节位置控制命令（需要UCODE）
echo "6. 发送关节位置控制命令 (UCODE: $UCODE)"
curl -s -X POST "$BASE_URL/api/v1/control/command?ucode=$UCODE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "joint_position",
    "command_id": "cmd_001",
    "priority": 5,
    "timestamp": 1640995200000,
    "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
    "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
  }' | jq .
echo ""

# 7. 发送速度控制命令（需要UCODE）
echo "7. 发送速度控制命令 (UCODE: $UCODE)"
curl -s -X POST "$BASE_URL/api/v1/control/command?ucode=$UCODE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "velocity",
    "command_id": "cmd_002",
    "priority": 3,
    "timestamp": 1640995200000,
    "joint_pos": [0.0, 0.0, 0.0, 0.0, 0.0, 0.0],
    "velocities": [0.5, 0.5, 0.5, 0.5, 0.5, 0.5]
  }' | jq .
echo ""

# 8. 发送紧急停止命令（需要UCODE）
echo "8. 发送紧急停止命令 (UCODE: $UCODE)"
curl -s -X POST "$BASE_URL/api/v1/control/command?ucode=$UCODE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "emergency_stop",
    "command_id": "cmd_003",
    "priority": 10,
    "timestamp": 1640995200000,
    "joint_pos": [],
    "velocities": []
  }' | jq .
echo ""

# 9. 发送回零命令（需要UCODE）
echo "9. 发送回零命令 (UCODE: $UCODE)"
curl -s -X POST "$BASE_URL/api/v1/control/command?ucode=$UCODE" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "home",
    "command_id": "cmd_004",
    "priority": 7,
    "timestamp": 1640995200000,
    "joint_pos": [0.0, 0.0, 0.0, 0.0, 0.0, 0.0],
    "velocities": [0.2, 0.2, 0.2, 0.2, 0.2, 0.2]
  }' | jq .
echo ""

# 10. 测试无效UCODE
echo "10. 测试无效UCODE"
curl -s -X GET "$BASE_URL/api/v1/control/status?ucode=invalid_ucode" | jq .
echo ""

# 11. 测试缺少UCODE参数
echo "11. 测试缺少UCODE参数"
curl -s -X GET "$BASE_URL/api/v1/control/status" | jq .
echo ""

echo "=== 测试完成 ==="
echo ""
echo "注意事项："
echo "1. 确保服务器已启动 (go run cmd/server/main.go)"
echo "2. 确保机器人已通过WebSocket连接并注册UCODE: $UCODE"
echo "3. 如果机器人未在线，相关API会返回404错误"
echo "4. 可以通过 /api/v1/system/status 查看在线机器人列表" 