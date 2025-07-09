#!/usr/bin/env python3
"""
控制命令解析测试脚本
用于测试和验证来自移动端的控制命令格式
"""

import json

def test_control_command_parsing():
    """测试控制命令解析"""
    
    # 测试用例1: 移动命令
    test_message_1 = {
        "level": "debug",
        "command": "CMD_CONTROL_ROBOT",
        "ucode": "operator_001",
        "client_type": "operator",
        "version": "1.0.0",
        "sequence": "222",
        "data": "{\"action\":\"Move\",\"params\":{\"priority\":\"1\",\"vx\":\"0.0760000000000003\",\"vy\":\"-0.22\",\"vyaw\":\"0\"},\"timestamp\":1751973741607}",
        "client": "mobile_operator"
    }
    
    # 测试用例2: 射击命令
    test_message_2 = {
        "level": "debug",
        "command": "CMD_CONTROL_ROBOT",
        "ucode": "operator_001",
        "client_type": "operator",
        "version": "1.0.0",
        "sequence": "223",
        "data": "{\"action\":\"shoot\",\"timestamp\":1751973741608}",
        "client": "mobile_operator"
    }
    
    # 测试用例3: 升高命令
    test_message_3 = {
        "level": "debug",
        "command": "CMD_CONTROL_ROBOT",
        "ucode": "operator_001",
        "client_type": "operator",
        "version": "1.0.0",
        "sequence": "224",
        "data": "{\"action\":\"raise\",\"timestamp\":1751973741609}",
        "client": "mobile_operator"
    }
    
    test_cases = [
        ("移动命令", test_message_1),
        ("射击命令", test_message_2),
        ("升高命令", test_message_3)
    ]
    
    for test_name, message in test_cases:
        print(f"\n🧪 测试: {test_name}")
        print(f"原始消息: {json.dumps(message, indent=2, ensure_ascii=False)}")
        
        # 解析data字段
        try:
            if isinstance(message.get("data"), str):
                data = json.loads(message["data"])
            else:
                data = message.get("data", {})
            
            action = data.get("action")
            params = data.get("params", {})
            timestamp = data.get("timestamp")
            
            print(f"✅ 解析结果:")
            print(f"   Action: {action}")
            print(f"   Params: {params}")
            print(f"   Timestamp: {timestamp}")
            
            # 如果是移动命令，解析速度参数
            if action == "Move":
                vx = float(params.get("vx", 0))
                vy = float(params.get("vy", 0))
                vyaw = float(params.get("vyaw", 0))
                priority = int(params.get("priority", 1))
                
                print(f"   🚶 移动参数:")
                print(f"      vx: {vx:.3f} m/s")
                print(f"      vy: {vy:.3f} m/s")
                print(f"      vyaw: {vyaw:.3f} rad/s")
                print(f"      priority: {priority}")
                
        except Exception as e:
            print(f"❌ 解析失败: {e}")

def test_robot_response():
    """测试机器人响应格式"""
    
    # 模拟机器人状态响应
    robot_status = {
        "type": "Response",
        "command": "CMD_CONTROL_ROBOT",
        "sequence": 222,
        "ucode": "robot_001",
        "client_type": "robot",
        "version": "1.0.0",
        "data": {
            "success": True,
            "message": "移动命令执行成功",
            "robot_status": {
                "status": "moving",
                "battery_level": 85,
                "temperature": 28.5,
                "base_position": [1.2, 0.5, 0.0],
                "base_orientation": [0.0, 0.0, 0.0, 1.0],
                "error_code": 0,
                "error_message": ""
            },
            "timestamp": 1751973741607
        }
    }
    
    print(f"\n🤖 机器人响应示例:")
    print(json.dumps(robot_status, indent=2, ensure_ascii=False))

def main():
    print("🔧 控制命令解析测试")
    print("=" * 50)
    
    # 测试命令解析
    test_control_command_parsing()
    
    # 测试机器人响应
    test_robot_response()
    
    print("\n✅ 测试完成")

if __name__ == "__main__":
    main() 