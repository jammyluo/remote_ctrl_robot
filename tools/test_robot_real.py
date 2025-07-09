#!/usr/bin/env python3
"""
机器人客户端测试脚本
集成Go2机器人SDK，通过WebSocket连接到服务器并处理控制命令
"""

import websocket
import json
import time
import threading
import sys
import math
from dataclasses import dataclass

# 尝试导入Go2 SDK
try:
    from unitree_sdk2py.core.channel import ChannelSubscriber, ChannelFactoryInitialize
    from unitree_sdk2py.go2.sport.sport_client import SportClient
    GO2_SDK_AVAILABLE = True
    print("✅ Go2 SDK 可用")
except ImportError:
    GO2_SDK_AVAILABLE = False
    print("⚠️  Go2 SDK 不可用，将使用模拟模式")

@dataclass
class RobotState:
    """机器人状态"""
    status: str = "idle"
    battery_level: int = 100
    temperature: float = 25.0
    base_position: list = None
    base_orientation: list = None
    error_code: int = 0
    error_message: str = ""
    
    def __post_init__(self):
        if self.base_position is None:
            self.base_position = [0.0, 0.0, 0.0]
        if self.base_orientation is None:
            self.base_orientation = [0.0, 0.0, 0.0, 0.0]

class RobotClient:
    def __init__(self, ucode, server_url="ws://localhost:8000/ws/control", network_interface="lo"):
        self.ucode = ucode
        self.server_url = server_url
        self.network_interface = network_interface
        self.ws = None
        self.connected = False
        self.sequence = 1
        self.robot_state = RobotState()
        
        # Go2 SDK 相关
        self.sport_client = None
        self.control_active = False
        self.last_move_time = 0
        self.move_timeout = 0.5  # 移动命令超时时间（秒）
        
        # 初始化Go2 SDK
        if GO2_SDK_AVAILABLE:
            self.init_go2_sdk()
    
    def init_go2_sdk(self):
        """初始化Go2 SDK"""
        try:
            print(f"🔧 初始化Go2 SDK，网络接口: {self.network_interface}")
            ChannelFactoryInitialize(0, self.network_interface)
            
            self.sport_client = SportClient()
            self.sport_client.SetTimeout(10.0)
            ret = self.sport_client.Init()
            
            if ret == 0:
                print("✅ Go2 SDK 初始化成功")
                # 让机器人站起来
                self.sport_client.StandUp()
                time.sleep(2)
                print("🤖 机器人已站立")
            else:
                print(f"❌ Go2 SDK 初始化失败，错误码: {ret}")
                self.sport_client = None
                
        except Exception as e:
            print(f"❌ Go2 SDK 初始化异常: {e}")
            self.sport_client = None
    
    def on_message(self, ws, message):
        """处理接收到的消息"""
        try:
            data = json.loads(message)
            print(f"📨 收到消息: {json.dumps(data, indent=2, ensure_ascii=False)}")
            
            # 处理控制命令
            if data.get("command") == "CMD_CONTROL_ROBOT":
                self.handle_control_command(data)
            
        except json.JSONDecodeError:
            print(f"📨 收到原始消息: {message}")
    
    def handle_control_command(self, message):
        """处理控制命令"""
        try:
            # 解析data字段
            if isinstance(message.get("data"), str):
                data = json.loads(message["data"])
            else:
                data = message.get("data", {})
            
            action = data.get("action")
            params = data.get("params", {})
            
            print(f"🎮 处理控制命令: {action}")
            
            if action == "Move":
                self.handle_move_command(params)
            elif action == "shoot":
                self.handle_shoot_command()
            elif action == "raise":
                self.handle_raise_command()
            elif action == "lower":
                self.handle_lower_command()
            else:
                print(f"⚠️  未知的控制命令: {action}")
                
        except Exception as e:
            print(f"❌ 处理控制命令异常: {e}")
    
    def handle_move_command(self, params):
        """处理移动命令"""
        try:
            # 解析速度参数
            vx = float(params.get("vx", 0))
            vy = float(params.get("vy", 0))
            vyaw = float(params.get("vyaw", 0))
            
            print(f"🚶 移动命令: vx={vx:.3f}, vy={vy:.3f}, vyaw={vyaw:.3f}")
            
            # 更新机器人状态
            self.robot_state.status = "moving"
            self.last_move_time = time.time()
            
            # 执行移动
            if self.sport_client:
                # 使用Go2 SDK执行移动
                if vyaw == 0 and vy == 0 and vx == 0:
                    self.sport_client.StopMove()
                else:
                    self.sport_client.Move(vx, vy, vyaw)
                print(f"🤖 机器人执行移动: vx={vx}, vy={vy}, vyaw={vyaw}")
            else:
                # 模拟模式
                print(f"🎭 模拟模式 - 机器人移动: vx={vx}, vy={vy}, vyaw={vyaw}")
                
        except Exception as e:
            print(f"❌ 处理移动命令异常: {e}")
    
    def handle_shoot_command(self):
        """处理射击命令"""
        print("🎯 执行射击命令")
        if self.sport_client:
            # 这里可以添加具体的射击动作
            print("🤖 机器人执行射击动作")
        else:
            print("🎭 模拟模式 - 机器人射击")
    
    def handle_raise_command(self):
        """处理升高命令"""
        print("⬆️  执行升高命令")
        if self.sport_client:
            # 这里可以添加具体的升高动作
            print("🤖 机器人执行升高动作")
            self.sport_client.StandUp()
        else:
            print("🎭 模拟模式 - 机器人升高")
    
    def handle_lower_command(self):
        """处理降低命令"""
        print("⬇️  执行降低命令")
        if self.sport_client:
            # 这里可以添加具体的降低动作
            print("🤖 机器人执行降低动作")
            self.sport_client.StandDown()
        else:
            print("🎭 模拟模式 - 机器人降低")
    
    def check_move_timeout(self):
        """检查移动超时，如果超时则停止移动"""
        if (self.robot_state.status == "moving" and 
            time.time() - self.last_move_time > self.move_timeout):
            
            print("⏰ 移动超时，停止机器人")
            if self.sport_client:
                self.sport_client.StopMove()
            self.robot_state.status = "idle"
    
    def on_error(self, ws, error):
        """处理错误"""
        print(f"❌ WebSocket错误: {error}")
        self.connected = False
    
    def on_close(self, ws, close_status_code, close_msg):
        """处理连接关闭"""
        print(f"🔌 连接关闭: {close_status_code} - {close_msg}")
        self.connected = False
        
        # 停止机器人
        if self.sport_client:
            self.sport_client.StopMove()
            self.sport_client.StandDown()
            print("🤖 机器人已停止并蹲下")
    
    def on_open(self, ws):
        """处理连接打开"""
        print(f"🔗 连接到服务器: {self.server_url}")
        self.connected = True
        
        # 发送注册消息
        register_msg = {
            "type": "Request",
            "command": "CMD_REGISTER",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "name": f"Go2机器人_{self.ucode}",
                "robot_type": "go2",
                "capabilities": ["move", "shoot", "raise", "lower"]
            }
        }
        ws.send(json.dumps(register_msg))
        print(f"📤 发送注册消息: {json.dumps(register_msg, ensure_ascii=False)}")
        self.sequence += 1
    
    def connect(self):
        """连接到服务器"""
        print(f"🤖 机器人 {self.ucode} 开始连接...")
        
        # 创建WebSocket连接
        self.ws = websocket.WebSocketApp(
            self.server_url,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        
        # 在新线程中运行WebSocket
        wst = threading.Thread(target=self.ws.run_forever)
        wst.daemon = True
        wst.start()
        
        # 等待连接建立
        timeout = 10
        start_time = time.time()
        while not self.connected and time.time() - start_time < timeout:
            time.sleep(0.1)
        
        if not self.connected:
            print("❌ 连接超时")
            return False
        
        return True
    
    def disconnect(self):
        """断开连接"""
        if self.ws:
            self.ws.close()
        self.connected = False
        
        # 停止机器人
        if self.sport_client:
            self.sport_client.StopMove()
            self.sport_client.StandDown()
        
        print(f"🔌 机器人 {self.ucode} 断开连接")
    
    def send_ping(self):
        """发送ping消息"""
        if self.connected and self.ws:
            ping_msg = {
                "type": "Request",
                "command": "CMD_PING",
                "sequence": self.sequence,
                "ucode": self.ucode,
                "client_type": "robot",
                "version": "1.0.0",
                "data": {
                    "timestamp": int(time.time() * 1000)
                }
            }
            self.ws.send(json.dumps(ping_msg))
            print(f"📤 发送ping消息: {json.dumps(ping_msg, ensure_ascii=False)}")
            self.sequence += 1
    
    def send_status_update(self):
        """发送状态更新"""
        if self.connected and self.ws:
            # 更新机器人状态
            self.check_move_timeout()
            
            status_msg = {
                "type": "Request",
                "command": "CMD_UPDATE_ROBOT_STATUS",
                "sequence": self.sequence,
                "ucode": self.ucode,
                "client_type": "robot",
                "version": "1.0.0",
                "data": {
                    "status": self.robot_state.status,
                    "battery_level": self.robot_state.battery_level,
                    "temperature": self.robot_state.temperature,
                    "base_position": self.robot_state.base_position,
                    "base_orientation": self.robot_state.base_orientation,
                    "error_code": self.robot_state.error_code,
                    "error_message": self.robot_state.error_message,
                    "timestamp": int(time.time() * 1000)
                }
            }
            self.ws.send(json.dumps(status_msg))
            print(f"📤 发送状态更新: {json.dumps(status_msg, ensure_ascii=False)}")
            self.sequence += 1
    
    def keep_alive(self, interval=30):
        """保持连接活跃"""
        while self.connected:
            time.sleep(interval)
            if self.connected:
                self.send_ping()
    
    def status_update_loop(self, interval=5):
        """状态更新循环"""
        while self.connected:
            time.sleep(interval)
            if self.connected:
                self.send_status_update()

def main():
    if len(sys.argv) < 2:
        print("使用方法: python3 test_robot_real.py <ucode> [network_interface]")
        print("示例: python3 test_robot_real.py robot_001")
        print("示例: python3 test_robot_real.py robot_001 eth0")
        sys.exit(1)
    
    ucode = sys.argv[1]
    network_interface = sys.argv[2] if len(sys.argv) > 2 else "lo"
    
    print(f"🤖 启动机器人客户端")
    print(f"   UCode: {ucode}")
    print(f"   网络接口: {network_interface}")
    print(f"   Go2 SDK: {'可用' if GO2_SDK_AVAILABLE else '不可用'}")
    
    robot = RobotClient(ucode, network_interface=network_interface)
    
    try:
        # 连接服务器
        if robot.connect():
            print(f"✅ 机器人 {ucode} 连接成功")
            
            # 启动保活线程
            keep_alive_thread = threading.Thread(target=robot.keep_alive)
            keep_alive_thread.daemon = True
            keep_alive_thread.start()
            
            # 启动状态更新线程
            status_thread = threading.Thread(target=robot.status_update_loop)
            status_thread.daemon = True
            status_thread.start()
            
            # 发送初始状态
            time.sleep(2)
            robot.send_status_update()
            
            # 保持运行
            print(f"🔄 机器人 {ucode} 运行中... (按 Ctrl+C 退出)")
            print(f"📱 可以通过移动端控制机器人")
            
            try:
                while robot.connected:
                    time.sleep(1)
            except KeyboardInterrupt:
                print("\n🛑 收到退出信号")
        
        else:
            print(f"❌ 机器人 {ucode} 连接失败")
    
    except Exception as e:
        print(f"❌ 异常: {e}")
    
    finally:
        robot.disconnect()

if __name__ == "__main__":
    main() 