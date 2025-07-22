# GPIO蜂鸣器控制功能

## 概述

这个模块使用 `github.com/warthog618/go-gpiocdev` 库来控制树莓派GPIO引脚上的蜂鸣器。这是一个现代的GPIO库，使用Linux的GPIO字符设备接口。

## 主要特性

- **现代API**: 使用Linux GPIO字符设备接口
- **高性能**: 直接访问GPIO硬件
- **错误处理**: 完善的错误处理和日志记录
- **资源管理**: 自动清理GPIO资源
- **并发安全**: 支持并发操作

## 文件结构

1. `robot/gpio_controller.go` - GPIO控制器实现
2. `test_gpio.go` - 测试程序

## 使用方法

### 基本使用

```go
import "remote-ctrl-robot/cmd/client/robot"

// 创建GPIO控制器
gpioCtrl := robot.NewGPIOController(27)

// 打开蜂鸣器
gpioCtrl.BuzzerOn()

// 关闭蜂鸣器
gpioCtrl.BuzzerOff()

// 启动蜂鸣器模式
gpioCtrl.StartBuzzerPattern()

// 停止蜂鸣器模式
gpioCtrl.StopBuzzerPattern()

// 清理资源
gpioCtrl.Cleanup()
```

### 测试程序

```bash
# 编译测试程序
cd cmd/client
go build -o test_gpio test_gpio.go

# 运行测试程序
sudo ./test_gpio
```

## 权限要求

### 方法1: 使用sudo运行

```bash
sudo ./test_gpio
```

### 方法2: 添加用户到gpio组

```bash
# 添加用户到gpio组
sudo usermod -a -G gpio $USER

# 重新登录或重启会话
newgrp gpio
```

### 方法3: 设置udev规则 (推荐)

创建文件 `/etc/udev/rules.d/99-gpio.rules`:

```bash
SUBSYSTEM=="gpio", GROUP="gpio", MODE="0660"
KERNEL=="gpiochip*", ACTION=="add", PROGRAM="/bin/sh -c 'chown root:gpio /sys/class/gpio/export /sys/class/gpio/unexport ; chmod 220 /sys/class/gpio/export /sys/class/gpio/unexport'"
KERNEL=="gpiochip*", ACTION=="add", PROGRAM="/bin/sh -c 'chown root:gpio /sys/devices/virtual/gpio/gpiochip*/ ; chmod 770 /sys/devices/virtual/gpio/gpiochip*/ ; chown root:gpio /sys/devices/virtual/gpio/gpiochip*/export /sys/devices/virtual/gpio/gpiochip*/unexport ; chmod 220 /sys/devices/virtual/gpio/gpiochip*/export /sys/devices/virtual/gpio/gpiochip*/unexport'"
KERNEL=="gpio*", ACTION=="add", PROGRAM="/bin/sh -c 'chown root:gpio /sys/devices/virtual/gpio/%k/active_low /sys/devices/virtual/gpio/%k/direction /sys/devices/virtual/gpio/%k/edge /sys/devices/virtual/gpio/%k/value ; chmod 660 /sys/devices/virtual/gpio/%k/active_low /sys/devices/virtual/gpio/%k/direction /sys/devices/virtual/gpio/%k/edge /sys/devices/virtual/gpio/%k/value'"
```

然后重新加载规则：

```bash
sudo udevadm control --reload-rules
sudo udevadm trigger
```

## 系统要求

- **Linux内核**: 4.8+
- **硬件**: 支持GPIO的设备 (如树莓派)
- **权限**: GPIO访问权限

## 错误解决

### 常见错误

1. **"no such file or directory"**
   - 确保内核版本 >= 4.8
   - 检查GPIO字符设备是否可用

2. **"permission denied"**
   - 使用sudo运行
   - 或添加用户到gpio组
   - 检查udev规则设置

3. **"invalid argument"**
   - 确保在支持GPIO的硬件上运行
   - 检查引脚号是否正确

## 硬件连接

将蜂鸣器连接到GPIO引脚27：

```
蜂鸣器正极 -> GPIO 27
蜂鸣器负极 -> GND
```

## 注意事项

1. **硬件要求**: 需要在支持GPIO的硬件上运行
2. **权限管理**: 确保适当的权限设置
3. **资源清理**: 程序退出时记得调用Cleanup()
4. **并发安全**: 支持并发操作
5. **错误处理**: 始终检查返回的错误

## 扩展功能

可以轻松扩展支持：

- 不同的GPIO引脚
- 其他GPIO设备（LED、按钮等）
- 更复杂的蜂鸣器模式
- PWM控制
- 事件监听 