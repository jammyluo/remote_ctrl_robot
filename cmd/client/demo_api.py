#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
机器人客户端API演示脚本
演示如何使用HTTP API接口
"""

import requests
import json
import time
import sys

# API基础URL
BASE_URL = "http://localhost:8080/api/v1"

def print_response(response, title):
    """打印API响应"""
    print(f"\n{'='*50}")
    print(f" {title}")
    print(f"{'='*50}")
    print(f"状态码: {response.status_code}")
    print(f"响应内容:")
    try:
        data = response.json()
        print(json.dumps(data, indent=2, ensure_ascii=False))
    except:
        print(response.text)
    print()

def test_health():
    """测试健康检查"""
    try:
        response = requests.get(f"{BASE_URL}/health")
        print_response(response, "健康检查")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器，请确保机器人客户端正在运行")
        return False

def test_get_name():
    """测试获取名称"""
    try:
        response = requests.get(f"{BASE_URL}/name")
        print_response(response, "获取名称")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_set_name(name):
    """测试设置名称"""
    try:
        data = {"name": name}
        response = requests.post(f"{BASE_URL}/name", json=data)
        print_response(response, f"设置名称: {name}")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_get_status():
    """测试获取状态"""
    try:
        response = requests.get(f"{BASE_URL}/status")
        print_response(response, "获取状态")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_invalid_requests():
    """测试无效请求"""
    print(f"\n{'='*50}")
    print(" 测试无效请求")
    print(f"{'='*50}")
    
    # 测试空名称
    try:
        data = {"name": ""}
        response = requests.post(f"{BASE_URL}/name", json=data)
        print(f"空名称测试 - 状态码: {response.status_code}")
        print(f"响应: {response.json()}")
    except Exception as e:
        print(f"空名称测试失败: {e}")
    
    # 测试无效JSON
    try:
        response = requests.post(f"{BASE_URL}/name", 
                               data="invalid json",
                               headers={"Content-Type": "application/json"})
        print(f"无效JSON测试 - 状态码: {response.status_code}")
        print(f"响应: {response.json()}")
    except Exception as e:
        print(f"无效JSON测试失败: {e}")
    
    # 测试不支持的方法
    try:
        response = requests.put(f"{BASE_URL}/name")
        print(f"不支持方法测试 - 状态码: {response.status_code}")
        print(f"响应: {response.json()}")
    except Exception as e:
        print(f"不支持方法测试失败: {e}")
    
    print()

def interactive_mode():
    """交互模式"""
    print("\n🎮 进入交互模式")
    print("输入 'help' 查看可用命令")
    print("输入 'quit' 退出")
    
    while True:
        try:
            command = input("\n请输入命令: ").strip().lower()
            
            if command == 'quit' or command == 'exit':
                print("👋 再见!")
                break
            elif command == 'help':
                print("可用命令:")
                print("  health    - 健康检查")
                print("  name      - 获取名称")
                print("  status    - 获取状态")
                print("  set <name> - 设置名称")
                print("  quit      - 退出")
            elif command == 'health':
                test_health()
            elif command == 'name':
                test_get_name()
            elif command == 'status':
                test_get_status()
            elif command.startswith('set '):
                name = command[4:].strip()
                if name:
                    test_set_name(name)
                else:
                    print("❌ 请提供名称")
            else:
                print("❌ 未知命令，输入 'help' 查看可用命令")
                
        except KeyboardInterrupt:
            print("\n👋 再见!")
            break
        except Exception as e:
            print(f"❌ 错误: {e}")

def main():
    """主函数"""
    print("🤖 机器人客户端API演示")
    print(f"API地址: {BASE_URL}")
    
    # 检查服务是否可用
    if not test_health():
        print("\n💡 提示:")
        print("1. 确保机器人客户端正在运行")
        print("2. 检查端口8080是否被占用")
        print("3. 运行: ./robot_client")
        return
    
    # 运行基本测试
    print("\n🚀 开始基本测试...")
    
    test_get_name()
    test_set_name("Python测试机器人")
    test_get_name()
    test_set_name("我的智能机器人")
    test_get_name()
    test_get_status()
    test_invalid_requests()
    
    # 询问是否进入交互模式
    try:
        choice = input("\n是否进入交互模式? (y/n): ").strip().lower()
        if choice in ['y', 'yes', '是']:
            interactive_mode()
    except KeyboardInterrupt:
        print("\n👋 再见!")

if __name__ == "__main__":
    main() 