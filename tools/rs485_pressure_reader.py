#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
RS485压力变送器数据读取程序
基于STC89C52RC单片机的通信协议实现
每隔50ms读取一次数据并写入文件
"""

import serial
import time
import struct
import csv
from datetime import datetime
import logging
import threading
from typing import Optional, Tuple

class RS485PressureReader:
    """RS485压力变送器读取器"""
    
    def __init__(self, port: str = '/dev/tty.usbserial-2130', baudrate: int = 9600, 
                 timeout: float = 1.0, data_file: str = 'pressure_data.csv'):
        """
        初始化RS485读取器
        
        Args:
            port: 串口设备名
            baudrate: 波特率
            timeout: 超时时间
            data_file: 数据文件路径
        """
        self.port = port
        self.baudrate = baudrate
        self.timeout = timeout
        self.data_file = data_file
        self.serial_port: Optional[serial.Serial] = None
        self.running = False
        self.lock = threading.Lock()
        
        # 设置日志
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler('rs485_reader.log'),
                logging.StreamHandler()
            ]
        )
        self.logger = logging.getLogger(__name__)
        
        # Modbus RTU查询指令 (基于原C代码)
        self.query_command = bytes([0x01, 0x03, 0x00, 0x00, 0x00, 0x01, 0x84, 0x0A])
        
        # 初始化CSV文件
        self._init_csv_file()
    
    def _init_csv_file(self):
        """初始化CSV文件，写入表头"""
        try:
            with open(self.data_file, 'w', newline='', encoding='utf-8') as csvfile:
                writer = csv.writer(csvfile)
                writer.writerow(['时间戳', '压力值(g)', '原始数据(hex)', 'CRC校验状态'])
            self.logger.info(f"CSV文件已初始化: {self.data_file}")
        except Exception as e:
            self.logger.error(f"初始化CSV文件失败: {e}")
    
    def _crc16_modbus(self, data: bytes) -> int:
        """
        计算Modbus CRC16校验码
        基于原C代码的CRC算法实现
        """
        crc = 0xFFFF
        for byte in data:
            crc ^= byte
            for _ in range(8):
                if crc & 0x0001:
                    crc = (crc >> 1) ^ 0xA001
                else:
                    crc >>= 1
        return crc
    
    def _send_query(self) -> bool:
        """
        发送查询指令
        
        Returns:
            bool: 发送是否成功
        """
        try:
            if self.serial_port and self.serial_port.is_open:
                # 清空接收缓冲区
                self.serial_port.reset_input_buffer()
                
                # 发送查询指令
                self.serial_port.write(self.query_command)
                self.serial_port.flush()
                
                self.logger.debug(f"发送查询指令: {self.query_command.hex()}")
                return True
            else:
                self.logger.error("串口未打开")
                return False
        except Exception as e:
            self.logger.error(f"发送查询指令失败: {e}")
            return False
    
    def _receive_response(self) -> Optional[bytes]:
        """
        接收响应数据
        
        Returns:
            bytes: 响应数据，如果接收失败返回None
        """
        try:
            if not self.serial_port or not self.serial_port.is_open:
                return None
            
            # 等待响应数据
            response = self.serial_port.read(7)  # 期望接收7字节
            
            if len(response) == 7:
                self.logger.debug(f"接收响应: {response.hex()}")
                return response
            else:
                self.logger.warning(f"接收数据长度不正确: {len(response)} 字节")
                return None
                
        except Exception as e:
            self.logger.error(f"接收响应失败: {e}")
            return None
    
    def _parse_pressure_data(self, response: bytes) -> Tuple[bool, int, str]:
        """
        解析压力数据
        
        Args:
            response: 响应数据
            
        Returns:
            Tuple[bool, int, str]: (CRC校验通过, 压力值, 状态信息)
        """
        try:
            # 检查帧头 (01 03 02)
            if response[0] != 0x01 or response[1] != 0x03 or response[2] != 0x02:
                return False, 0, "帧头错误"
            
            # CRC校验
            data_for_crc = response[:5]  # 前5字节用于CRC计算
            received_crc = struct.unpack('<H', response[5:7])[0]  # 小端序
            calculated_crc = self._crc16_modbus(data_for_crc)
            
            if received_crc != calculated_crc:
                return False, 0, f"CRC校验失败 (接收:{received_crc:04X}, 计算:{calculated_crc:04X})"
            
            # 解析压力值 (第3、4字节)
            pressure = (response[3] << 8) + response[4]
            
            return True, pressure, "数据有效"
            
        except Exception as e:
            return False, 0, f"数据解析错误: {e}"
    
    def _write_to_csv(self, pressure: int, raw_data: str, crc_status: str):
        """
        将数据写入CSV文件
        
        Args:
            pressure: 压力值
            raw_data: 原始数据(十六进制)
            crc_status: CRC校验状态
        """
        try:
            with self.lock:
                with open(self.data_file, 'a', newline='', encoding='utf-8') as csvfile:
                    writer = csv.writer(csvfile)
                    timestamp = datetime.now().strftime('%Y-%m-%d %H:%M:%S.%f')[:-3]
                    writer.writerow([timestamp, pressure, raw_data, crc_status])
        except Exception as e:
            self.logger.error(f"写入CSV文件失败: {e}")
    
    def open_serial(self) -> bool:
        """
        打开串口连接
        
        Returns:
            bool: 连接是否成功
        """
        try:
            self.serial_port = serial.Serial(
                port=self.port,
                baudrate=self.baudrate,
                timeout=self.timeout,
                bytesize=serial.EIGHTBITS,
                parity=serial.PARITY_NONE,
                stopbits=serial.STOPBITS_ONE
            )
            
            if self.serial_port.is_open:
                self.logger.info(f"串口连接成功: {self.port}")
                return True
            else:
                self.logger.error("串口打开失败")
                return False
                
        except Exception as e:
            self.logger.error(f"串口连接失败: {e}")
            return False
    
    def close_serial(self):
        """关闭串口连接"""
        if self.serial_port and self.serial_port.is_open:
            self.serial_port.close()
            self.logger.info("串口连接已关闭")
    
    def read_pressure_once(self) -> Optional[int]:
        """
        读取一次压力数据
        
        Returns:
            int: 压力值，如果读取失败返回None
        """
        # 发送查询指令
        if not self._send_query():
            return None
        
        # 短暂延时，等待设备响应
        time.sleep(0.01)
        
        # 接收响应
        response = self._receive_response()
        if not response:
            return None
        
        # 解析数据
        crc_ok, pressure, status = self._parse_pressure_data(response)
        
        # 记录日志
        if crc_ok:
            self.logger.info(f"压力值: {pressure} g")
        else:
            self.logger.warning(f"数据无效: {status}")
        
        # 写入CSV文件
        self._write_to_csv(pressure, response.hex(), status)
        
        return pressure if crc_ok else None
    
    def start_continuous_reading(self, interval_ms: int = 50):
        """
        开始连续读取数据
        
        Args:
            interval_ms: 读取间隔(毫秒)
        """
        if not self.open_serial():
            self.logger.error("无法启动连续读取，串口连接失败")
            return
        
        self.running = True
        self.logger.info(f"开始连续读取，间隔: {interval_ms}ms")
        
        try:
            while self.running:
                start_time = time.time()
                
                # 读取压力数据
                pressure = self.read_pressure_once()
                
                # 计算延时时间
                elapsed = (time.time() - start_time) * 1000  # 转换为毫秒
                sleep_time = max(0, (interval_ms - elapsed) / 1000)  # 转换为秒
                
                if sleep_time > 0:
                    time.sleep(sleep_time)
                    
        except KeyboardInterrupt:
            self.logger.info("用户中断，停止读取")
        except Exception as e:
            self.logger.error(f"连续读取过程中发生错误: {e}")
        finally:
            self.stop_continuous_reading()
    
    def stop_continuous_reading(self):
        """停止连续读取"""
        self.running = False
        self.close_serial()
        self.logger.info("连续读取已停止")


def main():
    """主函数"""
    import argparse
    
    parser = argparse.ArgumentParser(description='RS485压力变送器数据读取程序')
    parser.add_argument('--port', default='/dev/tty.usbserial-2130', help='串口设备名')
    parser.add_argument('--baudrate', type=int, default=9600, help='波特率')
    parser.add_argument('--interval', type=int, default=50, help='读取间隔(毫秒)')
    parser.add_argument('--file', default='pressure_data.csv', help='数据文件路径')
    parser.add_argument('--timeout', type=float, default=1.0, help='串口超时时间')
    
    args = parser.parse_args()
    
    # 创建读取器实例
    reader = RS485PressureReader(
        port=args.port,
        baudrate=args.baudrate,
        timeout=args.timeout,
        data_file=args.file
    )
    
    print(f"RS485压力变送器数据读取程序")
    print(f"串口: {args.port}")
    print(f"波特率: {args.baudrate}")
    print(f"读取间隔: {args.interval}ms")
    print(f"数据文件: {args.file}")
    print("按 Ctrl+C 停止程序")
    print("-" * 50)
    
    try:
        # 开始连续读取
        reader.start_continuous_reading(args.interval)
    except KeyboardInterrupt:
        print("\n程序已停止")


if __name__ == "__main__":
    main() 