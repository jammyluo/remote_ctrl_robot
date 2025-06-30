#!/usr/bin/env python3
"""
远程控制机器人 WebSocket 客户端测试
支持 UCODE 注册和消息交互
"""

import websocket
import json
import threading
import time
import sys

# 配置
SERVER_URL = "ws://localhost:8080/ws/control"
UCODE = "123456"  # 机器人UCODE
RECONNECT_DELAY = 5  # 重连延迟（秒）

class RobotWebSocketClient:
    def __init__(self, server_url, ucode):
        self.server_url = server_url
        self.ucode = ucode
        self.ws = None
        self.connected = False
        self.registered = False
        
    def on_message(self, ws, message):
        """处理接收到的消息"""
        try:
            data = json.loads(message)
            print(f"📨 收到消息: {json.dumps(data, ensure_ascii=False, indent=2)}")
            
            # 处理不同类型的消息
            msg_type = data.get("type", "")
            if msg_type == "welcome":
                print(f"✅ 成功注册UCODE: {self.ucode}")
                self.registered = True
            elif msg_type == "error":
                print(f"❌ 错误: {data.get('message', 'Unknown error')}")
            elif msg_type == "pong":
                print("🏓 收到pong响应")
                
        except json.JSONDecodeError:
            print(f"📨 收到原始消息: {message}")

    def on_error(self, ws, error):
        """处理错误"""
        print(f"❌ WebSocket错误: {error}")
        self.connected = False

    def on_close(self, ws, close_status_code, close_msg):
        """处理连接关闭"""
        print(f"🔌 连接关闭 - 状态码: {close_status_code}, 消息: {close_msg}")
        self.connected = False
        self.registered = False

    def on_open(self, ws):
        """处理连接打开"""
        print("🔗 连接成功，发送注册消息...")
        self.connected = True
        
        # 发送注册消息
        register_msg = {
            "type": "register",
            "ucode": self.ucode
        }
        ws.send(json.dumps(register_msg))
        print(f"📤 已发送注册消息: {json.dumps(register_msg, ensure_ascii=False)}")

    def send_ping(self):
        """发送ping消息"""
        if self.ws and self.connected:
            ping_msg = {
                "type": "ping",
                "message": "heartbeat"
            }
            self.ws.send(json.dumps(ping_msg))
            print("🏓 发送ping消息")

    def send_control_command(self, command_type="joint_position"):
        """发送控制命令"""
        if not self.registered:
            print("❌ 尚未注册，无法发送控制命令")
            return
            
        if self.ws and self.connected:
            command = {
                "type": "control_command",
                "data": {
                    "type": command_type,
                    "command_id": f"cmd_{int(time.time())}",
                    "priority": 5,
                    "timestamp": int(time.time() * 1000),
                    "joint_pos": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6],
                    "velocities": [0.1, 0.1, 0.1, 0.1, 0.1, 0.1]
                }
            }
            self.ws.send(json.dumps(command))
            print(f"📤 发送控制命令: {command_type}")

    def send_status_request(self):
        """发送状态请求"""
        if self.ws and self.connected:
            status_msg = {
                "type": "status_request",
                "message": "request robot status"
            }
            self.ws.send(json.dumps(status_msg))
            print("📤 发送状态请求")

    def start_heartbeat(self):
        """启动心跳线程"""
        def heartbeat_loop():
            while self.connected:
                time.sleep(10)  # 每10秒发送一次心跳
                if self.connected:
                    self.send_ping()
        
        heartbeat_thread = threading.Thread(target=heartbeat_loop, daemon=True)
        heartbeat_thread.start()

    def start_interactive_mode(self):
        """启动交互模式"""
        def interactive_loop():
            while self.connected:
                try:
                    print("\n" + "="*50)
                    print("交互菜单:")
                    print("1. 发送关节位置控制命令")
                    print("2. 发送速度控制命令")
                    print("3. 发送紧急停止命令")
                    print("4. 发送回零命令")
                    print("5. 请求机器人状态")
                    print("6. 发送ping")
                    print("0. 退出")
                    print("="*50)
                    
                    choice = input("请选择操作 (0-6): ").strip()
                    
                    if choice == "0":
                        print("👋 退出程序")
                        if self.ws:
                            self.ws.close()
                        break
                    elif choice == "1":
                        self.send_control_command("joint_position")
                    elif choice == "2":
                        self.send_control_command("velocity")
                    elif choice == "3":
                        self.send_control_command("emergency_stop")
                    elif choice == "4":
                        self.send_control_command("home")
                    elif choice == "5":
                        self.send_status_request()
                    elif choice == "6":
                        self.send_ping()
                    else:
                        print("❌ 无效选择，请重试")
                        
                except KeyboardInterrupt:
                    print("\n👋 用户中断，退出程序")
                    if self.ws:
                        self.ws.close()
                    break
                except Exception as e:
                    print(f"❌ 交互错误: {e}")
        
        interactive_thread = threading.Thread(target=interactive_loop, daemon=True)
        interactive_thread.start()

    def connect(self):
        """连接到服务器"""
        print(f"🚀 连接到服务器: {self.server_url}")
        print(f"🤖 注册UCODE: {self.ucode}")
        
        # 创建WebSocket连接
        self.ws = websocket.WebSocketApp(
            self.server_url,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )
        
        # 启动心跳
        self.start_heartbeat()
        
        # 启动交互模式
        self.start_interactive_mode()
        
        # 运行WebSocket
        self.ws.run_forever()

def main():
    """主函数"""
    print("🤖 远程控制机器人 WebSocket 客户端")
    print("="*50)
    
    # 检查命令行参数
    if len(sys.argv) > 1:
        ucode = sys.argv[1]
    else:
        ucode = UCODE
    
    if len(sys.argv) > 2:
        server_url = sys.argv[2]
    else:
        server_url = SERVER_URL
    
    print(f"服务器: {server_url}")
    print(f"UCODE: {ucode}")
    print("="*50)
    
    # 创建客户端并连接
    client = RobotWebSocketClient(server_url, ucode)
    
    try:
        client.connect()
    except KeyboardInterrupt:
        print("\n👋 程序被用户中断")
    except Exception as e:
        print(f"❌ 连接失败: {e}")
        print("💡 请确保服务器正在运行")

if __name__ == "__main__":
    main() 