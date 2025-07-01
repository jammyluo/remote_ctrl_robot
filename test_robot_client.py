#!/usr/bin/env python3
"""
机器人客户端测试脚本
模拟机器人通过WebSocket连接到服务器
"""

import websocket
import json
import time
import threading
import sys

class RobotClient:
    def __init__(self, ucode, server_url="ws://localhost:8000/ws/control"):
        self.ucode = ucode
        self.server_url = server_url
        self.ws = None
        self.connected = False
        
    def on_message(self, ws, message):
        """处理接收到的消息"""
        try:
            data = json.loads(message)
            print(f"📨 收到消息: {json.dumps(data, indent=2, ensure_ascii=False)}")
        except json.JSONDecodeError:
            print(f"📨 收到原始消息: {message}")
    
    def on_error(self, ws, error):
        """处理错误"""
        print(f"❌ WebSocket错误: {error}")
        self.connected = False
    
    def on_close(self, ws, close_status_code, close_msg):
        """处理连接关闭"""
        print(f"🔌 连接关闭: {close_status_code} - {close_msg}")
        self.connected = False
    
    def on_open(self, ws):
        """处理连接打开"""
        print(f"🔗 连接到服务器: {self.server_url}")
        self.connected = True
        
        # 发送注册消息
        register_msg = {
            "type": "register",
            "ucode": self.ucode,
            "client_type": "robot"
        }
        ws.send(json.dumps(register_msg))
        print(f"📤 发送注册消息: {json.dumps(register_msg, ensure_ascii=False)}")
    
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
        print(f"🔌 机器人 {self.ucode} 断开连接")
    
    def send_ping(self):
        """发送ping消息"""
        if self.connected and self.ws:
            ping_msg = {
                "type": "ping",
                "message": "ping"
            }
            self.ws.send(json.dumps(ping_msg))
            print(f"📤 发送ping消息")
    
    def send_status_request(self):
        """发送状态请求"""
        if self.connected and self.ws:
            status_msg = {
                "type": "status_request",
                "message": "Request robot status"
            }
            self.ws.send(json.dumps(status_msg))
            print(f"📤 发送状态请求")
    
    def keep_alive(self, interval=30):
        """保持连接活跃"""
        while self.connected:
            time.sleep(interval)
            if self.connected:
                self.send_ping()

def main():
    if len(sys.argv) < 2:
        print("使用方法: python3 test_robot_client.py <ucode>")
        print("示例: python3 test_robot_client.py robot_001")
        sys.exit(1)
    
    ucode = sys.argv[1]
    robot = RobotClient(ucode)
    
    try:
        # 连接服务器
        if robot.connect():
            print(f"✅ 机器人 {ucode} 连接成功")
            
            # 启动保活线程
            keep_alive_thread = threading.Thread(target=robot.keep_alive)
            keep_alive_thread.daemon = True
            keep_alive_thread.start()
            
            # 发送状态请求
            time.sleep(2)
            robot.send_status_request()
            
            # 保持运行
            print(f"🔄 机器人 {ucode} 运行中... (按 Ctrl+C 退出)")
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