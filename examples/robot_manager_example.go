package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"remote-ctrl-robot/internal/models"
	"remote-ctrl-robot/internal/services"
)

func main() {
	// 创建机器人管理器
	robotManager := services.NewRobotManager()

	// 启动管理器
	if err := robotManager.Start(); err != nil {
		log.Fatalf("Failed to start robot manager: %v", err)
	}
	defer robotManager.Stop()

	// 注册机器人
	robot1 := &models.RobotRegistration{
		UCode:        "robot_b2_001",
		Name:         "B2机器人1",
		Type:         models.RobotTypeB2,
		Version:      "1.0",
		IPAddress:    "192.168.1.100",
		Port:         8080,
		Capabilities: []string{"move", "stop", "reset", "stand", "sit"},
	}

	robot2 := &models.RobotRegistration{
		UCode:        "robot_go2_001",
		Name:         "Go2机器人1",
		Type:         models.RobotTypeGo2,
		Version:      "1.0",
		IPAddress:    "192.168.1.101",
		Port:         8080,
		Capabilities: []string{"move", "stop", "reset", "stand", "sit", "dance"},
	}

	// 注册机器人
	if _, err := robotManager.RegisterRobot(robot1); err != nil {
		log.Printf("Failed to register robot1: %v", err)
	}

	if _, err := robotManager.RegisterRobot(robot2); err != nil {
		log.Printf("Failed to register robot2: %v", err)
	}

	// 添加事件处理器
	robotManager.AddEventHandler(models.RobotEventConnected, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 连接成功: %s\n", event.UCode, event.Message)
	})

	robotManager.AddEventHandler(models.RobotEventDisconnected, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 连接断开: %s\n", event.UCode, event.Message)
	})

	robotManager.AddEventHandler(models.RobotEventError, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 发生错误: %s\n", event.UCode, event.Message)
	})

	robotManager.AddEventHandler(models.RobotEventCommand, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 执行命令: %s\n", event.UCode, event.Message)
	})

	// 监控机器人状态
	go monitorRobots(robotManager)

	// 模拟发送命令
	go sendCommands(robotManager)

	// 保持运行
	select {}
}

// monitorRobots 监控机器人状态
func monitorRobots(robotManager *services.RobotManager) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 获取所有机器人
			robots := robotManager.GetAllRobots()
			fmt.Printf("\n=== 机器人状态报告 ===\n")
			fmt.Printf("总机器人数量: %d\n", len(robots))

			for _, robot := range robots {
				status := robot.GetStatus()
				if status != nil {
					fmt.Printf("机器人 %s (%s):\n", robot.Name, robot.UCode)
					fmt.Printf("  连接状态: %t\n", status.Connected)
					fmt.Printf("  最后心跳: %s\n", status.LastHeartbeat.Format("15:04:05"))
					fmt.Printf("  延迟: %dms\n", status.Latency)
					fmt.Printf("  总命令数: %d\n", status.TotalCommands)
					fmt.Printf("  失败命令数: %d\n", status.FailedCommands)
					fmt.Printf("  在线状态: %t\n", robot.IsOnline())
					fmt.Printf("  健康状态: %t\n", robot.IsHealthy())
				}
				fmt.Println()
			}

			// 获取系统状态
			systemStatus := robotManager.GetSystemStatus()
			fmt.Printf("系统状态: %s\n", systemStatus.RobotStatus)
			fmt.Printf("活跃客户端: %d\n", systemStatus.ActiveClients)
			fmt.Printf("================================\n\n")
		}
	}
}

// sendCommands 发送命令
func sendCommands(robotManager *services.RobotManager) {
	time.Sleep(5 * time.Second) // 等待机器人连接

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	commands := []string{"move", "stop", "stand", "sit"}

	for {
		select {
		case <-ticker.C:
			// 获取在线机器人
			robots := robotManager.GetOnlineRobots()
			if len(robots) == 0 {
				fmt.Println("没有在线机器人，跳过命令发送")
				continue
			}

			// 随机选择一个命令
			command := commands[time.Now().Unix()%int64(len(commands))]

			// 发送命令到第一个在线机器人
			robotCommand := &models.RobotCommand{
				Action:        command,
				Params:        map[string]string{"speed": "0.5"},
				Priority:      5,
				Timestamp:     time.Now().Unix(),
				OperatorUCode: "example_operator",
			}

			response, err := robotManager.SendCommand(robots[0].UCode, robotCommand)
			if err != nil {
				fmt.Printf("发送命令失败: %v\n", err)
			} else {
				fmt.Printf("命令发送成功: %s -> %s, 响应: %s\n",
					command, robots[0].Name, response.Message)
			}
		}
	}
}

// 使用示例：完整的机器人管理系统
func exampleCompleteSystem() {
	// 1. 创建机器人管理器
	robotManager := services.NewRobotManager()
	defer robotManager.Stop()

	// 2. 启动管理器
	if err := robotManager.Start(); err != nil {
		log.Fatalf("Failed to start robot manager: %v", err)
	}

	// 3. 创建客户端管理器（这里只是示例，实际使用时需要处理客户端连接）
	_ = services.NewClientManager(robotManager)

	// 4. 注册多个机器人
	robots := []*models.RobotRegistration{
		{
			UCode:        "robot_b2_001",
			Name:         "B2机器人1",
			Type:         models.RobotTypeB2,
			Version:      "1.0",
			IPAddress:    "192.168.1.100",
			Port:         8080,
			Capabilities: []string{"move", "stop", "reset"},
		},
		{
			UCode:        "robot_go2_001",
			Name:         "Go2机器人1",
			Type:         models.RobotTypeGo2,
			Version:      "1.0",
			IPAddress:    "192.168.1.101",
			Port:         8080,
			Capabilities: []string{"move", "stop", "reset", "dance"},
		},
	}

	for _, robot := range robots {
		if _, err := robotManager.RegisterRobot(robot); err != nil {
			log.Printf("Failed to register robot %s: %v", robot.UCode, err)
		}
	}

	// 5. 设置事件处理器
	robotManager.AddEventHandler(models.RobotEventConnected, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 连接成功\n", event.UCode)
	})

	robotManager.AddEventHandler(models.RobotEventDisconnected, func(event *models.RobotEvent) {
		fmt.Printf("机器人 %s 连接断开\n", event.UCode)
	})

	// 6. 获取机器人信息
	allRobots := robotManager.GetAllRobots()
	fmt.Printf("注册的机器人数量: %d\n", len(allRobots))

	onlineRobots := robotManager.GetOnlineRobots()
	fmt.Printf("在线机器人数量: %d\n", len(onlineRobots))

	// 7. 发送命令示例
	if len(onlineRobots) > 0 {
		command := &models.RobotCommand{
			Action:        "move",
			Params:        map[string]string{"direction": "forward", "speed": "0.5"},
			Priority:      5,
			Timestamp:     time.Now().Unix(),
			OperatorUCode: "system_operator",
		}

		response, err := robotManager.SendCommand(onlineRobots[0].UCode, command)
		if err != nil {
			fmt.Printf("命令发送失败: %v\n", err)
		} else {
			fmt.Printf("命令发送成功: %s\n", response.Message)
		}
	}

	// 8. 获取统计信息
	statistics := robotManager.GetRobotStatistics()
	fmt.Printf("机器人统计信息: %+v\n", statistics)

	systemStatus := robotManager.GetSystemStatus()
	fmt.Printf("系统状态: %+v\n", systemStatus)

	// 9. 保持运行一段时间
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	<-ctx.Done()
	fmt.Println("示例运行完成")
}
