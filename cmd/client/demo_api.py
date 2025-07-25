#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
机器人客户端API演示脚本
演示如何使用HTTP API接口控制机器人
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
    """测试生命值查询"""
    try:
        response = requests.get(f"{BASE_URL}/health")
        print_response(response, "生命值查询")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器，请确保机器人客户端正在运行")
        return False

def test_shoot():
    """测试射击"""
    try:
        response = requests.post(f"{BASE_URL}/shoot")
        print_response(response, "执行射击")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_get_ammo():
    """测试获取弹药数量"""
    try:
        response = requests.get(f"{BASE_URL}/ammo")
        print_response(response, "弹药数量查询")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_change_ammo():
    """测试更换弹药"""
    try:
        response = requests.post(f"{BASE_URL}/ammo/change")
        print_response(response, "更换弹药")
        return response.status_code == 200
    except requests.exceptions.ConnectionError:
        print("❌ 无法连接到API服务器")
        return False

def test_invalid_requests():
    """测试无效请求"""
    print(f"\n{'='*50}")
    print(" 测试无效请求")
    print(f"{'='*50}")
    
    # 测试不支持的方法
    try:
        response = requests.get(f"{BASE_URL}/shoot")
        print(f"射击接口GET方法测试 - 状态码: {response.status_code}")
        print(f"响应: {response.json()}")
    except Exception as e:
        print(f"射击接口GET方法测试失败: {e}")
    
    # 测试不存在的接口
    try:
        response = requests.get(f"{BASE_URL}/nonexistent")
        print(f"不存在接口测试 - 状态码: {response.status_code}")
        print(f"响应: {response.text}")
    except Exception as e:
        print(f"不存在接口测试失败: {e}")
    
    # 测试错误的请求方法
    try:
        response = requests.put(f"{BASE_URL}/ammo")
        print(f"弹药接口PUT方法测试 - 状态码: {response.status_code}")
        print(f"响应: {response.json()}")
    except Exception as e:
        print(f"弹药接口PUT方法测试失败: {e}")
    
    print()

def test_robot_sequence():
    """测试机器人操作序列"""
    print(f"\n{'='*50}")
    print(" 机器人操作序列测试")
    print(f"{'='*50}")
    
    print("1. 查询初始状态...")
    test_get_ammo()
    test_health()
    
    print("2. 执行射击操作...")
    for i in range(3):
        print(f"   第{i+1}次射击:")
        test_shoot()
        time.sleep(1)  # 等待1秒
    
    print("3. 查询射击后状态...")
    test_get_ammo()
    test_health()
    
    print("4. 更换弹药...")
    test_change_ammo()
    
    print("5. 查询更换后状态...")
    test_get_ammo()
    test_health()
    
    print("6. 再次射击测试...")
    test_shoot()
    test_get_ammo()

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
                print("  health     - 查询生命值")
                print("  ammo       - 查询弹药数量")
                print("  shoot      - 执行射击")
                print("  change     - 更换弹药")
                print("  sequence   - 执行操作序列")
                print("  status     - 查询完整状态")
                print("  quit       - 退出")
            elif command == 'health':
                test_health()
            elif command == 'ammo':
                test_get_ammo()
            elif command == 'shoot':
                test_shoot()
            elif command == 'change':
                test_change_ammo()
            elif command == 'sequence':
                test_robot_sequence()
            elif command == 'status':
                print("查询完整状态...")
                test_health()
                test_get_ammo()
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
    print("功能: 射击、弹药管理、生命值查询")
    
    # 检查服务是否可用
    if not test_health():
        print("\n💡 提示:")
        print("1. 确保机器人客户端正在运行")
        print("2. 检查端口8080是否被占用")
        print("3. 运行: ./robot_client")
        return
    
    # 运行基本测试
    print("\n🚀 开始基本测试...")
    
    # 测试所有接口
    test_get_ammo()
    test_health()
    test_shoot()
    test_change_ammo()
    
    # 测试无效请求
    test_invalid_requests()
    
    # 询问是否执行操作序列
    try:
        choice = input("\n是否执行机器人操作序列测试? (y/n): ").strip().lower()
        if choice in ['y', 'yes', '是']:
            test_robot_sequence()
    except KeyboardInterrupt:
        print("\n👋 再见!")
        return
    
    # 询问是否进入交互模式
    try:
        choice = input("\n是否进入交互模式? (y/n): ").strip().lower()
        if choice in ['y', 'yes', '是']:
            interactive_mode()
    except KeyboardInterrupt:
        print("\n👋 再见!")

if __name__ == "__main__":
    main() 