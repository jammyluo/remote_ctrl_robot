#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
游戏客户端测试脚本
用于测试机器人对抗游戏功能
"""

import asyncio
import websockets
import json
import time
import random
from datetime import datetime

class GameClient:
    def __init__(self, ucode, name, server_url="ws://localhost:8080/ws/control"):
        self.ucode = ucode
        self.name = name
        self.server_url = server_url
        self.websocket = None
        self.connected = False
        self.in_game = False
        self.game_id = "test_game"
        self.sequence = 1
        
    async def connect(self):
        """连接服务器"""
        try:
            self.websocket = await websockets.connect(self.server_url)
            self.connected = True
            print(f"[{self.ucode}] 连接服务器成功")
            
            # 发送注册消息
            await self.register()
            
            # 启动消息接收
            asyncio.create_task(self.receive_messages())
            
        except Exception as e:
            print(f"[{self.ucode}] 连接失败: {e}")
            
    async def register(self):
        """注册机器人"""
        message = {
            "type": "Request",
            "command": "CMD_REGISTER",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "name": self.name
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        print(f"[{self.ucode}] 发送注册消息")
        
    async def join_game(self):
        """加入游戏"""
        message = {
            "type": "Request",
            "command": "CMD_JOIN_GAME",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "game_id": self.game_id,
                "name": self.name
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        print(f"[{self.ucode}] 发送加入游戏请求")
        
    async def leave_game(self):
        """离开游戏"""
        message = {
            "type": "Request",
            "command": "CMD_LEAVE_GAME",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "game_id": self.game_id
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        print(f"[{self.ucode}] 发送离开游戏请求")
        
    async def shoot(self, target_x, target_y, target_z=0):
        """射击"""
        message = {
            "type": "Request",
            "command": "CMD_GAME_SHOOT",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "target_x": target_x,
                "target_y": target_y,
                "target_z": target_z
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        print(f"[{self.ucode}] 射击: ({target_x}, {target_y}, {target_z})")
        
    async def move(self, x, y, direction=0):
        """移动"""
        message = {
            "type": "Request",
            "command": "CMD_GAME_MOVE",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "position": {
                    "x": x,
                    "y": y,
                    "z": 0
                },
                "direction": direction
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        print(f"[{self.ucode}] 移动: ({x}, {y}) 朝向: {direction}")
        
    async def get_game_status(self):
        """获取游戏状态"""
        message = {
            "type": "Request",
            "command": "CMD_GAME_STATUS",
            "sequence": self.sequence,
            "ucode": self.ucode,
            "client_type": "robot",
            "version": "1.0.0",
            "data": {
                "game_id": self.game_id
            }
        }
        
        await self.websocket.send(json.dumps(message))
        self.sequence += 1
        
    async def receive_messages(self):
        """接收消息"""
        try:
            async for message in self.websocket:
                data = json.loads(message)
                await self.handle_message(data)
        except Exception as e:
            print(f"[{self.ucode}] 接收消息错误: {e}")
            self.connected = False
            
    async def handle_message(self, message):
        """处理接收到的消息"""
        command = message.get("command")
        
        if command == "CMD_REGISTER":
            if message.get("data", {}).get("success"):
                print(f"[{self.ucode}] 注册成功")
                # 注册成功后加入游戏
                await asyncio.sleep(1)
                await self.join_game()
            else:
                print(f"[{self.ucode}] 注册失败: {message.get('data', {}).get('message')}")
                
        elif command == "CMD_JOIN_GAME":
            if message.get("data", {}).get("success"):
                print(f"[{self.ucode}] 加入游戏成功")
                self.in_game = True
            else:
                print(f"[{self.ucode}] 加入游戏失败: {message.get('data', {}).get('message')}")
                
        elif command == "CMD_GAME_STATUS":
            data = message.get("data", {})
            if data:
                game_state = data.get("game_state")
                my_robot = data.get("my_robot")
                if my_robot:
                    print(f"[{self.ucode}] 状态更新 - 血量: {my_robot.get('health')}/{my_robot.get('max_health')}, "
                          f"得分: {my_robot.get('score')}, 存活: {my_robot.get('is_alive')}")
                    
    async def game_loop(self):
        """游戏主循环"""
        while self.connected and self.in_game:
            try:
                # 随机移动
                x = random.uniform(-50, 50)
                y = random.uniform(-50, 50)
                direction = random.uniform(0, 6.28)
                await self.move(x, y, direction)
                
                await asyncio.sleep(2)
                
                # 随机射击
                target_x = random.uniform(-50, 50)
                target_y = random.uniform(-50, 50)
                await self.shoot(target_x, target_y)
                
                await asyncio.sleep(3)
                
                # 获取游戏状态
                await self.get_game_status()
                
                await asyncio.sleep(1)
                
            except Exception as e:
                print(f"[{self.ucode}] 游戏循环错误: {e}")
                break
                
    async def run(self):
        """运行客户端"""
        await self.connect()
        
        if self.connected:
            # 启动游戏循环
            await self.game_loop()
            
        # 清理
        if self.websocket:
            await self.websocket.close()

async def main():
    """主函数"""
    print("启动游戏客户端测试...")
    
    # 创建多个机器人客户端
    clients = [
        GameClient("robot_001", "机器人001"),
        GameClient("robot_002", "机器人002"),
        GameClient("robot_003", "机器人003"),
    ]
    
    # 并发运行所有客户端
    tasks = [client.run() for client in clients]
    await asyncio.gather(*tasks, return_exceptions=True)
    
    print("测试完成")

if __name__ == "__main__":
    asyncio.run(main()) 