package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"remote-ctrl-robot/cmd/client/robot"
)

func main() {
	fmt.Println("GPIO蜂鸣器测试程序")
	fmt.Println("按Ctrl+C退出")

	// 创建GPIO控制器
	gpioCtrl := robot.NewGPIOController(27)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动蜂鸣器模式
	fmt.Println("启动蜂鸣器模式...")
	gpioCtrl.StartBuzzerPattern()

	// 等待信号
	<-sigChan

	fmt.Println("\n收到退出信号，正在停止...")

	// 停止蜂鸣器模式
	gpioCtrl.StopBuzzerPattern()

	// 清理资源
	if err := gpioCtrl.Cleanup(); err != nil {
		fmt.Printf("清理GPIO资源失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("GPIO资源清理完成")
} 